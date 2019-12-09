package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-gatekeeper/broker"
	"github.com/tywin1104/mc-gatekeeper/cache"
	"github.com/tywin1104/mc-gatekeeper/db"
	"github.com/tywin1104/mc-gatekeeper/server"
	"github.com/tywin1104/mc-gatekeeper/server/sse"
	"github.com/tywin1104/mc-gatekeeper/watcher"
	"github.com/tywin1104/mc-gatekeeper/worker"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var log = logrus.New()

func main() {
	// Set up logrus logger
	// log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.DebugLevel)

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		log.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Fatal("Error reading config file")
	}

	err := watcher.ValidateConfig()
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Fatal("Invalid configuration")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(viper.GetString("mongodbConn")))

	if err != nil {
		log.Fatal("Unable to connect to mongodb: " + err.Error())
	}
	log.Info("Mongodb connection established")

	// Setup database service
	dbSvc := db.NewService(client)

	// Initilize server side event server for pushing out stats
	serverLogger := log.WithField("origin", "server")
	sseServer := sse.NewServer(serverLogger)
	// Setup redis cache
	cache := cache.NewService(dbSvc, sseServer)
	err = cache.SyncStats()
	if err != nil {
		log.Fatal("Unable to sync cache values: " + err.Error())
	}
	// Start background job to collect aggregate stats at a interval
	go watcher.AggregateStats(cache, log)

	// Set it running - listening and broadcasting events
	go sseServer.Listen(cache.BroadcastStats)

	broker := broker.NewService(log, make(chan *amqp.Error))
	// Watch for unexpected connection loss to rabbitMQ and re-establish connection
	go broker.WatchForReconnect()
	defer broker.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)
	// Start the worker
	workerLogger := log.WithField("origin", "worker")
	worker, err := worker.NewWorker(dbSvc, cache, workerLogger, make(chan *amqp.Error))
	if err != nil {
		log.Fatal("Unable to start worker: " + err.Error())
	}
	go worker.Start(&wg)
	defer worker.Close()
	// Setup and start the http REST API server
	httpServer := server.NewService(dbSvc, broker, cache, sseServer, serverLogger)
	go httpServer.Listen(viper.GetString("port"), &wg)
	wg.Wait()

	// Watch for configuration changes
	go watcher.WatchConfig(log)
	log.Info("Everything is up.")
	<-make(chan int)
}
