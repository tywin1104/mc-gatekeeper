package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-whitelist/broker"
	"github.com/tywin1104/mc-whitelist/config"
	"github.com/tywin1104/mc-whitelist/db"
	"github.com/tywin1104/mc-whitelist/server"
	"github.com/tywin1104/mc-whitelist/worker"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var log = logrus.New()

func main() {

	config, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Unable to load config: " + err.Error())
	}

	// Set up logrus logger
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongodbConnStr))

	if err != nil {
		log.Fatal("Unable to connect to mongodb: " + err.Error())
	}
	log.Info("Mongodb connection established")

	conn, err := amqp.Dial(config.RabbitmqConnStr)
	if err != nil {
		log.Fatal("Unable to connect to rabbitmq: " + err.Error())
	}
	log.Info("RabbitMQ connection established")

	defer conn.Close()
	broker, err := broker.NewService(conn, config.TaskQueueName)
	if err != nil {
		log.Fatal("Unable to setup broker: " + err.Error())
	}
	defer broker.Channel.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)
	// Start the worker
	go worker.Start(&wg)
	// Setup and start the http REST API server
	dbSvc := db.NewService(client)
	httpServer := server.NewService(dbSvc, broker, config, log)
	go httpServer.Listen(config.APIPort, &wg)
	wg.Wait()
	log.Info("Everything is up.")
	<-make(chan int)
}
