package nrslog

import (
	"bytes"
	"context"
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

	/*
		logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
			EntityGUID: integrationsupport.TestEntityGUID,
			Hostname:   host,
			EntityName: integrationsupport.SampleAppName,
		}) */

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

func TestWithGroup(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(false),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	out := bytes.NewBuffer([]byte{})
	handler := TextHandler(app.Application, out, &slog.HandlerOptions{})
	log := slog.New(handler)
	message := "Hello World!"
	log = log.With(slog.Group("test group", slog.String("string key", "val")))
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
