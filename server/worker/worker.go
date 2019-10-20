package main

import (
	"bytes"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-whitelist/mailer"
	"github.com/tywin1104/mc-whitelist/types"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error(msg)
	}
}

var (
	// TODO: Read from config
	ops = []string{"tiaven1104@gmail.com"}
)

func main() {

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	args := make(amqp.Table)
	// Dead letter exchange name
	args["x-dead-letter-exchange"] = "dead.letter.ex"
	// Default message ttl 24 hours
	args["x-message-ttl"] = int32(86400)

	q, err := ch.QueueDeclare(
		"whitelist.request.queue", // name
		true,                      // durable
		false,                     // delete when unused
		false,                     // exclusive
		false,                     // no-wait
		args,                      // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		args,   // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			whitelistRequest, err := deserialize(d.Body)
			if err != nil {
				log.WithFields(log.Fields{
					"messageBody": d.Body,
				}).Error("Unable to decode message into whitelistRequest")
				// Unable to process this message, put to the dead-letter queue
				d.Nack(false, false)
			} else {
				log.WithFields(log.Fields{
					"body": whitelistRequest,
				}).Info("Received a new message")
				// Concrete actions to do when receiving task from message queue
				// Send application confirmation email to user
				emailConfirmation(whitelistRequest)
				// Send approval request emails to op(s)
				emailToOps(whitelistRequest)

				d.Ack(false)
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func emailConfirmation(whitelistRequest types.WhitelistRequest) {
	subject := "Your request to join the server has been received"
	err := mailer.Send("../mailer/templates/confirmation.html", map[string]string{"link": "www.checkstatus.com/" + whitelistRequest.Username}, subject, whitelistRequest.Email)
	if err != nil {
		log.WithFields(log.Fields{
			"recipent": whitelistRequest.Email,
			"err":      err,
		}).Error("Failed to send confirmation email")
	} else {
		log.WithFields(log.Fields{
			"recipent": whitelistRequest.Email,
		}).Info("Confirmation email sent")
	}
}

func emailToOps(whitelistRequest types.WhitelistRequest) {
	subject := "[Action Required] Whitelist request from " + whitelistRequest.Username
	for _, op := range ops {
		err := mailer.Send("../mailer/templates/ops.html", map[string]string{"link": "www.approvewhitelist.com/" + op}, subject, op)
		if err != nil {
			log.WithFields(log.Fields{
				"recipent": op,
				"err":      err,
			}).Error("Failed to send email to op")
		} else {
			log.WithFields(log.Fields{
				"recipent": op,
			}).Info("Action email sent to op")
		}
	}
}
func deserialize(b []byte) (types.WhitelistRequest, error) {
	var msg types.WhitelistRequest
	buf := bytes.NewBuffer(b)
	decoder := json.NewDecoder(buf)
	err := decoder.Decode(&msg)
	return msg, err
}
