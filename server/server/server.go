package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/tywin1104/mc-whitelist/utils"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/tywin1104/mc-whitelist/broker"
	"github.com/tywin1104/mc-whitelist/config"
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
	c         *config.Config
	logger    *logrus.Logger
}

// NewService create new mongoDb service that handles database level operations
func NewService(db *db.Service, broker *broker.Service, c *config.Config, logger *logrus.Logger) *Service {
	return &Service{
		dbService: db,
		router:    mux.NewRouter().StrictSlash(true),
		broker:    broker,
		c:         c,
		logger:    logger,
	}
}

// Listen opens up the http port for REST API and register all routes
func (svc *Service) Listen(port string) {
	log := svc.logger
	s := svc.router.PathPrefix("/api/v1/requests").Subrouter()
	s.HandleFunc("/", getRequestsHandler(svc.dbService, log)).Methods("GET")
	s.HandleFunc("/", createRequestHandler(svc.dbService, svc.broker, log)).Methods("POST")
	s.HandleFunc("/{requestIdEncoded}", getRequestByIDHandler(svc.dbService, svc.c.PassPhrase, log)).Methods("GET")
	s.HandleFunc("/{requestIdEncoded}", patchRequestHandler(svc.dbService, svc.broker, svc.c.PassPhrase, log)).Methods("PATCH").Queries("adm", "{adm}")
	s.HandleFunc("/{requestId}", deleteRequestHandler(svc.dbService, log)).Methods("DELETE")

	// Endpoint to verify validity of admin token for frontend to consume
	r := svc.router.PathPrefix("/api/v1/verify").Subrouter()
	r.HandleFunc("/", verifyAdminTokenHandler(svc.c.PassPhrase, svc.c.Ops)).Methods("GET").Queries("adm", "{adm}")
	log.WithFields(logrus.Fields{
		"port": port,
	}).Info("The API http server starts listening")
	// Configure CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PATCH"},
	})
	handler := c.Handler(svc.router)
	// log.Error(http.ListenAndServe(port, svc.router))
	log.Error(http.ListenAndServe(port, handler))
}

