package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"

	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-whitelist/broker"
	"github.com/tywin1104/mc-whitelist/db"
	"github.com/tywin1104/mc-whitelist/server"
	"github.com/tywin1104/mc-whitelist/types"
	"github.com/tywin1104/mc-whitelist/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var s *server.Service
var dbClient *mongo.Client

var newRequest1 *types.WhitelistRequest
var newRequest2 *types.WhitelistRequest
var newRequest3 *types.WhitelistRequest
var newRequest4 *types.WhitelistRequest
var newRequest5 *types.WhitelistRequest

var log = logrus.New()

func TestMain(m *testing.M) {
	// Mock the main application using the test configuration file
	viper.SetConfigName("config_test")
	viper.AddConfigPath("../")
	viper.AutomaticEnv()
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		log.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Fatal("Error reading config file")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(viper.GetString("mongodbConn")))
	if err != nil {
		log.Fatal(err)
	}
	dbClient = client
	client.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})

	dbSvc := db.NewService(client)

	broker := broker.NewService(log, make(chan *amqp.Error))
	defer broker.Close()
	serverLogger := log.WithField("origin", "server")
	s = server.NewService(dbSvc, broker, serverLogger)

	// Create mock db objects
	_id1, err := primitive.ObjectIDFromHex("5dc4dc43f7310f4c2a005673")
	if err != nil {
		log.Fatal(err)
	}
	newRequest1 = &types.WhitelistRequest{
		ID:        _id1,
		Username:  "user1",
		Email:     "user1@gmail.com",
		Age:       19,
		Gender:    "female",
		Status:    "Pending",
		Timestamp: time.Now(),
		Assignees: []string{"op1@gmail.com", "op2@gmail.com"},
	}
	newRequest2 = &types.WhitelistRequest{
		ID:        primitive.NewObjectID(),
		Username:  "user2",
		Email:     "user2@gmail.com",
		Age:       22,
		Gender:    "male",
		Status:    "Pending",
		Timestamp: time.Now(),
		Assignees: []string{"op3@gmail.com"},
	}

	_id3, err := primitive.ObjectIDFromHex("5dc4dc43f7310f4c2a005674")
	if err != nil {
		log.Fatal(err)
	}
	newRequest3 = &types.WhitelistRequest{
		ID:        _id3,
		Username:  "user3",
		Email:     "user3@gmail.com",
		Age:       39,
		Gender:    "female",
		Status:    "Pending",
		Timestamp: time.Now(),
	}

	_id4, err := primitive.ObjectIDFromHex("5dc4dc43f7310f4c2a005676")
	if err != nil {
		log.Fatal(err)
	}
	newRequest4 = &types.WhitelistRequest{
		ID:        _id4,
		Username:  "user4",
		Email:     "user4@gmail.com",
		Age:       29,
		Gender:    "male",
		Status:    "Denied",
		Timestamp: time.Now(),
	}

	newRequest5 = &types.WhitelistRequest{
		ID:        _id4,
		Username:  "user5",
		Email:     "user5@gmail.com",
		Age:       21,
		Gender:    "female",
		Status:    "Approved",
		Timestamp: time.Now(),
	}

	// Run all test cases
	m.Run()
}

func TestCreateRequest(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	var jsonStr = []byte(`{
		"info": {
		  "applicationText": "I'd like to join the server"
		},
		"username": "doggie",
		"email": "doggie@gmail.com",
		"age": 19,
		"gender": "female"
	  }`)

	req, err := http.NewRequest("POST", "/api/v1/requests/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandleCreateRequest())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusCreated)
	}
}

func TestCreateDupRequest(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest1)

	// Try to create request with same username again should fail
	var jsonStr = []byte(`{
		"info": {
		  "applicationText": "I'd like to join the server"
		},
		"username": "user1",
		"email": "user1@gmail.com",
		"age": 19,
		"gender": "female"
	  }`)

	req, err := http.NewRequest("POST", "/api/v1/requests/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandleCreateRequest())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusUnprocessableEntity {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusUnprocessableEntity)
	}
}

func TestCreateRequestWithAlreadyApproved(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest5)

	// Try to create request with same username again should fail
	var jsonStr = []byte(`{
		"info": {
		  "applicationText": "I'd like to join the server"
		},
		"username": "user5",
		"email": "user5fake@gmail.com",
		"age": 19,
		"gender": "female"
	  }`)

	req, err := http.NewRequest("POST", "/api/v1/requests/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandleCreateRequest())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusConflict {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusConflict)
	}
}

