package worker

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/tywin1104/mc-whitelist/db"
	"github.com/tywin1104/mc-whitelist/mailer"
	"github.com/tywin1104/mc-whitelist/rcon"
	"github.com/tywin1104/mc-whitelist/types"
	"github.com/tywin1104/mc-whitelist/utils"
	"go.mongodb.org/mongo-driver/bson"
	try "gopkg.in/matryer/try.v1"
)

// Worker defines message queue worker
type Worker struct {
	dbService        *db.Service
	logger           *logrus.Entry
	rconClient       *rcon.Client
	conn             *amqp.Connection
	channel          *amqp.Channel
	rabbitCloseError chan *amqp.Error
	delivery         <-chan amqp.Delivery
}

// NewWorker creates a worker to constantly listen and handle messages in the queue
func NewWorker(db *db.Service, logger *logrus.Entry) (*Worker, error) {
	// Initialize rcon client to interact with game server
	rconClient, err := rcon.NewClient(viper.GetString("RCONServer"), viper.GetInt("RCONPort"), viper.GetString("RCONPassword"))
	if err != nil {
		return nil, err
	}
	return &Worker{
		dbService:        db,
		logger:           logger,
		rconClient:       rconClient,
		rabbitCloseError: make(chan *amqp.Error),
	}, nil
}

func (worker *Worker) failOnError(err error, msg string) {
	if err != nil {
		worker.logger.WithFields(logrus.Fields{
			"err": err,
		}).Fatal(msg)
	}
}

// Close connection and channel associated with the worker
func (worker *Worker) Close() {
	worker.channel.Close()
	worker.conn.Close()
}

// Start the worker to process the messages pushed into the queue
func (worker *Worker) Start(wg *sync.WaitGroup) {
	log := worker.logger

	conn, err := amqp.Dial(viper.GetString("rabbitMQConn"))
	worker.failOnError(err, "Failed to connect to RabbitMQ")
	worker.conn = conn
	worker.rabbitCloseError = worker.conn.NotifyClose(make(chan *amqp.Error))

	ch, err := conn.Channel()
	worker.failOnError(err, "Failed to open a channel")
	worker.channel = ch
	args := make(amqp.Table)
	// Dead letter exchange name
	args["x-dead-letter-exchange"] = "dead.letter.ex"
	// Default message ttl 24 hours
	args["x-message-ttl"] = int32(8.64e+7)

	_, err = ch.QueueDeclare(
		viper.GetString("taskQueueName"), // name
		true,                             // durable
		false,                            // delete when unused
		false,                            // exclusive
		false,                            // no-wait
		args,                             // arguments
	)
	worker.failOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	worker.failOnError(err, "Failed to set QoS")

	forever := make(chan bool)
	// Set initial delivery channel from the initial connection
	worker.updateDeliveryChannel()
	go worker.runLoop()
	log.Info("Worker started. Listening for messages..")
	wg.Done()

	<-forever
}

// Update the messages fetching origin to be from the channel of the new connection
// Is called whenver a new connection is established and the old one is closed
func (worker *Worker) updateDeliveryChannel() {
	msgs, err := worker.channel.Consume(
		viper.GetString("taskQueueName"), // queue
		"",                               // consumer
		false,                            // auto-ack
		false,                            // exclusive
		false,                            // no-local
		false,                            // no-wait
		nil,                              // args
	)
	worker.failOnError(err, "Failed to register a consumer")
	worker.delivery = msgs
}

func (worker *Worker) reconnect() {
	worker.logger.Warning("Worker connection with message queue closed unexpectedly. About to reconnect")
	worker.rabbitCloseError = make(chan *amqp.Error)

	conn, err := amqp.Dial(viper.GetString("rabbitMQConn"))
	worker.failOnError(err, "Failed to connect to RabbitMQ")
	worker.conn = conn

	ch, err := conn.Channel()
	worker.failOnError(err, "Failed to open a channel")
	worker.channel = ch
	// Update worker's delivery from newly created channel of new connection
	worker.updateDeliveryChannel()
	worker.logger.Info("Worker-message queue connection established. Continue to process messages")
	worker.conn.NotifyClose(worker.rabbitCloseError)
}
func (worker *Worker) runLoop() {
	for {
		select {
		case rabbitErr := <-worker.rabbitCloseError:
			if rabbitErr != nil {
				worker.reconnect()
			}
			break
		case d := <-worker.delivery:
			log := worker.logger
			whitelistRequest, err := deserialize(d.Body)
			if err != nil {
				log.WithFields(logrus.Fields{
					"messageBody": d.Body,
					"err":         err,
				}).Error("Unable to decode message into whitelistRequest")
				// Unable to decode this message, put to the dead-letter queue
				d.Nack(false, false)
			} else {
				// Concrete actions to do when receiving task from message queue
				// From the message body to determine which type of work to do
				switch whitelistRequest.Status {
				case "Approved":
					worker.processApproval(d, whitelistRequest)
				case "Denied":
					worker.processDenial(d, whitelistRequest)
				case "Pending":
					worker.processNewRequest(d, whitelistRequest)
				}
			}
		}
	}
}

