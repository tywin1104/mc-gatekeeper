package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-whitelist/broker"
	"github.com/tywin1104/mc-whitelist/db"
	"github.com/tywin1104/mc-whitelist/server"
	"github.com/tywin1104/mc-whitelist/worker"
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
	log.WithFields(logrus.Fields{
		"addr": strings.Split(viper.GetString("mongodbConn"), "@")[1],
	}).Info("Mongodb connection established")

	dbSvc := db.NewService(client)

	broker := broker.NewService(log, make(chan *amqp.Error))
	// Watch for unexpected connection loss to rabbitMQ and re-establish connection
	go broker.WatchForReconnect()
	defer broker.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)
	// Start the worker
	workerLogger := log.WithField("origin", "worker")
	worker, err := worker.NewWorker(dbSvc, workerLogger)
	if err != nil {
		log.Fatal("Unable to start worker: " + err.Error())
	}
	go worker.Start(&wg)
	defer worker.Close()
	// Setup and start the http REST API server
	serverLogger := log.WithField("origin", "server")
	httpServer := server.NewService(dbSvc, broker, serverLogger)
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
