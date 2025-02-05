package nrslog

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/internal/logcontext"
	"github.com/newrelic/go-agent/v3/internal/sysinfo"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var (
	host, _ = sysinfo.Hostname()
)

func TestHandler(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	out := bytes.NewBuffer([]byte{})
	handler := TextHandler(app.Application, out, &slog.HandlerOptions{})
	log := slog.New(handler)
	message := "Hello World!"
	log.Info(message)
	logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
	})
	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  slog.LevelInfo.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
		},
	})
}

func TestHandlerNilApp(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	logger := New(nil, slog.NewTextHandler(out, &slog.HandlerOptions{}))
	message := "Hello World!"
	logger.Info(message)

	logStr := out.String()
	if strings.Contains(logStr, nrlinking) {
		t.Errorf(" %s should not contain %s", logStr, nrlinking)
	}
	if len(logStr) == 0 {
		t.Errorf("log string should not be empty")
	}
}

func TestJSONHandler(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	out := bytes.NewBuffer([]byte{})
	handler := JSONHandler(app.Application, out, &slog.HandlerOptions{})
	log := slog.New(handler)
	message := "Hello World!"
	log.Info(message)
	logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
	})
	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  slog.LevelInfo.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
		},
	})
}

func TestHandlerTransactions(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	out := bytes.NewBuffer([]byte{})
	message := "Hello World!"

	handler := TextHandler(app.Application, out, &slog.HandlerOptions{})
	log := slog.New(handler)

	txn := app.Application.StartTransaction("my txn")
	txninfo := txn.GetLinkingMetadata()

	txnLogger := WithTransaction(txn, log)
	txnLogger.Info(message)

	backgroundMsg := "this is a background message"
	log.Debug(backgroundMsg)
	txn.End()

	logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
	})

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  slog.LevelInfo.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
			SpanID:    txninfo.SpanID,
			TraceID:   txninfo.TraceID,
		},
	})
}

func TestHandlerTransactionCtx(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	out := bytes.NewBuffer([]byte{})
	message := "Hello World!"

	handler := TextHandler(app.Application, out, &slog.HandlerOptions{})
	log := slog.New(handler)

	txn := app.Application.StartTransaction("my txn")
	ctx := newrelic.NewContext(context.Background(), txn)
	txninfo := txn.GetLinkingMetadata()

	txnLogger := WithContext(ctx, log)
	txnLogger.Info(message)

	backgroundMsg := "this is a background message"
	log.Debug(backgroundMsg)
	txn.End()

	logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
	})

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  slog.LevelInfo.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
			SpanID:    txninfo.SpanID,
			TraceID:   txninfo.TraceID,
		},
	})
}

func TestHandlerTransactionsAndBackground(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	out := bytes.NewBuffer([]byte{})
	message := "Hello World!"
	messageTxn := "Hello Transaction!"
	messageBackground := "Hello Background!"

	handler := TextHandler(app.Application, out, &slog.HandlerOptions{})
	log := slog.New(handler)

	log.Info(message)

	txn := app.Application.StartTransaction("my txn")
	txninfo := txn.GetLinkingMetadata()

	txnLogger := WithTransaction(txn, log)
	txnLogger.Info(messageTxn)

	log.Warn(messageBackground)
	txn.End()

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  slog.LevelInfo.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
		},
		{
			Severity:  slog.LevelWarn.String(),
			Message:   messageBackground,
			Timestamp: internal.MatchAnyUnixMilli,
		},
		{
			Severity:  slog.LevelInfo.String(),
			Message:   messageTxn,
			Timestamp: internal.MatchAnyUnixMilli,
			SpanID:    txninfo.SpanID,
			TraceID:   txninfo.TraceID,
		},
	})
}

