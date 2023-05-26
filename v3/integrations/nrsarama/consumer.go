package nrsarama

import (
	"context"
	"net/http"

	"github.com/Shopify/sarama"
	"github.com/newrelic/go-agent/v3/internal"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "messagebroker", "saramaconsumer") }

type ConsumerWrapper struct {
	consumerGroup sarama.ConsumerGroup
}

type ConsumerHandler struct {
	app            *newrelic.Application
	txn            *newrelic.Transaction
	topic          string
	clientID       string
	saramaConfig   *sarama.Config
	messageHandler func(ctx context.Context, message *sarama.ConsumerMessage)
}

// NOTE: Creates and ends one transaction per claim consumed

// NewConsumerHandlerFromApp takes in a new relic application and creates a transaction using it
func NewConsumerHandlerFromApp(app *newrelic.Application, topic string, clientID string, saramaConfig *sarama.Config, messageHandler func(ctx context.Context, message *sarama.ConsumerMessage)) *ConsumerHandler {
	return &ConsumerHandler{
		app:            app,
		topic:          topic,
		messageHandler: messageHandler,
		saramaConfig:   saramaConfig,
		clientID:       clientID,
	}
}

// NewConsumerHandlerFromTxn takes in a new relic transaction. No application instance is required
func NewConsumerHandlerFromTxn(txn *newrelic.Transaction, topic string, clientID string, saramaConfig *sarama.Config, messageHandler func(ctx context.Context, message *sarama.ConsumerMessage)) *ConsumerHandler {
	return &ConsumerHandler{
		txn:            txn,
		topic:          topic,
		messageHandler: messageHandler,
		saramaConfig:   saramaConfig,
		clientID:       clientID,
	}
}

func (cw *ConsumerWrapper) Consume(ctx context.Context, handler *ConsumerHandler) error {
	txn := newrelic.FromContext(ctx)
	consume := cw.consumerGroup.Consume(ctx, []string{handler.topic}, handler)
	if consume != nil {
		txn.Application().RecordCustomMetric("MessageBroker/Kafka/Heartbeat/Fail", 1.0)
	}
	return nil
}

// Setup is ran at the beginning of a new session
func (ch *ConsumerHandler) Setup(_ sarama.ConsumerGroupSession) error {
	// Record session timeout/poll timeout intervals
	ch.app.RecordCustomMetric("MessageBroker/Kafka/Heartbeat/SessionTimeout", ch.saramaConfig.Consumer.Group.Session.Timeout.Seconds())
	ch.app.RecordCustomMetric("MessageBroker/Kafka/Heartbeat/PollTimeout", ch.saramaConfig.Consumer.Group.Heartbeat.Interval.Seconds())

	return nil
}

// Cleanup is ran at the end of a new session
func (ch *ConsumerHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func ClaimIngestion(ch *ConsumerHandler, session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
	// if txn exists, make claims segments of that txn otherwise create a new one
	txn := ch.txn
	if ch.txn == nil {
		txn = ch.app.StartTransaction("kafkaconsumer")
	}
	ctx := newrelic.NewContext(context.Background(), txn)
	segment := txn.StartSegment("Message/Kafka/Topic/Consume/Named/" + ch.topic)

	// Deserialized key/value
	deserializeKeySegment := txn.StartSegment("MessageBroker/Kafka/Topic/Named/" + ch.topic + "/Deserialization/Key")
	key := string(message.Key)
	deserializeKeySegment.End()

	deserializeVaueSegment := txn.StartSegment("MessageBroker/Kafka/Topic/Named/" + ch.topic + "/Deserialization/Value")
	value := string(message.Value)
	deserializeVaueSegment.End()

	ch.processMessage(ctx, message, key, value)
	segment.End()

	session.MarkMessage(message, "")

	// Heartbeat metric to log a new message received successfully
	txn.Application().RecordCustomMetric("MessageBroker/Kafka/Heartbeat/Receive", 1.0)
	txn.End()

}

func (ch *ConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		ClaimIngestion(ch, session, message)
	}
	return nil
}

func (ch *ConsumerHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage, key string, value string) {
	txn := newrelic.FromContext(ctx)
	messageHandlingSegment := txn.StartSegment("Message/Kafka/Topic/Consume/Named/" + ch.topic + "/MessageProcessing/")
	ch.messageHandler(ctx, message)
	byteCount := float64(len(message.Value))
	hdrs := http.Header{}
	for _, hdr := range message.Headers {
		hdrs.Add(string(hdr.Key), string(hdr.Value))

	}

	txn.InsertDistributedTraceHeaders(hdrs)

	txn.AddAttribute("kafka.consume.byteCount", byteCount)
	txn.AddAttribute("kafka.consume.ClientID", ch.clientID)

	txn.Application().RecordCustomMetric("Message/Kafka/Topic/Named/"+ch.topic+"/Received/Bytes", byteCount)
	txn.Application().RecordCustomMetric("Message/Kafka/Topic/Named/"+ch.topic+"/Received/Messages", 1)
	messageHandlingSegment.End()

}
