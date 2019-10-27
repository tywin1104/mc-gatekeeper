package broker

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-whitelist/types"
)

// Service represents broker(message queue) service
type Service struct {
	conn    *amqp.Connection
	Channel *amqp.Channel
	queue   amqp.Queue
}

// NewService set up all thing rabbitMq related
func NewService(conn *amqp.Connection, queueName string) (*Service, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, errors.New("Failed to open a channel")
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
		return nil, err
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
		return nil, err
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
		return nil, err
	}

	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		args,      // arguments
	)
	if err != nil {
		return nil, errors.New("Failed to declare the queue")
	}
	return &Service{
		conn:    conn,
		Channel: ch,
		queue:   q,
	}, nil
}

// Publish a whitelistRequest message for the queue to consume
func (s *Service) Publish(message types.WhitelistRequest) error {
	encodedMessage, err := serialize(message)
	if err != nil {
		return err
	}
	err = s.Channel.Publish(
		"",           // exchange
		s.queue.Name, // routing key
		false,        // mandatory
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         []byte(encodedMessage),
		})
	return err
}

func serialize(msg types.WhitelistRequest) ([]byte, error) {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	err := encoder.Encode(msg)
	return b.Bytes(), err
}