func TestCreateRequestAfterDenial(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest4)

	// Try to create request with same username again should fail
	var jsonStr = []byte(`{
		"info": {
		  "applicationText": "I'd like to join the server"
		},
		"username": "user4",
		"email": "user4fake@gmail.com",
		"age": 19,
		"gender": "female"
	  }`)

	req, err := http.NewRequest("POST", "/api/v1/requests/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandleCreateRequest())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusCreated)
	}
}

func TestGetRequestByIDExternal(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest3)
	//encoded ID for passphrase "passphrase"
	req, err := http.NewRequest("GET", "/api/v1/requests/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = mux.SetURLVars(req, map[string]string{
		"requestIdEncoded": "UkWw8mNTfvN7ToC7Mkov6_pInwF3KoF1PuB3LG2jQ2MnLk_dOdNNQ8ufDFhoGjANsT03HQ==",
	})
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandleGetRequestByID())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body to make sure what we get is the correct request as we set up for
	var response map[string]map[string]interface{}
	json.Unmarshal([]byte(rr.Body.String()), &response)
	if response["request"]["email"] != "user3@gmail.com" {
		t.Error("handler returned wrong request")
	}
}

func TestGetRequestByIDExternalFail(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest3)
	req, err := http.NewRequest("GET", "/api/v1/requests/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = mux.SetURLVars(req, map[string]string{
		// Wrong encoded ID token
		"requestIdEncoded": "UkWw8mNTfd2d2vN7ToC7Mkov6_pInwF3KoF1PuB3LG2jQ2MnLk_dOdNNQ8ufDFhoGjANsT03HQ==",
	})
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandleGetRequestByID())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestUpdateRequestByID(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest1)
	var jsonStr = []byte(`{"status": "Approved"}`)

	req, err := http.NewRequest("PATCH", "/api/v1/requests/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	req = mux.SetURLVars(req, map[string]string{
		// Encoded request ID for newReuqest1
		"requestIdEncoded": "MP4QqcxRRN7CIJYcmpO81XldXzY30aIvflB00D_Qh6E-TVkBab9ygcmaOortaa4WUwFMuw==",
	})
	q := req.URL.Query()
	// Encoded adm token for "op1@gmail.com", should work since op1 is part of the assigness for request1
	q.Add("adm", "Xt-mlteCyiQe7sSS0HnLUOGJSgIW0lpi_SkYz7sahK411cgi5ecE8uQ=")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandlePatchRequestByID())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	// Check the response body to make sure what we indeed updated the old request object
	var response map[string]map[string]interface{}
	json.Unmarshal([]byte(rr.Body.String()), &response)
	if response["updated"]["status"] != "Approved" {
		t.Error("handler returned wrong request")
	}
	// Also verify the correct op email got attached to the admin field
	if response["updated"]["admin"] != "op1@gmail.com" {
		t.Error("handler returned wrong request")
	}
}

