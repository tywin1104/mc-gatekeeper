package server

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/tywin1104/mc-gatekeeper/types"
	"github.com/tywin1104/mc-gatekeeper/utils"
	"go.mongodb.org/mongo-driver/bson"
)

// HandleGetRequestByID get one request by encoded id
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
			"_id":       request.ID.Hex(),
			"gender":    request.Gender,
		}}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(msg)
	}
}

// HandleCreateRequest create new request
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
		newRequest.Status = types.StatusPending
		err = svc.broker.Publish(newRequest)
		if err != nil {
			http.Error(w, "Unable to create new request", http.StatusInternalServerError)
			log.WithFields(logrus.Fields{
				"error":      err.Error(),
				"newRequest": newRequest,
			}).Error("Unable to publish message to broker")
			return
		}

		w.WriteHeader(http.StatusCreated)
		msg := map[string]interface{}{"message": "success", "created": newRequestID}
		json.NewEncoder(w).Encode(msg)
	}
}

// HandlePatchRequestByID update the request by encrypted id
func (svc *Service) HandlePatchRequestByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get admin info from ?adm=<EncodedAdminEmail>
		keys, ok := r.URL.Query()["adm"]

		if !ok || len(keys[0]) < 1 {
			http.Error(w, "adm token is missing", http.StatusBadRequest)
			return
		}
		admToken := keys[0]
		// Only proceed if two tokens are matching correctly
		request, opEmail, err := svc.verifyMatchingTokens(mux.Vars(r)["requestIdEncoded"], admToken)
		if err != nil {
			http.Error(w, "Tokens do not match", http.StatusBadRequest)
			return
		}
		// Only update a request if its status is still pending
		if request.Status != types.StatusPending {
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
		"status":   bson.M{"$in": []string{types.StatusPending, types.StatusApproved, types.StatusBanned}},
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
		foundRequest := foundRequests[0]
		if foundRequest.Status == types.StatusApproved {
			message = "The request associated with this username is already approved"
			return http.StatusConflict, errors.New(message)
		} else if foundRequest.Status == types.StatusPending {
			message = "There is a pending request associated with this username. " +
				"You can not submit another request at this time. If you haven't received " +
				"result within 24 hours, please contact admin"
			return http.StatusUnprocessableEntity, errors.New(message)
		} else if foundRequest.Status == types.StatusBanned {
			message = "The user has been banned from the server"
			return http.StatusForbidden, errors.New(message)
		}
	}
	return http.StatusOK, nil
}

// HandleVerifyMatchingTokens verifys the correct matching pair between adm token and request ID token
// Mainly used for client application to verify first before rending information that is only supposed to
// be displayed to relevant admin. The update request logic will also double check the matching tokens
func (svc *Service) HandleVerifyMatchingTokens() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get admin info from ?adm=<EncodedAdminEmail>
		keys, ok := r.URL.Query()["adm"]

		if !ok || len(keys[0]) < 1 {
			http.Error(w, "adm token is missing", http.StatusBadRequest)
			return
		}
		admToken := keys[0]
		_, _, err := svc.verifyMatchingTokens(mux.Vars(r)["requestIdEncoded"], admToken)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// returns request object and corresponding op's email if tokens match
// return err if either token is invalid or two tokens does not match by assignee relation
func (svc *Service) verifyMatchingTokens(requestIDToken, admToken string) (types.WhitelistRequest, string, error) {
	log := svc.logger
	opEmail, err := utils.DecodeAndDecrypt(admToken, viper.GetString("passphrase"))
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Error("Unable to decode adm token")
		return types.WhitelistRequest{}, "", err
	}
	request, _, err := svc.getRequestByEncryptedID(requestIDToken)
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Error("Unable to get request by enceyptedID")
		return types.WhitelistRequest{}, "", err
	}
	for _, op := range request.Assignees {
		if opEmail == op {
			return request, opEmail, nil
		}
	}
	return types.WhitelistRequest{}, "", errors.New("Tokens do not match")
}
