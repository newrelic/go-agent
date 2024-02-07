package nrslog

import (
	"bytes"
	"log/slog"
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