func TestUpdateRequestByIDMatchingFailed(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest3)
	var jsonStr = []byte(`{"status": "Approved"}`)

	req, err := http.NewRequest("PATCH", "/api/v1/requests/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	req = mux.SetURLVars(req, map[string]string{
		"requestIdEncoded": "UkWw8mNTfvN7ToC7Mkov6_pInwF3KoF1PuB3LG2jQ2MnLk_dOdNNQ8ufDFhoGjANsT03HQ==",
	})
	q := req.URL.Query()
	// Encoded adm token for "op1@gmail.com", since newReuqest3 does not have corresponding assignee
	// Should fail
	q.Add("adm", "Xt-mlteCyiQe7sSS0HnLUOGJSgIW0lpi_SkYz7sahK411cgi5ecE8uQ=")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandlePatchRequestByID())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestUpdateRequestByIDWrongAdmToken(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest3)
	var jsonStr = []byte(`{"status": "Approved"}`)

	req, err := http.NewRequest("PATCH", "/api/v1/requests/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	req = mux.SetURLVars(req, map[string]string{
		"requestIdEncoded": "UkWw8mNTfvN7ToC7Mkov6_pInwF3KoF1PuB3LG2jQ2MnLk_dOdNNQ8ufDFhoGjANsT03HQ==",
	})
	q := req.URL.Query()
	// Wrong adm token for "op1@gmail.com"
	q.Add("adm", "Xt-mlteCyiQe7sSS0HnLUOGJIW0lpi_SkYz7sahK411cgi5ecE8uQ=")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandlePatchRequestByID())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestUpdateRequestByIDWrongEncodedID(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest3)
	var jsonStr = []byte(`{"status": "Approved"}`)

	req, err := http.NewRequest("PATCH", "/api/v1/requests/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	req = mux.SetURLVars(req, map[string]string{
		// Wrong
		"requestIdEncoded": "UkWw8mNTfvN7ToC7Mk6_pInwF3KoF1PuB3LG2jQ2MnLk_dOdNNQ8ufDFhoGjANsT03HQ==",
	})
	q := req.URL.Query()
	q.Add("adm", "Xt-mlteCyiQe7sSS0HnLUOGJSgIW0lpi_SkYz7sahK411cgi5ecE8uQ=")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandlePatchRequestByID())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestVerificationFail(t *testing.T) {
	// newRequest2 is assigned to op3
	// adm token for op1 + encoded id for newRequest1 should fail
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest2)
	var jsonStr = []byte(`{"status": "Approved"}`)

	req, err := http.NewRequest("PATCH", "/api/v1/verify/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	encodedID1, err := utils.EncodeAndEncrypt("5dc4dc43f7310f4c2a005673", "passphrase")
	if err != nil {
		t.Fatal(err)
	}
	req = mux.SetURLVars(req, map[string]string{
		"requestIdEncoded": encodedID1,
	})
	q := req.URL.Query()
	//adm token for op1
	q.Add("adm", "Xt-mlteCyiQe7sSS0HnLUOGJSgIW0lpi_SkYz7sahK411cgi5ecE8uQ=")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandleVerifyMatchingTokens())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestVerification(t *testing.T) {
	// newRequest1 is assigned to both op1 and op2
	// adm token for op1 + encoded id for newRequest1 should pass
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest1)
	var jsonStr = []byte(`{"status": "Approved"}`)

	req, err := http.NewRequest("PATCH", "/api/v1/verify/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	encodedID1, err := utils.EncodeAndEncrypt("5dc4dc43f7310f4c2a005673", "passphrase")
	if err != nil {
		t.Fatal(err)
	}
	req = mux.SetURLVars(req, map[string]string{
		"requestIdEncoded": encodedID1,
	})
	q := req.URL.Query()
	//adm token for op1
	q.Add("adm", "Xt-mlteCyiQe7sSS0HnLUOGJSgIW0lpi_SkYz7sahK411cgi5ecE8uQ=")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandleVerifyMatchingTokens())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
func TestAuth(t *testing.T) {
	var jsonStr = []byte(`{"username": "testadmin", "password": "testadminpassword"}`)
	req, err := http.NewRequest("POST", "/api/v1/auth/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandleAdminSignin())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestAuthWrongCredential(t *testing.T) {
	var jsonStr = []byte(`{"username": "admin", "password": "password"}`)
	req, err := http.NewRequest("POST", "/api/v1/auth/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandleAdminSignin())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusUnauthorized)
	}
}

func TestGetRequestsInternal(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest1)
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest2)
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest3)
	// Generate jwt token with admin login
	var jsonStr = []byte(`{"username": "testadmin", "password": "testadminpassword"}`)
	req, err := http.NewRequest("POST", "/api/v1/auth/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandleAdminSignin())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	var response map[string]map[string]interface{}
	json.Unmarshal([]byte(rr.Body.String()), &response)
	token := response["token"]["value"]
	// Use the newly obtained token to issue internal get requests
	req, err = http.NewRequest("GET", "/api/v1/internal/requests/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	//Set Authorization Bearer header
	tokenStr := fmt.Sprintf("%v", token)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr = httptest.NewRecorder()
	negroni.New(
		negroni.HandlerFunc(s.GetAuthMiddleware().HandlerWithNext),
		negroni.Wrap(s.HandleGetRequests()),
	).ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	// Check that we indeed get all three requests in thr db
	var requestsResponse map[string][]types.WhitelistRequest
	json.Unmarshal([]byte(rr.Body.String()), &requestsResponse)
	requests := requestsResponse["requests"]
	if len(requests) != 3 {
		t.Errorf("Expect to get 3 requests, but got %d", len(requests))
	}
}

func TestGetRequestsInternalUnauthorized(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest1)
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest2)
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest3)

	req, err := http.NewRequest("GET", "/api/v1/internal/requests/", nil)
	if err != nil {
		log.Fatal(err)
	}
	rr := httptest.NewRecorder()
	negroni.New(
		negroni.HandlerFunc(s.GetAuthMiddleware().HandlerWithNext),
		negroni.Wrap(s.HandleGetRequests()),
	).ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusUnauthorized)
	}
}

