package broker

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-whitelist/types"
)

type Service struct {
	conn    *amqp.Connection
	Channel *amqp.Channel
	queue   amqp.Queue
}

// NewService
func NewService(conn *amqp.Connection, queueName string) (*Service, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, errors.New("Failed to open a channel")
	}

	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
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

// Publish
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
