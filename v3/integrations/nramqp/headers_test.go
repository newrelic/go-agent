package nramqp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"

	amqp "github.com/rabbitmq/amqp091-go"
)

var replyFn = func(reply *internal.ConnectReply) {
	reply.SetSampleEverything()
	reply.AccountID = "123"
	reply.TrustedAccountKey = "123"
	reply.PrimaryAppID = "456"
}

var cfgFn = func(cfg *newrelic.Config) {
	cfg.Attributes.Include = append(cfg.Attributes.Include,
		newrelic.AttributeMessageRoutingKey,
		newrelic.AttributeMessageQueueName,
		newrelic.AttributeMessageExchangeType,
		newrelic.AttributeMessageReplyTo,
		newrelic.AttributeMessageCorrelationID,
		newrelic.AttributeMessageHeaders,
	)
}

func createTestApp() integrationsupport.ExpectApp {
	return integrationsupport.NewTestApp(replyFn, cfgFn, integrationsupport.ConfigFullTraces, newrelic.ConfigCodeLevelMetricsEnabled(false))
}

func TestAddHeaderAttribute(t *testing.T) {
	app := createTestApp()
	txn := app.StartTransaction("test")

	hdrs := amqp.Table{
		"str":          "hello",
		"int":          5,
		"bool":         true,
		"nil":          nil,
		"time":         time.Now(),
		"bytes":        []byte("a slice of bytes"),
		"decimal":      amqp.Decimal{Scale: 2, Value: 12345},
		"zero decimal": amqp.Decimal{Scale: 0, Value: 12345},
	}
	attrStr, err := getHeadersAttributeString(hdrs)
	if err != nil {
		t.Fatal(err)
	}
	integrationsupport.AddAgentAttribute(txn, newrelic.AttributeMessageHeaders, attrStr, hdrs)

	txn.End()

	app.ExpectTxnTraces(t, []internal.WantTxnTrace{
		{
			AgentAttributes: map[string]interface{}{
				newrelic.AttributeMessageHeaders: attrStr,
			},
		},
	})
}

func TestInjectHeaders(t *testing.T) {
	nrApp := createTestApp()
	txn := nrApp.StartTransaction("test txn")
	defer txn.End()

	msg := amqp.Publishing{}
	msg.Headers = injectDtHeaders(txn, msg.Headers)

	if len(msg.Headers) != 3 {
		t.Error("Expected DT headers to be injected into Headers object")
	}
}

func TestInjectHeadersPreservesExistingHeaders(t *testing.T) {
	nrApp := createTestApp()
	txn := nrApp.StartTransaction("test txn")
	defer txn.End()

	msg := amqp.Publishing{
		Headers: amqp.Table{
			"one": 1,
			"two": 2,
		},
	}
	msg.Headers = injectDtHeaders(txn, msg.Headers)

	if len(msg.Headers) != 5 {
		t.Error("Expected DT headers to be injected into Headers object")
	}
}

func TestToHeader(t *testing.T) {
	nrApp := createTestApp()
	txn := nrApp.StartTransaction("test txn")
	defer txn.End()

	msg := amqp.Publishing{
		Headers: amqp.Table{
			"one": 1,
			"two": 2,
		},
	}
	msg.Headers = injectDtHeaders(txn, msg.Headers)

	hdr := toHeader(msg.Headers)

	if v := hdr.Get(newrelic.DistributedTraceNewRelicHeader); v == "" {
		t.Errorf("header did not contain a DT header with the key %s", newrelic.DistributedTraceNewRelicHeader)
	}
	if v := hdr.Get(newrelic.DistributedTraceW3CTraceParentHeader); v == "" {
		t.Errorf("header did not contain a DT header with the key %s", newrelic.DistributedTraceW3CTraceParentHeader)
	}
	if v := hdr.Get(newrelic.DistributedTraceW3CTraceStateHeader); v == "" {
		t.Errorf("header did not contain a DT header with the key %s", newrelic.DistributedTraceW3CTraceStateHeader)
	}
}

