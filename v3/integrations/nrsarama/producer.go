package nrsarama

import (
	"log"
	"net/http"

	"github.com/Shopify/sarama"
	"github.com/newrelic/go-agent/v3/internal"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "messagebroker", "saramakafka") }

type ProducerWrapper struct {
	producer sarama.SyncProducer
	txn      *newrelic.Transaction
}

type KafkaMessageCarrier struct {
	http.Header
	msg *sarama.ProducerMessage
}

func NewProducerWrapper(producer sarama.SyncProducer, txn *newrelic.Transaction) *ProducerWrapper {
	return &ProducerWrapper{
		producer: producer,
		txn:      txn,
	}
}
func (pw *ProducerWrapper) carrier(msg *sarama.ProducerMessage) *KafkaMessageCarrier {
	return &KafkaMessageCarrier{
		Header: make(http.Header),
		msg:    msg,
	}
}

func (carrier KafkaMessageCarrier) Set(key, val string) {
	carrier.Header.Set(key, val)
	carrier.msg.Headers = append(carrier.msg.Headers, sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(val),
	})
}

func (pw *ProducerWrapper) SendMessage(topic string, key []byte, value []byte) error {
	// Traces for encoding key/value
	keyEncoding := pw.txn.StartSegment("MessageBroker/Kafka/Topic/Named/" + topic + "/Serialization/Key")
	keyEncoded := sarama.ByteEncoder(key)
	keyEncoding.End()

	valueEncoding := pw.txn.StartSegment("MessageBroker/Kafka/Topic/Named/" + topic + "/Serialization/Value")
	encodedValue := sarama.ByteEncoder(value)
	valueEncoding.End()

	// Create kafka message
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   keyEncoded,
		Value: encodedValue,
	}
	// DT Headers
	carrier := pw.carrier(msg)
	pw.txn.InsertDistributedTraceHeaders(carrier.Header)

	// Send message using kafka producer
	producerSegment := pw.txn.StartSegment("MessageBroker/Kafka/Topic/Produce/Named/" + topic)
	partition, offset, err := pw.producer.SendMessage(msg)
	defer producerSegment.End()
	if err != nil {
		return err
	}
	log.Printf("Sent to partion %v and the offset is %v", partition, offset)
	return nil

}
