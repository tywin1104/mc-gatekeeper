package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/tywin1104/mc-gatekeeper/types"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"go.mongodb.org/mongo-driver/mongo"
)

// Service represents struct that deals with database level operations
type Service struct {
	db *mongo.Client
}

// NewService create new mongoDb service that handles database level operations
func NewService(db *mongo.Client) *Service {
	return &Service{
		db: db,
	}
}

// Ping checks for db connection
func (s *Service) Ping() {
	err := s.db.Ping(context.TODO(), readpref.Primary())
	if err != nil {
		fmt.Println("Unable to ping the db")
	} else {
		fmt.Println("Pong")
	}
}

// CreateRequest create new whitelistRequest
func (s *Service) CreateRequest(newRequest types.WhitelistRequest) (primitive.ObjectID, error) {
	collection := s.db.Database("mc-whitelist").Collection("requests")
	newRequest.ID = primitive.NewObjectID()
	// Set initial request status and attach timestamp
	newRequest.Timestamp = time.Now()
	newRequest.Status = "Pending"
	newRequest.OnserverStatus = "None"
	_, err := collection.InsertOne(context.TODO(), newRequest)
	if err != nil {
		return primitive.ObjectID{}, err
	}
	return newRequest.ID, nil
}

// GetRequests query for whitelistRequests in db
func (s *Service) GetRequests(limit int64, filter interface{}) ([]types.WhitelistRequest, error) {
	collection := s.db.Database("mc-whitelist").Collection("requests")
	cur, err := collection.Find(context.TODO(), filter, options.Find().SetSort(map[string]int{"timestamp": -1}))
	if err != nil {
		return nil, err
	}

	requests := make([]types.WhitelistRequest, 0)
	for cur.Next(context.TODO()) {
		var request types.WhitelistRequest
		err := cur.Decode(&request)
		if err != nil {
			return nil, err
		}

		requests = append(requests, request)
	}
	return requests, nil
}

// UpdateRequest perform partial update to the specified whitelistRequest in db
func (s *Service) UpdateRequest(filter, update interface{}) (bson.M, error) {
	collection := s.db.Database("mc-whitelist").Collection("requests")
	upsert := true
	after := options.After
	opt := options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
		Upsert:         &upsert,
	}

	result := collection.FindOneAndUpdate(context.TODO(), filter, update, &opt)
	if result.Err() != nil {
		return nil, result.Err()
	}
	updatedRequest := bson.M{}
	decodeErr := result.Decode(&updatedRequest)
	return updatedRequest, decodeErr
}