func BenchmarkGetAttributeHeaders(b *testing.B) {
	hdrs := amqp.Table{
		"str":          "hello",
		"int":          5,
		"bool":         true,
		"nil":          nil,
		"time":         time.Now(),
		"bytes":        []byte("a slice of bytes"),
		"decimal":      amqp.Decimal{Scale: 2, Value: 12345},
		"zero decimal": amqp.Decimal{Scale: 0, Value: 12345},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		getHeadersAttributeString(hdrs)
	}
}

func TestGetAttributeHeaders(t *testing.T) {
	ti := time.Now()
	hdrs := amqp.Table{
		"str":          "hello",
		"int":          5,
		"bool":         true,
		"nil":          nil,
		"time":         ti,
		"bytes":        []byte("a slice of bytes"),
		"decimal":      amqp.Decimal{Scale: 2, Value: 12345},
		"zero decimal": amqp.Decimal{Scale: 0, Value: 12345},
		"array":        []interface{}{5, true, "hi", ti},
	}

	hdrStr, err := getHeadersAttributeString(hdrs)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(hdrStr)

	var v map[string]any
	err = json.Unmarshal([]byte(hdrStr), &v)
	if err != nil {
		t.Fatal(err)
	}

	if len(v) != 9 {
		t.Errorf("expected 6 key value pairs, but got %d", len(v))
	}

	_, ok := v["str"]
	if !ok {
		t.Error("string header key value pair was dropped")
	}

	_, ok = v["bytes"]
	if !ok {
		t.Error("bytes header key value pair was dropped")
	}

	_, ok = v["int"]
	if !ok {
		t.Error("int header key value pair was dropped")
	}

	_, ok = v["bool"]
	if !ok {
		t.Error("bool header key value pair was dropped")
	}

	_, ok = v["nil"]
	if !ok {
		t.Error("nil header key value pair was dropped")
	}

	_, ok = v["decimal"]
	if !ok {
		t.Error("decimal header key value pair was dropped")
	}

	_, ok = v["zero decimal"]
	if !ok {
		t.Error("zero decimal header key value pair was dropped")
	}

	_, ok = v["array"]
	if !ok {
		t.Error("array header key value pair was dropped")
	}

	_, ok = v["time"]
	if !ok {
		t.Error("time header key value pair was dropped")
	}
}

func TestGetAttributeHeadersEmpty(t *testing.T) {
	hdrs := amqp.Table{}

	hdrStr, err := getHeadersAttributeString(hdrs)
	if err != nil {
		t.Fatal(err)
	}
	if hdrStr != "" {
		t.Errorf("should return empty string for empty or nil header table, instead got: %s", hdrStr)
	}
}

func TestGetAttributeHeadersNil(t *testing.T) {
	hdrStr, err := getHeadersAttributeString(nil)
	if err != nil {
		t.Fatal(err)
	}
	if hdrStr != "" {
		t.Errorf("should return empty string for empty or nil header table, instead got: %s", hdrStr)
	}
}

func TestGetAttributeHeadersIgnoresDT(t *testing.T) {
	app := createTestApp()
	txn := app.StartTransaction("test")
	defer txn.End()

	hdrs := amqp.Table{
		"str": "hello",
	}

	injectDtHeaders(txn, hdrs)

	hdrStr, err := getHeadersAttributeString(hdrs)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(hdrStr)

	var v map[string]any
	err = json.Unmarshal([]byte(hdrStr), &v)
	if err != nil {
		t.Fatal(err)
	}

	if len(v) != 1 {
		t.Errorf("expected 1 key value pair, but got %d", len(v))
	}

	val, ok := v["str"]
	if !ok {
		t.Error("string header key value pair was dropped")
	} else if val.(string) != "hello" {
		t.Error("string header value was corrupted")
	}
}

func TestGetAttributeHeadersEmptyAfterStrippingDT(t *testing.T) {
	app := createTestApp()
	txn := app.StartTransaction("test")
	defer txn.End()

	hdrs := amqp.Table{}

	injectDtHeaders(txn, hdrs)

	hdrStr, err := getHeadersAttributeString(hdrs)
	if err != nil {
		t.Fatal(err)
	}

	if hdrStr != "" {
		t.Errorf("expected an empty header string, but got: %s", hdrStr)
	}
}
