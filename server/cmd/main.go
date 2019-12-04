package main

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-gatekeeper/broker"
	"github.com/tywin1104/mc-gatekeeper/cache"
	"github.com/tywin1104/mc-gatekeeper/db"
	"github.com/tywin1104/mc-gatekeeper/server"
	"github.com/tywin1104/mc-gatekeeper/server/sse"
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

	err := validateConfig()
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Fatal("Invalid configuration")
	}
	// Watch for configuration changes
	go watchConfig(log)

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
	go aggregatingStats(cache)

	// Set it running - listening and broadcasting events
	go sseServer.Listen(cache.BroadcastViaSSE)

	broker := broker.NewService(log, make(chan *amqp.Error))
	// Watch for unexpected connection loss to rabbitMQ and re-establish connection
	go broker.WatchForReconnect()
	defer broker.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)
	// Start the worker
	workerLogger := log.WithField("origin", "worker")
	worker1, err := worker.NewWorker(dbSvc, cache, workerLogger, make(chan *amqp.Error))
	if err != nil {
		log.Fatal("Unable to start worker: " + err.Error())
	}
	go worker1.Start(&wg)
	defer worker1.Close()
	// Setup and start the http REST API server
	httpServer := server.NewService(dbSvc, broker, cache, sseServer, serverLogger)
	go httpServer.Listen(viper.GetString("port"), &wg)
	wg.Wait()
	log.Info("Everything is up.")
	<-make(chan int)
}

func validateConfig() error {
	// TODO: add more constraints to fail fast
	strategy := viper.GetString("dispatchingStrategy")
	if strategy != "Broadcast" && strategy != "Random" {
		return errors.New("Invalid configuration. Allowed values for dispatchingStrategy: [Broadcast, Random]")
	}
	if strategy == "Random" && viper.GetInt("randomDispatchingThreshold") > len(viper.GetStringSlice("ops")) {
		return errors.New("Invalid configuration. Threshold value for random dispatching can not exceed total number of ops")
	}
	return nil
}

func watchConfig(log *logrus.Logger) {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		// Re-validate config each time it changes
		err := validateConfig()
		if err != nil {
			log.WithFields(logrus.Fields{
				"err": err.Error(),
			}).Error("Invalid configuration. The application will not not work properly.")
		}
		log.WithFields(logrus.Fields{
			"file": e.Name,
		}).Info("Config file changed:")
	})
}

func aggregatingStats(cache *cache.Service) {
	for range time.Tick(5 * time.Minute) {
		go func() {
			err := cache.UpdateAggregateStats()
			if err != nil {
				log.WithFields(logrus.Fields{
					"err": err.Error(),
				}).Error("Unable to aggregate stats")
			} else {
				log.Info("Aggregate stats data completed")
			}
		}()
	}
}
