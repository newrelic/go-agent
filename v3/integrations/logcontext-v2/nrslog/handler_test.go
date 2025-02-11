package nrslog

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

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

func TestWrap(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	handler := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})

	type test struct {
		name          string
		app           *newrelic.Application
		h             slog.Handler
		expectErr     error
		expectHandler *NRHandler
	}

	tests := []test{
		{
			name:          "nil app",
			app:           nil,
			h:             handler,
			expectErr:     ErrNilApp,
			expectHandler: nil,
		},
		{
			name:          "nil handler",
			app:           app.Application,
			h:             nil,
			expectErr:     ErrNilHandler,
			expectHandler: nil,
		},
		{
			name:          "duplicated handler",
			app:           app.Application,
			h:             &NRHandler{},
			expectErr:     ErrAlreadyWrapped,
			expectHandler: nil,
		},
		{
			name:      "valid",
			app:       app.Application,
			h:         handler,
			expectErr: nil,
			expectHandler: &NRHandler{
				app:     app.Application,
				handler: handler,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := Wrap(tt.app, tt.h)
			if err != tt.expectErr {
				t.Errorf("incorrect error return; expected: %v; got: %v", tt.expectErr, err)
			}
			if tt.expectHandler != nil {
				if h == nil {
					t.Errorf("expected handler to not be nil")
				}
				if tt.expectHandler.app != h.app {
					t.Errorf("expected: %v; got: %v", tt.expectHandler.app, h.app)
				}
				if tt.expectHandler.handler != h.handler {
					t.Errorf("expected: %v; got: %v", tt.expectHandler.handler, h.handler)
				}
			} else if h != nil {
				t.Errorf("expected handler to be nil")
			}
		})
	}
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

	txn := app.StartTransaction("hi")
	txnLog := WithTransaction(txn, log)
	txnLog.Info(message)
	data := txn.GetLinkingMetadata()
	txn.End()

	additionalAttrs := slog.String("additional", "attr")

	log = log.WithGroup("group1")
	log.Info(message, additionalAttrs)

	log = log.WithGroup("group2")
	log.Info(message, additionalAttrs)

	log = log.With(additionalAttrs)
	log.Info(message)

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Attributes: map[string]interface{}{
				"string key": "val",
				"int key":    1,
			},
			Severity:  slog.LevelInfo.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
		},
		{
			Attributes: map[string]interface{}{
				"string key": "val",
				"int key":    1,
			},
			Severity:  slog.LevelInfo.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
			SpanID:    data.SpanID,
			TraceID:   data.TraceID,
		},
		{
			Attributes: map[string]interface{}{
				"string key":        "val",
				"int key":           1,
				"group1.additional": "attr",
			},
			Severity:  slog.LevelInfo.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
		},
		{
			Attributes: map[string]interface{}{
				"string key":               "val",
				"int key":                  1,
				"group1.group2.additional": "attr",
			},
			Severity:  slog.LevelInfo.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
		},
		{
			Attributes: map[string]interface{}{
				"string key":               "val",
				"int key":                  1,
				"group1.group2.additional": "attr",
			},
			Severity:  slog.LevelInfo.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
		},
	})
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

	// purge the buffer
	writer.Reset()

	txn := app.StartTransaction("example transaction")
	ctx := newrelic.NewContext(context.Background(), txn)

	log.InfoContext(ctx, "I am a log inside a transaction with custom attributes!",
		slog.String("foo", "bar"),
		slog.Int("answer", 42),
	)
	metadata := txn.GetTraceMetadata()
	txn.End()

	logcontext.ValidateDecoratedOutput(t, writer, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		EntityName: integrationsupport.SampleAppName,
		Hostname:   host,
		TraceID:    metadata.TraceID,
		SpanID:     metadata.SpanID,
	})

	writer.Reset()

	gLog := log.WithGroup("group1")
	gLog.Info("I am a log message inside a group", slog.String("foo", "bar"), slog.Int("answer", 42))

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
				"foo":    "bar",
				"answer": 42,
			},
			TraceID: metadata.TraceID,
			SpanID:  metadata.SpanID,
		},
		{
			Severity:  slog.LevelInfo.String(),
			Message:   "I am a log message inside a group",
			Timestamp: internal.MatchAnyUnixMilli,
			Attributes: map[string]interface{}{
				"group1.foo":    "bar",
				"group1.answer": 42,
			},
		},
	})
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

func TestWithComplexAttributeOrGroup(t *testing.T) {
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

	log.With(attr).WithGroup("group3").With(slog.String("key3", "val3")).Info(message)

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
		{
			Severity:  slog.LevelInfo.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
			Attributes: map[string]interface{}{
				"group.key":         "val",
				"group.group2.key2": "val2",
				"group3.key3":       "val3",
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

func TestHandlerZeroTime(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	out := bytes.NewBuffer([]byte{})
	handler := WrapHandler(app.Application, slog.NewTextHandler(out, &slog.HandlerOptions{}))
	handler.Handle(context.Background(), slog.Record{
		Level:   slog.LevelInfo,
		Message: "Hello World!",
		Time:    time.Time{},
	})
	logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
	})
	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  slog.LevelInfo.String(),
			Message:   "Hello World!",
			Timestamp: internal.MatchAnyUnixMilli,
		},
	})
}

func BenchmarkDefaultHandler(b *testing.B) {
	handler := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	record := slog.Record{
		Time:    time.Now(),
		Message: "Hello World!",
		Level:   slog.LevelInfo,
	}

	ctx := context.Background()
	record.AddAttrs(slog.String("key", "val"), slog.Int("int", 1))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		handler.Handle(ctx, record)
	}
}

func BenchmarkHandler(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(false),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	handler, _ := Wrap(app.Application, slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	txn := app.Application.StartTransaction("my txn")
	defer txn.End()

	ctx := newrelic.NewContext(context.Background(), txn)

	record := slog.Record{
		Time:    time.Now(),
		Message: "Hello World!",
		Level:   slog.LevelInfo,
	}

	record.AddAttrs(slog.String("key", "val"), slog.Int("int", 1))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		handler.Handle(ctx, record)
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

	h, _ := Wrap(app.Application, slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	txn := app.Application.StartTransaction("my txn")
	defer txn.End()
	record := slog.Record{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		h.enrichRecord(app.Application, &record)
	}
}

func BenchmarkLinkingString(b *testing.B) {
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
	h, _ := Wrap(app.Application, slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	txn := app.Application.StartTransaction("my txn")
	defer txn.End()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		h.shouldEnrichLog(app.Application)
	}
}
