package worker_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-gatekeeper/cache"
	"github.com/tywin1104/mc-gatekeeper/db"
	"github.com/tywin1104/mc-gatekeeper/server/sse"
	"github.com/tywin1104/mc-gatekeeper/worker"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var log = logrus.New()
var rabbitCloseError chan *amqp.Error
var testWorker *worker.Worker

func TestMain(m *testing.M) {
	// Mock the main application using the test configuration file
	viper.SetConfigName("config_test")
	viper.AddConfigPath("../")
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		log.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Fatal("Error reading config file")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(viper.GetString("mongodbConn")))

	if err != nil {
		log.Fatal("Unable to connect to mongodb: " + err.Error())
	}
	dbSvc := db.NewService(client)
	serverLogger := log.WithField("origin", "server")
	sseServer := sse.NewServer(serverLogger)
	cache := cache.NewService(dbSvc, sseServer)
	workerLogger := log.WithField("origin", "worker")
	rabbitCloseError = make(chan *amqp.Error)
	testWorker, err = worker.NewWorker(dbSvc, cache, workerLogger, rabbitCloseError)
	if err != nil {
		log.Fatal("Unable to start worker: " + err.Error())
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go testWorker.Start(&wg)
	wg.Wait()
	defer testWorker.Close()
	m.Run()
}

func TestWorkerReconnect(t *testing.T) {
	oldConn := testWorker.GetConn()

	oldChannel := testWorker.GetChannel()
	// establish the rabbitmq reconnection by sending
	// an error and thus calling the error callback
	rabbitCloseError <- amqp.ErrClosed
	// Short wait for the reconnection to be established
	time.Sleep(3 * time.Second)
	newConn := testWorker.GetConn()
	newChannel := testWorker.GetChannel()
	// Should expect the new connection/channel to be different from the old connection/channel
	if oldConn == newConn || oldChannel == newChannel {
		t.Error("RabbitMQ connection and channel do not change after reconnect")
	}
}
