package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/nramqp"
	"github.com/newrelic/go-agent/v3/newrelic"

	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		panic(fmt.Sprintf("%s: %s\n", msg, err))
	}
}

// a rabit mq server must be running on localhost on port 5672
func main() {
	nrApp, err := newrelic.NewApplication(
		newrelic.ConfigAppName("AMQP Consumer Example App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigInfoLogger(os.Stdout),
	)

	if err != nil {
		panic(err)
	}

	nrApp.WaitForConnection(time.Second * 5)

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	var forever chan struct{}

	handleDelivery, msgs, err := nramqp.Consume(nrApp, ch,
		q.Name,
		"",
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args)
	)
	failOnError(err, "Failed to register a consumer")

	go func() {
		for d := range msgs {
			txn := handleDelivery(d)
			log.Printf("Received a message: %s\n", d.Body)
			txn.End()
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever

	nrApp.Shutdown(time.Second * 10)
}
