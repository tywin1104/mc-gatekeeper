package server

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/tywin1104/mc-whitelist/types"
	"github.com/tywin1104/mc-whitelist/utils"
	"go.mongodb.org/mongo-driver/bson"
)

func (svc *Service) HandleGetRequestByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, statusCode, err := svc.getRequestByEncryptedID(mux.Vars(r)["requestIdEncoded"])
		if err != nil {
			http.Error(w, err.Error(), statusCode)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		// For external facing get request, only display non-sensitive necessary fields
		msg := map[string]map[string]interface{}{"request": {
			"username":  request.Username,
			"email":     request.Email,
			"status":    request.Status,
			"timestamp": request.Timestamp,
			"info":      request.Info,
			"age":       request.Age,
		}}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(msg)
	}
}

func (svc *Service) HandleCreateRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := svc.logger
		// Validate request body
		var newRequest types.WhitelistRequest
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(reqBody, &newRequest)
		if err != nil {
			http.Error(w, "Unable to unmarshal request body", http.StatusInternalServerError)
			return
		}

		// Validate new request
		statusCode, err := svc.validateCreateRequest(&newRequest)
		if err != nil {
			http.Error(w, err.Error(), statusCode)
			return
		}

		// Add to db
		newRequestID, err := svc.dbService.CreateRequest(newRequest)
		if err != nil {
			http.Error(w, "Unable to create new request", http.StatusInternalServerError)
			log.WithFields(logrus.Fields{
				"err":        err.Error(),
				"newRequest": newRequest,
			}).Error("Unable to create new request")
			return
		}
		// Add new whitelist request to the message queue for worker to process
		// Need to fill in the ID field as it is generated from the db side
		newRequest.ID = newRequestID
		// Set initial status to be pending
		newRequest.Status = "Pending"
		err = svc.broker.Publish(newRequest)
		if err != nil {
			http.Error(w, "Unable to create new request", http.StatusInternalServerError)
			log.WithFields(logrus.Fields{
				"error":      err.Error(),
				"newRequest": newRequest,
			}).Error("Unable to publish message to broker")
			return
		}

		w.WriteHeader(http.StatusOK)
		msg := map[string]interface{}{"message": "success", "created": newRequestID}
		json.NewEncoder(w).Encode(msg)
	}
}

func (svc *Service) HandlePatchRequestByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := svc.logger
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

		// Get the request from encoded id url path field
		request, statusCode, err := svc.getRequestByEncryptedID(mux.Vars(r)["requestIdEncoded"])
		if err != nil {
			http.Error(w, err.Error(), statusCode)
			return
		}
		// Only update a request if its status is still pending
		if request.Status != "Pending" {
			http.Error(w, "Request is already fulfilled", http.StatusBadRequest)
			return
		}
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", http.StatusBadRequest)
			return
		}
		// Update the request in db and add new task to broker
		updatedRequest, statusCode, err := svc.updateRequestByID(request.ID.Hex(), reqBody, opEmail)
		if err != nil {
			http.Error(w, err.Error(), statusCode)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		msg := map[string]interface{}{"message": "success", "updated": updatedRequest}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(msg)
	}
}

func (svc *Service) validateCreateRequest(newRequest *types.WhitelistRequest) (int, error) {
	// Prevent new request from a approved or pending username
	foundRequests, err := svc.dbService.GetRequests(-1, bson.M{
		"username": newRequest.Username,
		"status":   bson.M{"$in": []string{"Pending", "Approved"}},
	})
	if err != nil {
		svc.logger.WithFields(logrus.Fields{
			"error":      err.Error(),
			"newRequest": newRequest,
		}).Error("Unable to validate new request")
		return http.StatusInternalServerError, errors.New("Unable to validate new request")
	}
	if len(foundRequests) > 0 {
		var message string
		if foundRequests[0].Status == "Approved" {
			message = "The request associated with this username is already approved"
		} else {
			message = "There is a pending request associated with this username. " +
				"You can not submit another request at this time. If you haven't received " +
				"result within 24 hours, please contact admin"
		}
		return http.StatusBadRequest, errors.New(message)
	}
	return http.StatusOK, nil
}
