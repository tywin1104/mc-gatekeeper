package worker

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-whitelist/config"
	"github.com/tywin1104/mc-whitelist/mailer"
	"github.com/tywin1104/mc-whitelist/types"
	"github.com/tywin1104/mc-whitelist/utils"
)

var log = logrus.New()

func failOnError(err error, msg string) {
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Error(msg)
	}
}

// Start the worker to process the messages pushed into the queue
func Start() {
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Unable to load config: " + err.Error())
	}

	// Setup logger
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)
	file, err := os.OpenFile(config.WorkerLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.Out = file
	} else {
		log.Info("Failed to log to file, using default stderr")
	}

	ops := config.Ops
	conn, err := amqp.Dial(config.RabbitmqConnStr)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	args := make(amqp.Table)
	// Dead letter exchange name
	args["x-dead-letter-exchange"] = "dead.letter.ex"
	// Default message ttl 24 hours
	args["x-message-ttl"] = int32(8.64e+7)

	q, err := ch.QueueDeclare(
		config.TaskQueueName, // name
		true,                 // durable
		false,                // delete when unused
		false,                // exclusive
		false,                // no-wait
		args,                 // arguments
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
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			whitelistRequest, err := deserialize(d.Body)
			if err != nil {
				log.WithFields(logrus.Fields{
					"messageBody": d.Body,
					"err":         err,
				}).Error("Unable to decode message into whitelistRequest")
				// Unable to process this message, put to the dead-letter queue
				d.Nack(false, false)
			} else {
				log.WithFields(logrus.Fields{
					"username": whitelistRequest.Username,
					"status":   whitelistRequest.Status,
					"ID":       whitelistRequest.ID,
				}).Info("Received new task")
				// Concrete actions to do when receiving task from message queue
				// From the message body to determine which type of work to do
				if whitelistRequest.Status == "Approved" || whitelistRequest.Status == "Denied" {
					// Need to send update status back to the user
					// Put message to dead letter queue for later investigation if unable to send decision email
					err := emailDecision(whitelistRequest, config)
					if err != nil {
						log.WithFields(logrus.Fields{
							"err":     err.Error(),
							"message": whitelistRequest,
						}).Error("Failed to send decision email")
						d.Nack(false, false)
					}
					d.Ack(false)
				} else if whitelistRequest.Status == "Pending" {
					// Need to handle new request
					// Send application confirmation email to user
					emailConfirmation(whitelistRequest, config)
					// Send approval request emails to op(s)
					err := emailToOps(whitelistRequest, 1, ops, config)
					if err != nil {
						// If success count for sending ops emails less than minimum quoram, put to dead letter queue
						log.WithFields(logrus.Fields{
							"err":     err.Error(),
							"message": whitelistRequest,
						}).Error("Failed to reach required number of ops")
						d.Nack(false, false)
					}
					d.Ack(false)
				}
			}
		}
	}()

	log.Printf(" [*] Worker start. Listening for messages..")
	<-forever
}

func emailDecision(whitelistRequest types.WhitelistRequest, c *config.Config) error {
	requestIDToken, err := utils.EncodeAndEncrypt(whitelistRequest.ID.Hex(), c.PassPhrase)
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to encode requestID Token")
		return err
	}
	var subject string
	var template string
	if whitelistRequest.Status == "Approved" {
		subject = "Your request to join the server is approved"
		template = "./mailer/templates/approve.html"
	} else {
		subject = "Update regarding your request to join the server"
		template = "./mailer/templates/deny.html"
	}
	err = mailer.Send(template, map[string]string{"link": requestIDToken}, subject, whitelistRequest.Email, c)
	if err != nil {
		log.WithFields(logrus.Fields{
			"recipent": whitelistRequest.Email,
			"err":      err,
		}).Error("Failed to send decision email")
	} else {
		log.WithFields(logrus.Fields{
			"recipent": whitelistRequest.Email,
		}).Info("Decision email sent")
	}
	return err
}

func emailConfirmation(whitelistRequest types.WhitelistRequest, c *config.Config) error {
	subject := "Your request to join the server has been received"
	requestIDToken, err := utils.EncodeAndEncrypt(whitelistRequest.ID.Hex(), c.PassPhrase)
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to encode requestID Token")
		return err
	}
	err = mailer.Send("./mailer/templates/confirmation.html", map[string]string{"link": requestIDToken}, subject, whitelistRequest.Email, c)
	if err != nil {
		log.WithFields(logrus.Fields{
			"recipent": whitelistRequest.Email,
			"err":      err,
		}).Error("Failed to send confirmation email")
	} else {
		log.WithFields(logrus.Fields{
			"recipent": whitelistRequest.Email,
		}).Info("Confirmation email sent")
	}
	return err
}

func emailToOps(whitelistRequest types.WhitelistRequest, quoram int, ops []string, c *config.Config) error {
	subject := "[Action Required] Whitelist request from " + whitelistRequest.Username
	successCount := 0
	requestIDToken, err := utils.EncodeAndEncrypt(whitelistRequest.ID.Hex(), c.PassPhrase)
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to encode requestID Token")
		return err
	}
	for _, op := range ops {
		opEmailToken, err := utils.EncodeAndEncrypt(op, c.PassPhrase)
		if err != nil {
			log.WithFields(logrus.Fields{
				"err": err,
			}).Error("Failed to encode opEmail Token")
			return err
		}
		err = mailer.Send("./mailer/templates/ops.html", map[string]string{"link": requestIDToken + "?adm=" + opEmailToken}, subject, op, c)
		if err != nil {
			log.WithFields(logrus.Fields{
				"recipent": op,
				"err":      err,
			}).Error("Failed to send email to op")
		} else {
			log.WithFields(logrus.Fields{
				"recipent": op,
			}).Info("Action email sent to op")
			successCount++
		}
	}
	if successCount >= quoram {
		return nil
	}
	return errors.New("Failed to send action emails to more than half of ops")
}
func deserialize(b []byte) (types.WhitelistRequest, error) {
	var msg types.WhitelistRequest
	buf := bytes.NewBuffer(b)
	decoder := json.NewDecoder(buf)
	err := decoder.Decode(&msg)
	return msg, err
}
