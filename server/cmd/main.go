package main

import (
	"context"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-whitelist/broker"
	"github.com/tywin1104/mc-whitelist/db"
	"github.com/tywin1104/mc-whitelist/server"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	// Set up logrus logger
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to db
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))

	if err != nil {
		panic("Unable to connect to mongodb")
	}
	log.Info("Mongodb connection established")

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		panic("Unable to connect to rabbitmq")
	}
	log.Info("RabbitMQ connection established")
	defer conn.Close()
	broker, err := broker.NewService(conn, "whitelist.request.queue")
	if err != nil {
		panic("Unable to set up broker")
	}
	defer broker.Channel.Close()

	// Set up http REST API server
	dbSvc := db.NewService(client)
	httpServer := server.NewService(dbSvc, broker)
	httpServer.Listen(":8080")
}
