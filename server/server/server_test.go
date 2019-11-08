package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-whitelist/broker"
	"github.com/tywin1104/mc-whitelist/config"
	"github.com/tywin1104/mc-whitelist/db"
	"github.com/tywin1104/mc-whitelist/server"
	"github.com/tywin1104/mc-whitelist/types"
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

func TestMain(m *testing.M) {
	// Mock the whole application
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatal()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongodbConnStr))
	if err != nil {
		log.Fatal(err)
	}
	dbClient = client
	client.Database("mc-whitelist").Collection("requests").DeleteMany(context.TODO(), bson.M{})

	dbSvc := db.NewService(client)
	conn, err := amqp.Dial(config.RabbitmqConnStr)
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()
	broker, err := broker.NewService(conn, config.TaskQueueName)
	if err != nil {
		log.Fatal("Unable to setup broker: " + err.Error())
	}
	defer broker.Channel.Close()
	serverLogger := log.WithField("origin", "server")
	s = server.NewService(dbSvc, broker, config, serverLogger)

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
	}
	newRequest2 = &types.WhitelistRequest{
		ID:        primitive.NewObjectID(),
		Username:  "user2",
		Email:     "user2@gmail.com",
		Age:       22,
		Gender:    "male",
		Status:    "Pending",
		Timestamp: time.Now(),
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
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
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
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
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
	// Encoded adm token for "op1@gmail.com"
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