// Nack if decision email is not sent. Ack if sent.
func (worker *Worker) processApproval(d amqp.Delivery, request types.WhitelistRequest) {
	worker.logger.WithFields(logrus.Fields{
		"username": request.Username,
		"ID":       request.ID,
		"Type":     "Approval Task",
	}).Info("Received new task")

	// Concrete whitelist action on the game server
	err := worker.whitelistUser(request.Username)
	if err != nil {
		worker.logger.WithFields(logrus.Fields{
			"username": request.Username,
			"err":      err.Error(),
		}).Error("Unable to issue whitelist cmd on the game server")
		d.Nack(false, false)
		return
	}
	err = worker.emailDecision(request)
	if err != nil {
		d.Nack(false, false)
		return
	}
	d.Ack(false)
}

// Nack if decision email is not sent. Ack if sent.
func (worker *Worker) processDenial(d amqp.Delivery, request types.WhitelistRequest) {
	// Need to send update status back to the user
	// Put message to dead letter queue for later investigation if unable to send decision email
	worker.logger.WithFields(logrus.Fields{
		"username": request.Username,
		"ID":       request.ID,
		"Type":     "Denial Task",
	}).Info("Received new task")
	err := worker.emailDecision(request)
	if err != nil {
		d.Nack(false, false)
		return
	}
	d.Ack(false)
}

//Nack: successful ops emails less than threshold; confirmation email does not count
func (worker *Worker) processNewRequest(d amqp.Delivery, request types.WhitelistRequest) {
	worker.logger.WithFields(logrus.Fields{
		"username": request.Username,
		"ID":       request.ID,
		"Type":     "New Reqeust Task",
	}).Info("Received new task")
	// Need to handle new request
	// Send application confirmation email to user
	worker.emailConfirmation(request)
	// Send approval request emails to op(s)
	successCount, err := worker.emailToOps(request, viper.GetInt("minRequiredReceiver"))
	if err != nil {
		// If success count for sending ops emails less than minimum quoram, put to dead letter queue
		worker.logger.WithFields(logrus.Fields{
			"message":      request,
			"successCount": successCount,
		}).Error("Failed to dispatch action emails to required number of ops")
		d.Nack(false, false)
		return
	}
	d.Ack(false)
}

func (worker *Worker) emailDecision(whitelistRequest types.WhitelistRequest) error {
	log := worker.logger
	requestIDToken, err := utils.EncodeAndEncrypt(whitelistRequest.ID.Hex(), viper.GetString("passphrase"))
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to encode requestID Token")
		return err
	}
	var subject string
	var template string
	if whitelistRequest.Status == "Approved" {
		subject = viper.GetString("approvedEmailTitle")
		template = "./mailer/templates/approve.html"
	} else {
		subject = viper.GetString("deniedEmailTitle")
		template = "./mailer/templates/deny.html"
	}
	err = mailer.Send(template, map[string]string{"link": requestIDToken}, subject, whitelistRequest.Email)
	if err != nil {
		log.WithFields(logrus.Fields{
			"recipent": whitelistRequest.Email,
			"err":      err,
			"ID":       whitelistRequest.ID.Hex(),
		}).Error("Failed to send decision email")
	} else {
		log.WithFields(logrus.Fields{
			"recipent": whitelistRequest.Email,
		}).Info("Decision email sent")
	}
	return err
}

