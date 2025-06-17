package nramqp

import (
	"context"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
)

const (
	RabbitMQLibrary = "RabbitMQ"
)

func init() { internal.TrackUsage("integration", "messagebroker", "nramqp") }

func createProducerSegment(exchange, key string) *newrelic.MessageProducerSegment {
	s := newrelic.MessageProducerSegment{
		Library:         RabbitMQLibrary,
		DestinationName: "Default",
		DestinationType: newrelic.MessageQueue,
	}

	if exchange != "" {
		s.DestinationName = exchange
		s.DestinationType = newrelic.MessageExchange
	} else if key != "" {
		s.DestinationName = key
	}

	return &s
}

func GetHostAndPortFromURL(url string) (string, string) {
	// url is of format amqp://user:password@host:port or amqp://host:port
	var hostPortPart string

	// extract the part after "@" symbol, if present
	if parts := strings.Split(url, "@"); len(parts) == 2 {
		hostPortPart = parts[1]
	} else {
		// assume the whole url after "amqp://" is the host:port part
		hostPortPart = strings.TrimPrefix(url, "amqp://")
	}

	// split the host:port part
	strippedURL := strings.Split(hostPortPart, ":")
	if len(strippedURL) != 2 {
		return "", ""
	}
	return strippedURL[0], strippedURL[1]
}

// PublishedWithContext looks for a newrelic transaction in the context object, and if found, creates a message producer segment.
// It will also inject distributed tracing headers into the message.
func PublishWithContext(ch *amqp.Channel, ctx context.Context, exchange, key, url string, mandatory, immediate bool, msg amqp.Publishing) error {
	host, port := GetHostAndPortFromURL(url)
	txn := newrelic.FromContext(ctx)
	if txn != nil {
		// generate message broker segment
		s := createProducerSegment(exchange, key)

		// capture telemetry for AMQP producer
		if msg.Headers != nil && len(msg.Headers) > 0 {
			hdrStr, err := getHeadersAttributeString(msg.Headers)
			if err != nil {
				return err
			}
			integrationsupport.AddAgentSpanAttribute(txn, newrelic.AttributeMessageHeaders, hdrStr)
		}
		s.StartTime = txn.StartSegmentNow()

		// inject DT headers into headers object
		msg.Headers = injectDtHeaders(txn, msg.Headers)
		integrationsupport.AddAgentSpanAttribute(txn, newrelic.AttributeSpanKind, "producer")
		integrationsupport.AddAgentSpanAttribute(txn, newrelic.AttributeServerAddress, host)
		integrationsupport.AddAgentSpanAttribute(txn, newrelic.AttributeServerPort, port)
		integrationsupport.AddAgentSpanAttribute(txn, newrelic.AttributeMessageDestinationName, exchange)
		integrationsupport.AddAgentSpanAttribute(txn, newrelic.AttributeMessageRoutingKey, key)
		integrationsupport.AddAgentSpanAttribute(txn, newrelic.AttributeMessageCorrelationID, msg.CorrelationId)
		integrationsupport.AddAgentSpanAttribute(txn, newrelic.AttributeMessageReplyTo, msg.ReplyTo)

		err := ch.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
		s.End()
		return err
	} else {
		return ch.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
	}
}

// Consume performs a consume request on the provided amqp Channel, and returns a consume function, a consumer channel, and an error.
// The consumer function should be applied to each amqp Delivery that is read from the consume Channel, in order to collect tracing data
// on that message. The consume function will then return a transaction for that message.
func Consume(app *newrelic.Application, ch *amqp.Channel, queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (func(amqp.Delivery) *newrelic.Transaction, <-chan amqp.Delivery, error) {
	var handler func(amqp.Delivery) *newrelic.Transaction
	if app != nil {
		handler = func(delivery amqp.Delivery) *newrelic.Transaction {
			namer := internal.MessageMetricKey{
				Library:         RabbitMQLibrary,
				DestinationType: string(newrelic.MessageExchange),
				DestinationName: queue,
				Consumer:        true,
			}

			txn := app.StartTransaction(namer.Name())

			hdrs := toHeader(delivery.Headers)
			txn.AcceptDistributedTraceHeaders(newrelic.TransportAMQP, hdrs)

			if delivery.Headers != nil && len(delivery.Headers) > 0 {
				hdrStr, err := getHeadersAttributeString(delivery.Headers)
				if err == nil {
					integrationsupport.AddAgentAttribute(txn, newrelic.AttributeMessageHeaders, hdrStr, nil)
				}
			}
			integrationsupport.AddAgentAttribute(txn, newrelic.AttributeSpanKind, "consumer", nil)
			integrationsupport.AddAgentAttribute(txn, newrelic.AttributeMessageQueueName, queue, nil)
			integrationsupport.AddAgentAttribute(txn, newrelic.AttributeMessageDestinationName, queue, nil)
			integrationsupport.AddAgentAttribute(txn, newrelic.AttributeMessagingDestinationPublishName, delivery.Exchange, nil)
			integrationsupport.AddAgentAttribute(txn, newrelic.AttributeMessageRoutingKey, delivery.RoutingKey, nil)
			integrationsupport.AddAgentAttribute(txn, newrelic.AttributeMessageCorrelationID, delivery.CorrelationId, nil)
			integrationsupport.AddAgentAttribute(txn, newrelic.AttributeMessageReplyTo, delivery.ReplyTo, nil)

			return txn
		}
	}

	msgChan, err := ch.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
	return handler, msgChan, err
}
