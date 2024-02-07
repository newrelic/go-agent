package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Shopify/sarama"
	nrsaramaproducer "github.com/newrelic/go-agent/v3/integrations/nrsarama"

	"github.com/newrelic/go-agent/v3/newrelic"
)

var brokers = []string{"localhost:9092"}

func main() {

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Kafka App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigDebugLogger(os.Stdout),
	)

	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}
	// Wait for the application to connect.
	if err := app.WaitForConnection(5 * time.Second); nil != err {
		fmt.Println(err)
	}

	// Sarama Producer configuration settings
	config := sarama.NewConfig()
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Return.Successes = true

	// Create Producer
	producer, err := sarama.NewSyncProducer(brokers, config)
	if nil != err {
		fmt.Println(err)
	}

	// Start new transaction
	txn := app.StartTransaction("kafkaproducer")

	kw := nrsaramaproducer.NewProducerWrapper(producer, txn)
	topic := "topicName"
	// Generate and send multiple messages
	numMessages := 10
	for i := 0; i < numMessages; i++ {
		key := []byte("key-" + strconv.Itoa(i))
		msg := []byte("test Message " + strconv.Itoa(i))

		err = kw.SendMessage(topic, key, msg)
		if nil != err {
			fmt.Println(err)
		}
	}
	txn.End()

	app.Shutdown(10 * time.Second)
}
