package broker

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-whitelist/types"
	try "gopkg.in/matryer/try.v1"
)

// Service represents broker(message queue) service
type Service struct {
	conn             *amqp.Connection
	channel          *amqp.Channel
	log              *logrus.Logger
	rabbitCloseError chan *amqp.Error
}

func (s *Service) GetConn() *amqp.Connection {
	return s.conn
}
func (s *Service) GetChannel() *amqp.Channel {
	return s.channel
}

//WatchForReconnect watch for unexpected connection loss to rabbitMQ and re-establish connection
func (s *Service) WatchForReconnect() {
	for {
		rabbitErr := <-s.rabbitCloseError
		if rabbitErr != nil {
			s.log.Warning("Broker connection with message queue closed unexpectedly. About to reconnect")
			s.rabbitCloseError = make(chan *amqp.Error)

			// Reconnect to message queue and establish a new channel
			// From then on, the newly created channel will be used to
			// do message publishing
			s.connectToRabbitMQ()
			s.conn.NotifyClose(s.rabbitCloseError)
			err := s.setup()
			if err != nil {
				s.log.WithFields(logrus.Fields{
					"err": err.Error(),
				}).Fatal("Unable to set up broker")
			}
		}
	}
}

// NewService set up all thing rabbitMq related
func NewService(log *logrus.Logger, rabbitCloseError chan *amqp.Error) *Service {
	s := new(Service)
	s.log = log
	s.rabbitCloseError = rabbitCloseError
	s.connectToRabbitMQ()
	err := s.setup()
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Fatal("Unable to set up broker")
	}
	s.conn.NotifyClose(rabbitCloseError)
	return s
}

// Close connection and channel associated with the broker
func (s *Service) Close() {
	s.channel.Close()
	s.conn.Close()
}

// Connect to message queue with retries, update conn field
func (s *Service) connectToRabbitMQ() {
	err := try.Do(func(attempt int) (bool, error) {
		if attempt > 1 {
			s.log.Infof("Trying to connect to RabbitMQ [%d/3]\n", attempt)
		}
		conn, e := amqp.Dial(viper.GetString("rabbitMQConn"))
		if e != nil {
			time.Sleep(5 * time.Second)
		} else {
			s.log.WithFields(logrus.Fields{
				"addr": strings.Split(viper.GetString("rabbitMQConn"), "@")[1],
			}).Info("Broker-message queue connection established")
			s.conn = conn
		}
		return attempt < 3, e
	})
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Fatal("Unable to connect to rabbitmq")
	}
}

// Setup queue declaration and update conn property
func (s *Service) setup() error {
	ch, err := s.conn.Channel()
	if err != nil {
		return errors.New("Failed to open a channel")
	}

	args := make(amqp.Table)
	// Dead letter exchange name
	args["x-dead-letter-exchange"] = "dead.letter.ex"
	// Default message ttl 24 hours
	args["x-message-ttl"] = int32(8.64e+7)

	// Declare the dead letter exchange
	err = ch.ExchangeDeclare(
		"dead.letter.ex", // name
		"fanout",         // type
		true,             // durable
		false,            // auto-deleted
		false,            // internal
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		return err
	}
	// Declare the dead letter queue
	_, err = ch.QueueDeclare(
		"dead.letter.queue", // name
		true,                // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return err
	}
	// Bind dead letter exchange to dead letter queue
	err = ch.QueueBind(
		"dead.letter.queue", // queue name
		"",                  // routing key
		"dead.letter.ex",    // exchange
		false,
		nil,
	)
	if err != nil {
		return err
	}

	_, err = ch.QueueDeclare(
		viper.GetString("taskQueueName"), // name
		true,                             // durable
		false,                            // delete when unused
		false,                            // exclusive
		false,                            // no-wait
		args,                             // arguments
	)
	if err != nil {
		return errors.New("Failed to declare the queue")
	}
	s.channel = ch
	return nil
}

// Publish a whitelistRequest message for the queue to consume
func (s *Service) Publish(message types.WhitelistRequest) error {
	encodedMessage, err := serialize(message)
	if err != nil {
		return err
	}

	err = try.Do(func(attempt int) (bool, error) {
		if attempt > 1 {
			s.log.Infof("Trying to publish message to broker [%d/3]\n", attempt)
		}
		e := s.channel.Publish(
			"",                               // exchange
			viper.GetString("taskQueueName"), // routing key
			false,                            // mandatory
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "application/json",
				Body:         []byte(encodedMessage),
			})
		return attempt < 3, e
	})
	return err
}

func serialize(msg types.WhitelistRequest) ([]byte, error) {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	err := encoder.Encode(msg)
	return b.Bytes(), err
}