func TestWithAttributes(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(false),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	out := bytes.NewBuffer([]byte{})
	handler := TextHandler(app.Application, out, &slog.HandlerOptions{})
	log := slog.New(handler)
	message := "Hello World!"
	log = log.With(slog.String("string key", "val"), slog.Int("int key", 1))

	log.Info(message)

	log1 := string(out.String())

	txn := app.StartTransaction("hi")
	txnLog := WithTransaction(txn, log)
	txnLog.Info(message)
	txn.End()

	log2 := string(out.String())

	attrString := `"string key"=val "int key"=1`
	if !strings.Contains(log1, attrString) {
		t.Errorf("expected %s to contain %s", log1, attrString)
	}

	if !strings.Contains(log2, attrString) {
		t.Errorf("expected %s to contain %s", log2, attrString)
	}

}

func TestWithAttributesFromContext(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	writer := &bytes.Buffer{}
	log := New(app.Application, slog.NewTextHandler(writer, &slog.HandlerOptions{}))

	log.Info("I am a log message")
	logcontext.ValidateDecoratedOutput(t, writer, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		EntityName: integrationsupport.SampleAppName,
		Hostname:   host,
	})

	logcontext.ValidateDecoratedOutput(t, writer, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		EntityName: integrationsupport.SampleAppName,
		Hostname:   host,
	})

	// purge the buffer
	writer.Reset()

	txn := app.StartTransaction("example transaction")
	ctx := newrelic.NewContext(context.Background(), txn)

	log.InfoContext(ctx, "I am a log inside a transaction with custom attributes!",
		slog.String("foo", "bar"),
		slog.Int("answer", 42),
		slog.Any("some_map", map[string]interface{}{"a": 1.0, "b": 2}),
	)
	metadata := txn.GetTraceMetadata()
	txn.End()

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  slog.LevelInfo.String(),
			Message:   "I am a log message",
			Timestamp: internal.MatchAnyUnixMilli,
		},
		{
			Severity:  slog.LevelInfo.String(),
			Message:   "I am a log inside a transaction with custom attributes!",
			Timestamp: internal.MatchAnyUnixMilli,
			Attributes: map[string]interface{}{
				"foo":      "bar",
				"answer":   42,
				"some_map": map[string]interface{}{"a": 1.0, "b": 2},
			},
			TraceID: metadata.TraceID,
			SpanID:  metadata.SpanID,
		},
	})

	logcontext.ValidateDecoratedOutput(t, writer, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		EntityName: integrationsupport.SampleAppName,
		Hostname:   host,
		TraceID:    metadata.TraceID,
		SpanID:     metadata.SpanID,
	})
}

func TestWithGroup(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(false),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	out := bytes.NewBuffer([]byte{})
	handler := TextHandler(app.Application, out, &slog.HandlerOptions{})
	log := slog.New(handler)
	message := "Hello World!"
	group := slog.Group("test group", slog.String("string key", "val"))
	log = log.With(group)
	log = log.WithGroup("test group")

	log.Info(message)

	log1 := string(out.String())

	txn := app.StartTransaction("hi")
	txnLog := WithTransaction(txn, log)
	txnLog.Info(message)
	txn.End()

	log2 := string(out.String())

	attrString := `"test group.string key"=val`
	if !strings.Contains(log1, attrString) {
		t.Errorf("expected %s to contain %s", log1, attrString)
	}

	if !strings.Contains(log2, attrString) {
		t.Errorf("expected %s to contain %s", log2, attrString)
	}
}

// Ensure deprecation compatibility
func TestDeprecatedWithTransactionFromContext(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	out := bytes.NewBuffer([]byte{})
	message := "Hello World!"

	handler := TextHandler(app.Application, out, &slog.HandlerOptions{})
	log := slog.New(WithTransactionFromContext(handler))

	txn := app.Application.StartTransaction("my txn")
	ctx := newrelic.NewContext(context.Background(), txn)
	txninfo := txn.GetLinkingMetadata()

	log.InfoContext(ctx, message)

	txn.End()

	logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
	})

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  slog.LevelInfo.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
			SpanID:    txninfo.SpanID,
			TraceID:   txninfo.TraceID,
		},
	})
}