func (worker *Worker) emailConfirmation(whitelistRequest types.WhitelistRequest) error {
	log := worker.logger
	subject := viper.GetString("confirmationEmailTitle")
	requestIDToken, err := utils.EncodeAndEncrypt(whitelistRequest.ID.Hex(), viper.GetString("passphrase"))
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to encode requestID Token")
		return err
	}
	confirmationLink := os.Getenv("FRONTEND_DEPLOYED_URL") + "status/" + requestIDToken
	err = mailer.Send("./mailer/templates/confirmation.html", map[string]string{"link": confirmationLink}, subject, whitelistRequest.Email)
	if err != nil {
		log.WithFields(logrus.Fields{
			"recipent": whitelistRequest.Email,
			"err":      err,
			"ID":       whitelistRequest.ID.Hex(),
		}).Error("Failed to send confirmation email")
	} else {
		log.WithFields(logrus.Fields{
			"recipent": whitelistRequest.Email,
		}).Info("Confirmation email sent")
	}
	return err
}

func (worker *Worker) emailToOps(whitelistRequest types.WhitelistRequest, quoram int) (int, error) {
	log := worker.logger
	subject := "[Action Required] Whitelist request from " + whitelistRequest.Username
	successCount := 0
	requestIDToken, err := utils.EncodeAndEncrypt(whitelistRequest.ID.Hex(), viper.GetString("passphrase"))
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to encode requestID Token")
		return 0, err
	}
	// ops who received the action emails successfully will be added to the assignees
	// and attach as the metadata for the request db object
	assignees := []string{}
	// Get target ops to send action emails according to the configured dispatching strategy
	ops := worker.getTargetOps()
	for _, op := range ops {
		opEmailToken, err := utils.EncodeAndEncrypt(op, viper.GetString("passphrase"))
		if err != nil {
			log.WithFields(logrus.Fields{
				"err": err,
			}).Error("Failed to encode opEmail Token")
			return 0, err
		}
		opLink := os.Getenv("FRONTEND_DEPLOYED_URL") + "action/" + requestIDToken + "?adm=" + opEmailToken
		err = mailer.Send("./mailer/templates/ops.html", map[string]string{"link": opLink}, subject, op)
		if err != nil {
			log.WithFields(logrus.Fields{
				"recipent": op,
				"err":      err,
				"ID":       whitelistRequest.ID.Hex(),
			}).Error("Failed to send email to op")
		} else {
			log.WithFields(logrus.Fields{
				"recipent": op,
				"ID":       whitelistRequest.ID.Hex(),
			}).Info("Action email sent to op")
			assignees = append(assignees, op)
			successCount++
		}
	}
	// Attach assignee info to the db request object to keep track of each request
	if len(assignees) > 0 {
		requestedChange := make(bson.M)
		requestedChange["assignees"] = assignees
		_, err := worker.dbService.UpdateRequest(bson.D{{"_id", whitelistRequest.ID}}, bson.M{
			"$set": requestedChange,
		})
		if err != nil {
			log.WithFields(logrus.Fields{
				"err":       err,
				"assignees": assignees,
				"ID":        whitelistRequest.ID.Hex(),
			}).Error("Unable to update request db object with assignees metadata")
		}
	}
	if successCount >= quoram {
		return successCount, nil
	}
	return successCount, errors.New("Success count does not reach minimum requirement")
}

func (worker *Worker) getTargetOps() []string {
	// Strategy: Broadcast / Random with threshold
	ops := viper.GetStringSlice("ops")
	if viper.GetString("dispatchingStrategy") == "Broadcast" {
		return ops
	}
	n := viper.GetInt("randomDispatchingThreshold")
	// Choose random n out of all ops as the target request handlers
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(ops), func(i, j int) { ops[i], ops[j] = ops[j], ops[i] })
	return ops[:n]
}

// issue whitelist command againest a user with retries
func (worker *Worker) whitelistUser(username string) error {
	err := try.Do(func(attempt int) (bool, error) {
		_, e := worker.rconClient.SendCommand("whitelist add " + username)
		if e != nil {
			time.Sleep(5 * time.Second) // 5 seconds delay between retrys
		}
		return attempt < 3, e // try 3 times
	})
	if err != nil {
		return err
	}
	worker.logger.WithFields(logrus.Fields{
		"username": username,
	}).Info("User has been whitelisted successfully")
	return nil
}
func deserialize(b []byte) (types.WhitelistRequest, error) {
	var msg types.WhitelistRequest
	buf := bytes.NewBuffer(b)
	decoder := json.NewDecoder(buf)
	err := decoder.Decode(&msg)
	return msg, err
}
