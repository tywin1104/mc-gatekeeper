package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/tywin1104/mc-whitelist/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

func parseTimestamp(timestamp interface{}) (time.Time, error) {
	timestampStr := fmt.Sprintf("%v", timestamp)
	t, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
