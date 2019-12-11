package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/tywin1104/mc-gatekeeper/types"
	"github.com/tywin1104/mc-gatekeeper/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Update the request object's metadata and add corresponding task to broker
func (svc *Service) updateRequestByID(requestID string, reqBody []byte, admin string) (types.WhitelistRequest, int, error) {
	log := svc.logger
	var requestedChange bson.M
	json.Unmarshal(reqBody, &requestedChange)
	// Update the admin field to be the op'e email behind adm email token
	requestedChange["admin"] = admin
	// update timestamp metadata according to different type of status change
	if newStatus, ok := requestedChange["status"]; ok {
		if newStatus == types.StatusApproved || newStatus == types.StatusDenied {
			requestedChange["processedTimestamp"] = time.Now()
			requestedChange["lastUpdatedTimestamp"] = time.Now()
		} else if newStatus == types.StatusDeactivated || newStatus == types.StatusBanned {
			requestedChange["lastUpdatedTimestamp"] = time.Now()
		}
	}

	_id, _ := primitive.ObjectIDFromHex(requestID)
	updatedRequest, err := svc.dbService.UpdateRequest(bson.D{{"_id", _id}}, bson.M{
		"$set": requestedChange,
	})
	if err != nil {
		log.WithFields(logrus.Fields{
			"err":             err.Error(),
			"requestID":       requestID,
			"requestedChange": requestedChange,
		}).Error("Unable to update request")
		return types.WhitelistRequest{}, http.StatusInternalServerError, errors.New("Unable to update request")
	}
	// Add updated request to the broker for worker to process
	// convert bson.M to struct
	var updatedRequestObj types.WhitelistRequest
	bsonBytes, _ := bson.Marshal(updatedRequest)
	bson.Unmarshal(bsonBytes, &updatedRequestObj)

	// Publish the updatedRequestObj to broker
	err = svc.broker.Publish(updatedRequestObj)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error":             err.Error(),
			"updatedReqeustObj": updatedRequestObj,
		}).Error("Unable to publish message to broker")
		return types.WhitelistRequest{}, http.StatusInternalServerError, errors.New("Unable to update request")
	}
	return updatedRequestObj, http.StatusOK, nil
}

// Get request object from db by encrypted and url-encoded request ID
func (svc *Service) getRequestByEncryptedID(requestIDEncoded string) (types.WhitelistRequest, int, error) {
	log := svc.logger
	requestID, err := utils.DecodeAndDecrypt(requestIDEncoded, viper.GetString("passphrase"))
	if err != nil {
		log.WithFields(logrus.Fields{
			"err":      err.Error(),
			"urlParam": requestIDEncoded,
		}).Warn("Unable to decode requestID token")
		return types.WhitelistRequest{}, http.StatusBadRequest, errors.New("Unable to decode token")
	}

	_id, _ := primitive.ObjectIDFromHex(string(requestID))
	requests, err := svc.dbService.GetRequests(1, bson.D{{"_id", _id}})
	if err != nil {
		log.WithFields(logrus.Fields{
			"err":       err.Error(),
			"requestID": requestID,
		}).Error("Unable to get reqeuest by ID")
		return types.WhitelistRequest{}, http.StatusInternalServerError, errors.New("Unable to get reqeuest by ID")
	}
	if len(requests) == 0 {
		return types.WhitelistRequest{}, http.StatusBadRequest, errors.New("Resource not found")
	}
	request := requests[0]
	return request, http.StatusOK, nil
}