func TestGetRequestsInternalWrongToken(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest1)
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest2)
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest3)
	req, err := http.NewRequest("GET", "/api/v1/internal/requests/", nil)
	if err != nil {
		log.Fatal(err)
	}
	// Use invalid token
	req.Header.Set("Authorization", "Bearer "+"randomwrongtoken")
	rr := httptest.NewRecorder()
	negroni.New(
		negroni.HandlerFunc(s.GetAuthMiddleware().HandlerWithNext),
		negroni.Wrap(s.HandleGetRequests()),
	).ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusUnauthorized)
	}
}

func TestInternalUpdateRequest(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest1)
	// Generate jwt token with admin login
	var jsonStr = []byte(`{"username": "testadmin", "password": "testadminpassword"}`)
	req, err := http.NewRequest("POST", "/api/v1/auth/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandleAdminSignin())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	var response map[string]map[string]interface{}
	json.Unmarshal([]byte(rr.Body.String()), &response)
	token := response["token"]["value"]
	//Set Authorization Bearer header
	tokenStr := fmt.Sprintf("%v", token)
	rr2 := httptest.NewRecorder()
	jsonStr = []byte(`{"status": "Denied"}`)
	req2, err := http.NewRequest("PATCH", "/api/v1/internal/requests/", bytes.NewBuffer(jsonStr))
	req2 = mux.SetURLVars(req2, map[string]string{
		"requestId": "5dc4dc43f7310f4c2a005673",
	})
	if err != nil {
		t.Fatal(err)
	}
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+tokenStr)
	negroni.New(
		negroni.HandlerFunc(s.GetAuthMiddleware().HandlerWithNext),
		negroni.Wrap(s.HandleInternalPatchRequestByID()),
	).ServeHTTP(rr2, req2)
	if status := rr2.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	// Check the response body to make sure what we indeed updated the old request object
	var response2 map[string]map[string]interface{}
	json.Unmarshal([]byte(rr2.Body.String()), &response2)
	if response2["updated"]["status"] != "Denied" {
		t.Error("handler returned wrong request")
	}
	// Also verify the correct op email got attached to the admin field
	if response2["updated"]["admin"] != "admin" {
		t.Error("handler returned wrong request")
	}
}

func TestInternalUpdateRequestFail(t *testing.T) {
	dbClient.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})
	dbClient.Database("mc-whitelist").Collection("requests").InsertOne(context.TODO(), newRequest4)
	// Generate jwt token with admin login
	var jsonStr = []byte(`{"username": "testadmin", "password": "testadminpassword"}`)
	req, err := http.NewRequest("POST", "/api/v1/auth/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HandleAdminSignin())
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	var response map[string]map[string]interface{}
	json.Unmarshal([]byte(rr.Body.String()), &response)
	token := response["token"]["value"]
	//Set Authorization Bearer header
	tokenStr := fmt.Sprintf("%v", token)
	rr2 := httptest.NewRecorder()
	jsonStr = []byte(`{"status": "Denied"}`)
	req2, err := http.NewRequest("PATCH", "/api/v1/internal/requests/", bytes.NewBuffer(jsonStr))
	req2 = mux.SetURLVars(req2, map[string]string{
		"requestId": "5dc4dc43f7310f4c2a005676",
	})
	if err != nil {
		t.Fatal(err)
	}
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+tokenStr)
	negroni.New(
		negroni.HandlerFunc(s.GetAuthMiddleware().HandlerWithNext),
		negroni.Wrap(s.HandleInternalPatchRequestByID()),
	).ServeHTTP(rr2, req2)
	if status := rr2.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}
