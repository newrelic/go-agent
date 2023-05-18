package nrkafka

import (
	"context"
	"log"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/stretchr/testify/mock"
)

type MockConsumerGroupSession struct {
	mock.Mock
}

func (m *MockConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {}
func (m *MockConsumerGroupSession) Commit()                                                  {}
func (m *MockConsumerGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
}
func (m *MockConsumerGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
}
func (m *MockConsumerGroupSession) Context() context.Context   { return nil }
func (m *MockConsumerGroupSession) Claims() map[string][]int32 { return nil }
func (m *MockConsumerGroupSession) MemberID() string           { return "" }
func (m *MockConsumerGroupSession) GenerationID() int32        { return 0 }

func TestProducerSendMessage(t *testing.T) {
	producer := mocks.NewSyncProducer(t, nil)
	producer.ExpectSendMessageAndSucceed()
	txn := &newrelic.Transaction{}
	kw := NewProducerWrapper(producer, txn)

	// Compose message
	key := []byte("key")
	msg := []byte("value")
	err := kw.SendMessage("topicName", key, msg)

	if nil != err {
		t.Error(err)
	}
}

func TestProducerSetHeaders(t *testing.T) {
	producer := mocks.NewSyncProducer(t, nil)
	txn := &newrelic.Transaction{}
	kw := NewProducerWrapper(producer, txn)

	// Create kafka message
	keyEncoded := sarama.ByteEncoder("key")
	valEncoded := sarama.ByteEncoder("val")
	msg := &sarama.ProducerMessage{
		Topic: "topic",
		Key:   keyEncoded,
		Value: valEncoded,
	}
	// Set Headers
	carrier := kw.carrier(msg)
	carrier.Set("k", "v")

	// check to see if headers set in carrier are correct
	carrierhdrs := carrier.Header
	hdrs := make(http.Header)
	hdrs.Set("k", "v")
	eq := reflect.DeepEqual(carrierhdrs, hdrs)
	if !eq {
		t.Error("actual headers does not match what is expected", carrierhdrs, hdrs)
	}

}

// Custom message handler that controls what happens when a new message is received by the consumer
func messageHandler(ctx context.Context, msg *sarama.ConsumerMessage) {
	log.Printf("received message %v\n", string(msg.Key))
}

func TestConsumerClaimIngestion(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()

	// Setup sarama config, including session timeout/heartbeat intervals
	config := sarama.NewConfig()
	config.ClientID = "CustomClientID"
	config.Consumer.Group.Session.Timeout = 10 * time.Second
	config.Consumer.Group.Heartbeat.Interval = 3 * time.Second

	kafkaTopicName := "topicName"
	keyEncoded := sarama.ByteEncoder("key")
	encodedValue := sarama.ByteEncoder("value")
	msg := &sarama.ConsumerMessage{
		Topic:   "topic",
		Key:     keyEncoded,
		Value:   encodedValue,
		Headers: []*sarama.RecordHeader{},
	}

	mockSession := new(MockConsumerGroupSession)

	ch := NewConsumerHandler(app.Application, kafkaTopicName, config.ClientID, config, messageHandler)
	ClaimIngestion(ch, mockSession, msg)

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransactionTotalTime/Go/kafkaconsumer"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all"},
		{Name: "Custom/MessageBroker/Kafka/Topic/Named/topicName/Deserialization/Key", Scope: "OtherTransaction/Go/kafkaconsumer", Forced: false, Data: nil},
		{Name: "Custom/Message/Kafka/Topic/Consume/Named/topicName/MessageProcessing/", Scope: "OtherTransaction/Go/kafkaconsumer"},
		{Name: "Custom/MessageBroker/Kafka/Topic/Named/topicName/Deserialization/Value", Scope: "OtherTransaction/Go/kafkaconsumer", Forced: false, Data: nil},
		{Name: "Custom/Message/Kafka/Topic/Consume/Named/topicName", Scope: "OtherTransaction/Go/kafkaconsumer"},
		{Name: "OtherTransaction/all"},
		{Name: "Custom/MessageBroker/Kafka/Heartbeat/Receive"},
		{Name: "OtherTransaction/Go/kafkaconsumer"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther"},
		{Name: "Custom/MessageBroker/Kafka/Topic/Named/topicName/Deserialization/Key"},
		{Name: "Custom/Message/Kafka/Topic/Named/topicName/Received/Bytes"},
		{Name: "Custom/MessageBroker/Kafka/Topic/Named/topicName/Deserialization/Value"},
		{Name: "Custom/Message/Kafka/Topic/Consume/Named/topicName/MessageProcessing/"},
		{Name: "Custom/Message/Kafka/Topic/Named/topicName/Received/Messages"},
		{Name: "Custom/Message/Kafka/Topic/Consume/Named/topicName"},
		{Name: "OtherTransactionTotalTime"},
	})
}