func TestAttributeCapture(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	message := "Hello World!"
	attr := slog.Group("group", slog.String("key", "val"), slog.Group("group2", slog.String("key2", "val2")))
	log := New(app.Application, slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

	log.Info(message, attr)
	fooLog := log.WithGroup("foo")
	fooLog.Info(message, attr)

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  slog.LevelInfo.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
			Attributes: map[string]interface{}{
				"group.key":         "val",
				"group.group2.key2": "val2",
			},
		},
		{
			Severity:  slog.LevelInfo.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
			Attributes: map[string]interface{}{
				"foo.group.key":         "val",
				"foo.group.group2.key2": "val2",
			},
		},
	})
}

func TestAppendAttr(t *testing.T) {
	h := &NRHandler{}
	nrAttrs := map[string]interface{}{}

	attr := slog.Group("group", slog.String("key", "val"), slog.Group("group2", slog.String("key2", "val2")))
	h.appendAttr(nrAttrs, attr, "")
	if len(nrAttrs) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(nrAttrs))
	}

	entry1, ok := nrAttrs["group.key"]
	if !ok {
		t.Errorf("expected group.key to be in the map")
	}
	if entry1 != "val" {
		t.Errorf("expected value of 'group.key' to be val, got '%s'", entry1)
	}

	entry2, ok := nrAttrs["group.group2.key2"]
	if !ok {
		t.Errorf("expected group.group2.key2 to be in the map")
	}
	if entry2 != "val2" {
		t.Errorf("expected value of 'group.group2.key2' to be val2, got '%s'", entry2)
	}
}

func TestAppendAttrWithGroupPrefix(t *testing.T) {
	h := &NRHandler{}
	nrAttrs := map[string]interface{}{}

	attr := slog.Group("group", slog.String("key", "val"), slog.Group("group2", slog.String("key2", "val2")))
	h.appendAttr(nrAttrs, attr, "prefix")

	if len(nrAttrs) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(nrAttrs))
	}

	entry1, ok := nrAttrs["prefix.group.key"]
	if !ok {
		t.Errorf("expected group.key to be in the map")
	}
	if entry1 != "val" {
		t.Errorf("expected value of 'group.key' to be val, got '%s'", entry1)
	}

	entry2, ok := nrAttrs["prefix.group.group2.key2"]
	if !ok {
		t.Errorf("expected group.group2.key2 to be in the map")
	}
	if entry2 != "val2" {
		t.Errorf("expected value of 'group.group2.key2' to be val2, got '%s'", entry2)
	}
}

// the maps are costing so much here
func BenchmarkAppendAttribute(b *testing.B) {
	h := &NRHandler{}
	nrAttrs := map[string]interface{}{}

	attr := slog.Group("group", slog.String("key", "val"), slog.Group("group2", slog.String("key2", "val2")))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		h.appendAttr(nrAttrs, attr, "")
	}
}

func BenchmarkEnrichLog(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	txn := app.Application.StartTransaction("my txn")
	defer txn.End()
	record := slog.Record{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		nrLinking := bytes.NewBuffer([]byte{})
		err := newrelic.EnrichLog(nrLinking, newrelic.FromTxn(txn))
		if err == nil {
			record.AddAttrs(slog.String("newrelic", nrLinking.String()))
		}
	}
}

func BenchmarkLinkingStringEnrichment(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	txn := app.Application.StartTransaction("my txn")
	defer txn.End()
	record := slog.Record{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		enrichRecordTxn(txn, &record)
	}
}

func BenchmarkStringBuilder(b *testing.B) {
	md := newrelic.LinkingMetadata{
		EntityGUID: "entityGUID",
		Hostname:   "hostname",
		TraceID:    "traceID",
		SpanID:     "spanID",
		EntityName: "entityName",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		nrLinkingString(md)
	}
}

func BenchmarkShouldEnrichLog(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	txn := app.Application.StartTransaction("my txn")
	defer txn.End()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		shouldEnrichLog(app.Application)
	}
}
