package nrzerolog

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"

	"github.com/rs/zerolog"
)

func newLogger(out io.Writer, app *newrelic.Application) zerolog.Logger {
	logger := zerolog.New(out)
	return logger.Hook(NewRelicHook{
		App: app,
	})
}

func newTxnLogger(out io.Writer, app *newrelic.Application, ctx context.Context) zerolog.Logger {
	logger := zerolog.New(out)
	return logger.Hook(NewRelicHook{
		App:     app,
		Context: ctx,
	})
}

func BenchmarkZerolog(b *testing.B) {
	log := zerolog.New(bytes.NewBuffer([]byte("")))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Info().Msg("This is a test log")
	}
}

func BenchmarkZerologLoggingDisabled(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogEnabled(false))
	log := newLogger(bytes.NewBuffer([]byte("")), app.Application)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Info().Msg("This is a test log")
	}
}

func BenchmarkZerologLogForwarding(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogForwardingEnabled(true))
	log := newLogger(bytes.NewBuffer([]byte("")), app.Application)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Info().Msg("This is a test log")
	}
}

/*

func BenchmarkFormattingWithOutTransaction(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	log := newLogger(bytes.NewBuffer([]byte("")), app.Application)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Info().Msg("Hello World!")
	}
}

func BenchmarkFormattingWithTransaction(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	txn := app.StartTransaction("TestLogDistributedTracingDisabled")
	defer txn.End()
	out := bytes.NewBuffer([]byte{})
	ctx := newrelic.NewContext(context.Background(), txn)
	log := newTxnLogger(out, app.Application, ctx)


	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.Info().Msg("Hello World!")
	}
}
*/

func TestBackgroundLog(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	out := bytes.NewBuffer([]byte{})
	log := newLogger(out, app.Application)
	message := "Hello World!"
	log.Info().Msg(message)

	// Un-comment when local decorating enabled
	/*
		logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
			EntityGUID: integrationsupport.TestEntityGUID,
			Hostname:   host,
			EntityName: integrationsupport.SampleAppName,
		})
	*/

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  zerolog.InfoLevel.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
		},
	})
}

func TestLogEmptyContext(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	out := bytes.NewBuffer([]byte{})
	log := newTxnLogger(out, app.Application, context.Background())
	message := "Hello World!"
	log.Info().Msg(message)

	// Un-comment when local decorating enabled
	/*
		logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
			EntityGUID: integrationsupport.TestEntityGUID,
			Hostname:   host,
			EntityName: integrationsupport.SampleAppName,
		}) */

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  zerolog.InfoLevel.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
		},
	})
}

func TestLogDebugLevel(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	out := bytes.NewBuffer([]byte{})
	log := newTxnLogger(out, app.Application, context.Background())
	message := "Hello World!"
	log.Print(message)

	// Un-comment when local decorating enabled
	/*
		logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
			EntityGUID: integrationsupport.TestEntityGUID,
			Hostname:   host,
			EntityName: integrationsupport.SampleAppName,
		}) */

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  zerolog.DebugLevel.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
		},
	})
}

func TestLogInContext(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	out := bytes.NewBuffer([]byte{})
	txn := app.StartTransaction("test txn")
	ctx := newrelic.NewContext(context.Background(), txn)
	log := newTxnLogger(out, app.Application, ctx)
	message := "Hello World!"
	log.Info().Msg(message)

	// Un-comment when local decorating enabled
	/*
		logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
			EntityGUID: integrationsupport.TestEntityGUID,
			Hostname:   host,
			EntityName: integrationsupport.SampleAppName,
			TraceID:    txn.GetLinkingMetadata().TraceID,
			SpanID:     txn.GetLinkingMetadata().SpanID,
		})
	*/

	txn.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  zerolog.InfoLevel.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
			SpanID:    txn.GetLinkingMetadata().SpanID,
			TraceID:   txn.GetLinkingMetadata().TraceID,
		},
	})

	txn.End()
}
