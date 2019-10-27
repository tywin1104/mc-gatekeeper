package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/tywin1104/mc-whitelist/utils"

	"github.com/felixge/httpsnoop"
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
	svc.routes()
	log.WithFields(logrus.Fields{
		"port": port,
	}).Info("The API http server starts listening")

	// Configure CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PATCH"},
	})

	// Listen and serve
	handler := c.Handler(svc.router)

	// capture http related metrics
	wrappedH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := httpsnoop.CaptureMetrics(handler, w, r)
		svc.logger.Infof("%s %s (code=%d dt=%s)",
			r.Method,
			r.URL,
			m.Code,
			m.Duration,
		)
	})
	log.Fatal(http.ListenAndServe(port, wrappedH))
}

func (svc *Service) routes() {
	// Endpoints that are public accessible
	s := svc.router.PathPrefix("/api/v1/requests").Subrouter()
	s.HandleFunc("/", svc.handleCreateRequest()).Methods("POST")
	s.HandleFunc("/{requestIdEncoded}", svc.handleGetRequestByID()).Methods("GET")
	s.HandleFunc("/{requestIdEncoded}", svc.handlePatchRequestByID()).Methods("PATCH").Queries("adm", "{adm}")

	// Endpoint to verify validity of admin token for frontend to consume
	r := svc.router.PathPrefix("/api/v1/verify").Subrouter()
	r.HandleFunc("/", svc.handleVerifyAdminToken()).Methods("GET").Queries("adm", "{adm}")

	// Endpoints for internal consumptiono only
	internal := svc.router.PathPrefix("/api/v1/internal/requests").Subrouter()
	internal.HandleFunc("/", svc.handleGetRequests()).Methods("GET")
	internal.HandleFunc("/{requestId}", svc.handleInternalPatchRequestByID()).Methods("PATCH")
	internal.HandleFunc("/{requestId}", svc.handleDeleteRequestByID()).Methods("DELETE")

}

func (svc *Service) handleGetRequests() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := svc.logger
		requests, err := svc.dbService.GetRequests(-1, bson.D{{}})
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

func (svc *Service) handleCreateRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := svc.logger
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
		foundRequests, err := svc.dbService.GetRequests(-1, bson.M{
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
		newRequestID, err := svc.dbService.CreateRequest(newRequest)
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
		newRequest.Status = "Pending"
		err = svc.broker.Publish(newRequest)
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

func (svc *Service) handleGetRequestByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := svc.logger
		requestID, err := utils.DecodeAndDecrypt(mux.Vars(r)["requestIdEncoded"], svc.c.PassPhrase)
		if err != nil {
			log.WithFields(logrus.Fields{
				"err":      err.Error(),
				"urlParam": mux.Vars(r)["requestIdEncoded"],
			}).Error("Unable to decode requestID token")
			http.Error(w, "Unable to decode token", http.StatusBadRequest)
			return
		}

		_id, _ := primitive.ObjectIDFromHex(string(requestID))
		requests, err := svc.dbService.GetRequests(1, bson.D{{"_id", _id}})
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
		request := requests[0]
		// For external facing get request, only display non-sensitive necessary fields
		msg := map[string]map[string]interface{}{"request": {
			"username":  request.Username,
			"email":     request.Email,
			"status":    request.Status,
			"timestamp": request.Timestamp,
			"info":      request.Info,
		}}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(msg)
	}
}

func (svc *Service) handlePatchRequestByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := svc.logger
		requestID, err := utils.DecodeAndDecrypt(mux.Vars(r)["requestIdEncoded"], svc.c.PassPhrase)
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
		opEmail, err := utils.DecodeAndDecrypt(admToken, svc.c.PassPhrase)
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
		foundRequests, err := svc.dbService.GetRequests(-1, bson.M{
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
			// Parse timestamp string into time type
			processedTimestamp, err := parseTimestamp(requestedChange["processedTimestamp"])
			requestedChange["processedTimestamp"] = processedTimestamp
			updatedRequest, err := svc.dbService.UpdateRequest(bson.D{{"_id", _id}}, bson.M{
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
			err = svc.broker.Publish(updatedRequestObj)
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
func (svc *Service) handleDeleteRequestByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := svc.logger
		requestID := mux.Vars(r)["requestId"]
		_id, _ := primitive.ObjectIDFromHex(requestID)
		deleteCount, err := svc.dbService.DeleteRequest(bson.D{{"_id", _id}})
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

func (svc *Service) handleVerifyAdminToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get admin info from ?adm=<EncodedAdminEmail>
		keys, ok := r.URL.Query()["adm"]

		if !ok || len(keys[0]) < 1 {
			http.Error(w, "adm token is missing", http.StatusBadRequest)
			return
		}
		admToken := keys[0]
		admEmail, err := utils.DecodeAndDecrypt(admToken, svc.c.PassPhrase)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		valid := false
		for _, op := range svc.c.Ops {
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

func (svc *Service) handleInternalPatchRequestByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := svc.logger
		requestID := mux.Vars(r)["requestId"]
		_id, _ := primitive.ObjectIDFromHex(requestID)
		var requestedChange bson.M
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", http.StatusBadRequest)
			return
		}
		// Only update a request if its status is still pending
		foundRequests, err := svc.dbService.GetRequests(-1, bson.M{
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
			requestedChange["admin"] = "admin"

			processedTimestamp, err := parseTimestamp(requestedChange["processedTimestamp"])
			fmt.Println(requestedChange["processedTimestamp"])
			requestedChange["processedTimestamp"] = processedTimestamp
			updatedRequest, err := svc.dbService.UpdateRequest(bson.D{{"_id", _id}}, bson.M{
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
			err = svc.broker.Publish(updatedRequestObj)
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

func parseTimestamp(timestamp interface{}) (time.Time, error) {
	timestampStr := fmt.Sprintf("%v", timestamp)
	t, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
