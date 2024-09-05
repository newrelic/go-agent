package nramqp

import (
	"testing"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func BenchmarkCreateProducerSegment(b *testing.B) {
	app := createTestApp()
	txn := app.StartTransaction("test")
	defer txn.End()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		createProducerSegment("exchange", "key")
	}
}

func TestCreateProducerSegment(t *testing.T) {
	app := createTestApp()
	txn := app.StartTransaction("test")
	defer txn.End()

	type testObject struct {
		exchange string
		key      string
		expect   newrelic.MessageProducerSegment
	}

	tests := []testObject{
		{
			"test exchange",
			"",
			newrelic.MessageProducerSegment{
				DestinationName: "test exchange",
				DestinationType: newrelic.MessageExchange,
			},
		},
		{
			"",
			"test queue",
			newrelic.MessageProducerSegment{
				DestinationName: "test queue",
				DestinationType: newrelic.MessageQueue,
			},
		},
		{
			"",
			"",
			newrelic.MessageProducerSegment{
				DestinationName: "Default",
				DestinationType: newrelic.MessageQueue,
			},
		},
		{
			"test exchange",
			"test queue",
			newrelic.MessageProducerSegment{
				DestinationName: "test exchange",
				DestinationType: newrelic.MessageExchange,
			},
		},
	}

	for _, test := range tests {
		s := createProducerSegment(test.exchange, test.key)
		if s.DestinationName != test.expect.DestinationName {
			t.Errorf("expected destination name %s, got %s", test.expect.DestinationName, s.DestinationName)
		}
		if s.DestinationType != test.expect.DestinationType {
			t.Errorf("expected destination type %s, got %s", test.expect.DestinationType, s.DestinationType)
		}
	}

}

func TestPublishWithContext(t *testing.T) {
	app := createTestApp()
	txn := app.StartTransaction("test")
	defer txn.End()

}

func TestHostAndPortParsing(t *testing.T) {
	app := createTestApp()
	txn := app.StartTransaction("test")
	defer txn.End()

	type testObject struct {
		url        string
		expectHost string
		expectPort string
	}

	tests := []testObject{
		{
			"amqp://user:password@host:port",
			"host",
			"port",
		},
		{
			"amqp://user:password@host",
			"",
			"",
		},
		{
			"amqp://user:password@host:port:extra",
			"",
			"",
		},
	}

	for _, test := range tests {
		host, port := GetHostAndPortFromURL(test.url)
		if host != test.expectHost {
			t.Errorf("expected host %s, got %s", test.expectHost, host)
		}
		if port != test.expectPort {
			t.Errorf("expected port %s, got %s", test.expectPort, port)
		}
	}

}
