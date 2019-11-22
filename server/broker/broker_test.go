package broker_test

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-gatekeeper/broker"
)

var log = logrus.New()
var rabbitCloseError chan *amqp.Error
var testBroker *broker.Service

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
	rabbitCloseError = make(chan *amqp.Error)
	testBroker = broker.NewService(log, rabbitCloseError)
	go testBroker.WatchForReconnect()
	defer testBroker.Close()
	m.Run()
}

func TestBrokerReconnect(t *testing.T) {
	oldConn := testBroker.GetConn()
	oldChannel := testBroker.GetChannel()
	// establish the rabbitmq reconnection by sending
	// an error and thus calling the error callback
	rabbitCloseError <- amqp.ErrClosed
	// Short wait for the reconnection to be established
	time.Sleep(3 * time.Second)
	newConn := testBroker.GetConn()
	newChannel := testBroker.GetChannel()
	// Should expect the new connection/channel to be different from the old connection/channel
	if oldConn == newConn || oldChannel == newChannel {
		t.Error("RabbitMQ connection and channel do not change after reconnect")
	}
}
