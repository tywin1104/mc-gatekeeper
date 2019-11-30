package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HandleGetRequests handle get requests from authenticated admin user
func (svc *Service) HandleGetRequests() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := svc.logger
		var msg map[string]interface{}
		// Try to fetch value from cache first
		cachedRequests, err := svc.cache.GetAllRequests()
		if err != nil {
			log.Debug("Got results from db")
			requests, err := svc.dbService.GetRequests(-1, bson.D{{}})
			if err != nil {
				http.Error(w, "Unable to get all requests", http.StatusInternalServerError)
				log.WithFields(logrus.Fields{
					"err": err.Error(),
				}).Error("Unable to get all requests")
				return
			}
			msg = map[string]interface{}{"requests": requests}
		} else {
			log.Debug("Got cached results")
			msg = map[string]interface{}{"requests": cachedRequests}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(msg)
	}
}

// HandleInternalPatchRequestByID handle patch request from authenticated admin user
func (svc *Service) HandleInternalPatchRequestByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := mux.Vars(r)["requestId"]
		_id, _ := primitive.ObjectIDFromHex(requestID)
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
			http.Error(w, "Unable to get request", http.StatusInternalServerError)
			return
		}
		if len(foundRequests) > 0 {
			updatedRequest, statusCode, err := svc.updateRequestByID(requestID, reqBody, "admin")
			if err != nil {
				http.Error(w, err.Error(), statusCode)
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

func parseTimestamp(timestamp interface{}) (time.Time, error) {
	timestampStr := fmt.Sprintf("%v", timestamp)
	t, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