func getRequestsHandler(dbService *db.Service, log *logrus.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requests, err := dbService.GetRequests(-1, bson.D{{}})
		if err != nil {
			http.Error(w, "Unable to get all requests", http.StatusInternalServerError)
			log.WithFields(logrus.Fields{
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

func getRequestByIDHandler(dbService *db.Service, passphrase string, log *logrus.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID, err := utils.DecodeAndDecrypt(mux.Vars(r)["requestIdEncoded"], passphrase)
		if err != nil {
			log.WithFields(logrus.Fields{
				"err":      err.Error(),
				"urlParam": mux.Vars(r)["requestIdEncoded"],
			}).Error("Unable to decode requestID token")
			http.Error(w, "Unable to decode token", http.StatusBadRequest)
			return
		}

		_id, _ := primitive.ObjectIDFromHex(string(requestID))
		requests, err := dbService.GetRequests(1, bson.D{{"_id", _id}})
		if err != nil {
			http.Error(w, "Unable to get reqeuest by ID", http.StatusInternalServerError)
			log.WithFields(logrus.Fields{
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

func createRequestHandler(dbService *db.Service, broker *broker.Service, log *logrus.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validating request body
		var newRequest types.WhitelistRequest
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", http.StatusInternalServerError)
			return
		}
		err = json.Unmarshal(reqBody, &newRequest)
		if err != nil {
			http.Error(w, "Unable to unmarshal request body", http.StatusInternalServerError)
			return
		}

		// Prevent new request from a approved or pending username
		foundRequests, err := dbService.GetRequests(-1, bson.M{
			"username": newRequest.Username,
			"status":   bson.M{"$in": []string{"Pending", "Approved"}},
		})
		if err != nil {
			http.Error(w, "Unable to check user's record", http.StatusInternalServerError)
			return
		}
		if len(foundRequests) > 0 {
			w.WriteHeader(http.StatusBadRequest)
			var message string
			if foundRequests[0].Status == "Approved" {
				message = "The request associated with this username is already approved"
			} else {
				message = "There is a pending request associated with this username. " +
					"You can not submit another request at this time. If you haven't received " +
					"result within 24 hours, please contact admin"
			}
			msg := map[string]interface{}{"message": message}
			json.NewEncoder(w).Encode(msg)
			return
		}
		// Add corresponding whitelist request to the database
		newRequestID, err := dbService.CreateRequest(newRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.WithFields(logrus.Fields{
				"err": err.Error(),
			}).Error("Unable to create new request")
			return
		}
		// Add corresponding whitelist request to the queue for worker to fetch and handle
		// Need to append its object ID that is being generated by DB service here
		newRequest.ID = newRequestID
		err = broker.Publish(newRequest)
		if err != nil {
			log.WithFields(logrus.Fields{
				"error":      err,
				"newRequest": newRequest,
			}).Error("Unable to publish message to broker")
		}

		w.WriteHeader(http.StatusOK)
		msg := map[string]interface{}{"message": "success", "created": newRequestID}
		json.NewEncoder(w).Encode(msg)
	}
}

func patchRequestHandler(dbService *db.Service, broker *broker.Service, passphrase string, log *logrus.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID, err := utils.DecodeAndDecrypt(mux.Vars(r)["requestIdEncoded"], passphrase)
		if err != nil {
			log.WithFields(logrus.Fields{
				"err":      err.Error(),
				"urlParam": mux.Vars(r)["requestIdEncoded"],
			}).Error("Unable to decode requestID token")
			http.Error(w, "Unable to decode token", http.StatusBadRequest)
			return
		}
		// Get admin info from ?adm=<EncodedAdminEmail>
		keys, ok := r.URL.Query()["adm"]

		if !ok || len(keys[0]) < 1 {
			http.Error(w, "adm token is missing", http.StatusBadRequest)
			return
		}
		admToken := keys[0]
		opEmail, err := utils.DecodeAndDecrypt(admToken, passphrase)
		if err != nil {
			log.WithFields(logrus.Fields{
				"err":      err.Error(),
				"urlParam": mux.Vars(r)["requestIdEncoded"],
			}).Error("Unable to decode adm token")
			http.Error(w, "Unable to decode token", http.StatusBadRequest)
			return
		}

		_id, _ := primitive.ObjectIDFromHex(requestID)
		var requestedChange bson.M
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", http.StatusBadRequest)
			return
		}
		// Only update a request if its status is still pending
		foundRequests, err := dbService.GetRequests(-1, bson.M{
			"_id":    _id,
			"status": "Pending",
		})
		if err != nil {
			http.Error(w, "Unable to check user's record", http.StatusInternalServerError)
			return
		}
		if len(foundRequests) > 0 {
			json.Unmarshal(reqBody, &requestedChange)
			// Going to update the admin field of the request to whoever perform the action through UI
			requestedChange["admin"] = opEmail
			updatedRequest, err := dbService.UpdateRequest(bson.D{{"_id", _id}}, bson.M{
				"$set": requestedChange,
			})
			if err != nil {
				http.Error(w, "Unable to update request", http.StatusInternalServerError)
				log.WithFields(logrus.Fields{
					"err":             err.Error(),
					"requestID":       requestID,
					"requestedChange": requestedChange,
				}).Error("Unable to update request")
				return
			}
			// Add task to broker so that user will receive status update email
			// convert bson.M to struct
			var updatedRequestObj types.WhitelistRequest
			bsonBytes, _ := bson.Marshal(updatedRequest)
			bson.Unmarshal(bsonBytes, &updatedRequestObj)

			// Publish the updatedRequestObj to broker
			err = broker.Publish(updatedRequestObj)
			if err != nil {
				log.WithFields(logrus.Fields{
					"error":             err,
					"updatedReqeustObj": updatedRequestObj,
				}).Error("Unable to publish message to broker")
			}
			w.Header().Set("Content-Type", "application/json")
			msg := map[string]interface{}{"message": "success", "updated": updatedRequestObj}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(msg)
		} else {
			msg := map[string]interface{}{"message": "Invalid requestId or the target request is already closed"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(msg)
		}
	}
}

func deleteRequestHandler(dbService *db.Service, log *logrus.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := mux.Vars(r)["requestId"]
		_id, _ := primitive.ObjectIDFromHex(requestID)
		deleteCount, err := dbService.DeleteRequest(bson.D{{"_id", _id}})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.WithFields(logrus.Fields{
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

func verifyAdminTokenHandler(passphrase string, ops []string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get admin info from ?adm=<EncodedAdminEmail>
		keys, ok := r.URL.Query()["adm"]

		if !ok || len(keys[0]) < 1 {
			http.Error(w, "adm token is missing", http.StatusBadRequest)
			return
		}
		admToken := keys[0]
		admEmail, err := utils.DecodeAndDecrypt(admToken, passphrase)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		valid := false
		for _, op := range ops {
			if admEmail == op {
				valid = true
			}
		}
		if !valid {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
