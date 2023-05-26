package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Shopify/sarama"
	nrsaramaconsumer "github.com/newrelic/go-agent/v3/integrations/nrsarama"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var brokers = []string{"localhost:9092"}

// Custom message handler that controls what happens when a new message is received by the consumer
// Note: delay is present only to simulate handling of message
func messageHandler(ctx context.Context, msg *sarama.ConsumerMessage) {
	log.Printf("received message %v\n", string(msg.Key))
	delay := time.Duration(2 * time.Millisecond)
	time.Sleep(delay)
}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Kafka App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigDistributedTracerEnabled(true),
	)

	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}
	// Wait for the application to connect.
	if err := app.WaitForConnection(5 * time.Second); nil != err {
		fmt.Println(err)
	}

	// Setup sarama config, including session timeout/heartbeat intervals
	config := sarama.NewConfig()
	config.ClientID = "CustomClientID"
	config.Consumer.Group.Session.Timeout = 10 * time.Second
	config.Consumer.Group.Heartbeat.Interval = 3 * time.Second

	// Create new sarama consumer group
	consumerGroup, err := sarama.NewConsumerGroup(brokers, "test-group", config)

	if nil != err {
		fmt.Println(err)
	}
	kafkaTopicName := "topicName"

	// Create new kafka consumer handler
	handler := nrsaramaconsumer.NewConsumerHandler(app, kafkaTopicName, config.ClientID, config, messageHandler)

	for {
		err := consumerGroup.Consume(context.Background(), []string{kafkaTopicName}, handler)
		if nil != err {
			fmt.Println(err)

		}
	}

	// NOTE: Whenever the consumer no longer accepts messages be sure to close it out using consumerGroup.Close()

}
