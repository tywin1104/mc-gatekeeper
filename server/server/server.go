package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/tywin1104/mc-whitelist/broker"
	"github.com/tywin1104/mc-whitelist/db"
	"github.com/tywin1104/mc-whitelist/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Service represents struct that deals with database level operations
type Service struct {
	dbService *db.Service
	router    *mux.Router
	broker    *broker.Service
}

// NewService create new mongoDb service that handles database level operations
func NewService(db *db.Service, broker *broker.Service) *Service {
	return &Service{
		dbService: db,
		router:    mux.NewRouter().StrictSlash(true),
		broker:    broker,
	}
}

// Listen opens up the http port for REST API and register all routes
func (svc *Service) Listen(port string) {
	s := svc.router.PathPrefix("/api/v1/requests").Subrouter()
	s.HandleFunc("/", getRequestsHandler(svc.dbService)).Methods("GET")
	s.HandleFunc("/", createRequestHandler(svc.dbService, svc.broker)).Methods("POST")
	s.HandleFunc("/{requestId}", getRequestByIDHandler(svc.dbService)).Methods("GET")
	s.HandleFunc("/{requestId}", patchRequestHandler(svc.dbService)).Methods("PATCH")
	s.HandleFunc("/{requestId}", deleteRequestHandler(svc.dbService)).Methods("DELETE")
	log.WithFields(log.Fields{
		"port": port,
	}).Info("The API http server starts listening")
	log.Error(http.ListenAndServe(port, svc.router))
}

func getRequestsHandler(dbService *db.Service) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requests, err := dbService.GetRequests(-1, bson.D{{}})
		if err != nil {
			http.Error(w, "Unable to get all requests", http.StatusInternalServerError)
			log.WithFields(log.Fields{
				"err": err.Error(),
			}).Error("Unable to get all requests")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		msg := map[string]interface{}{"requests": requests}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(msg)
	}
}

func getRequestByIDHandler(dbService *db.Service) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := mux.Vars(r)["requestId"]
		_id, _ := primitive.ObjectIDFromHex(requestID)
		requests, err := dbService.GetRequests(1, bson.D{{"_id", _id}})
		if err != nil {
			http.Error(w, "Unable to get reqeuest by ID", http.StatusInternalServerError)
			log.WithFields(log.Fields{
				"err":       err.Error(),
				"requestID": requestID,
			}).Error("Unable to get reqeuest by ID")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if len(requests) == 0 {
			http.Error(w, "Resource not found", http.StatusBadRequest)
			return
		}
		msg := map[string]interface{}{"request": requests[0]}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(msg)
	}
}

func createRequestHandler(dbService *db.Service, broker *broker.Service) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validating request body
		var newRequest types.WhitelistRequest
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(reqBody, &newRequest)
		if err != nil {
			http.Error(w, "Unable to unmarshal request body", http.StatusBadRequest)
			return
		}

		// Check if same user has pending request within 24 hours
		foundRequests, err := dbService.GetRequests(-1, bson.M{
			"username": newRequest.Username,
			"timestamp": bson.M{
				"$gt": time.Now().Add(-24 * time.Hour),
			},
		})
		if err != nil {
			http.Error(w, "Unable to check user's record", http.StatusInternalServerError)
			return
		}
		if len(foundRequests) > 0 {
			w.WriteHeader(http.StatusBadRequest)
			msg := map[string]interface{}{"message": "Target user has pending request. Try later"}
			json.NewEncoder(w).Encode(msg)
			return
		}
		// Add corresponding whitelist request to the database
		newRequestID, err := dbService.CreateRequest(newRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.WithFields(log.Fields{
				"err": err.Error(),
			}).Error("Unable to create new request")
			return
		}
		// Add corresponding whitelist request to the queue for worker to fetch and handle
		err = broker.Publish(newRequest)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"newRequest": newRequest,
			}).Error("Unable to publish message to broker")
		}

		w.WriteHeader(http.StatusOK)
		msg := map[string]interface{}{"message": "success", "created": newRequestID}
		json.NewEncoder(w).Encode(msg)
	}
}

func patchRequestHandler(dbService *db.Service) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := mux.Vars(r)["requestId"]
		_id, _ := primitive.ObjectIDFromHex(requestID)
		var requestedChange bson.M
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", http.StatusBadRequest)
			return
		}
		// Only update a request if its status is still pending
		foundRequests, err := dbService.GetRequests(-1, bson.M{
			"_id":    requestID,
			"status": "Pending",
		})
		if err != nil {
			http.Error(w, "Unable to check user's record", http.StatusInternalServerError)
			return
		}
		if len(foundRequests) > 0 {
			json.Unmarshal(reqBody, &requestedChange)
			updatedRequest, err := dbService.UpdateRequest(bson.D{{"_id", _id}}, bson.M{
				"$set": requestedChange,
			})
			if err != nil {
				http.Error(w, "Unable to update request", http.StatusInternalServerError)
				log.WithFields(log.Fields{
					"err":       err.Error(),
					"requestID": requestID,
				}).Error("Unable to update request")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			msg := map[string]interface{}{"message": "success", "updated": updatedRequest}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(msg)
		} else {
			msg := map[string]interface{}{"message": "Invalid requestId or the target request is already closed"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(msg)
		}
	}
}

func deleteRequestHandler(dbService *db.Service) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := mux.Vars(r)["requestId"]
		_id, _ := primitive.ObjectIDFromHex(requestID)
		deleteCount, err := dbService.DeleteRequest(bson.D{{"_id", _id}})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.WithFields(log.Fields{
				"err":       err.Error(),
				"requestID": requestID,
			}).Error("Unable to delete request")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		msg := map[string]interface{}{"message": "success", "deleteCount": deleteCount}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(msg)
	}
}
