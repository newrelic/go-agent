package nrlogrus

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/internal/logcontext"
	"github.com/newrelic/go-agent/v3/internal/sysinfo"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sirupsen/logrus"
)

var (
	host, _ = sysinfo.Hostname()
)

func newTextLogger(out io.Writer, app *newrelic.Application) *logrus.Logger {
	l := logrus.New()
	l.Formatter = NewFormatter(app, &logrus.TextFormatter{
		DisableColors: true,
	})
	l.SetReportCaller(true)
	l.SetOutput(out)
	return l
}

func newJSONLogger(out io.Writer, app *newrelic.Application) *logrus.Logger {
	l := logrus.New()
	l.Formatter = NewFormatter(app, &logrus.JSONFormatter{})
	l.SetReportCaller(true)
	l.SetOutput(out)
	return l
}

func BenchmarkFormatterLogic(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	formatter := NewFormatter(app.Application, &logrus.TextFormatter{})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := formatter.Format(logrus.New().WithContext(context.Background()))
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkLogrusTextFormatter(b *testing.B) {
	log := newTextLogger(bytes.NewBuffer([]byte("")), nil)
	log.Formatter = new(logrus.TextFormatter)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.WithContext(ctx).Info("Hello World!")
	}
}

func BenchmarkFormattingWithOutTransaction(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	log := newTextLogger(bytes.NewBuffer([]byte("")), app.Application)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.WithContext(ctx).Info("Hello World!")
	}
}

func BenchmarkFormattingWithTransaction(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	txn := app.StartTransaction("TestLogDistributedTracingDisabled")
	out := bytes.NewBuffer([]byte{})
	log := newTextLogger(out, app.Application)
	ctx := newrelic.NewContext(context.Background(), txn)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.WithContext(ctx).Info("Hello World!")
	}
}

func TestBackgroundLog(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	out := bytes.NewBuffer([]byte{})
	log := newTextLogger(out, app.Application)
	message := "Hello World!"
	log.Info(message)
	logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
	})
	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  logrus.InfoLevel.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
		},
	})
}

func TestJSONBackgroundLog(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	out := bytes.NewBuffer([]byte{})
	log := newJSONLogger(out, app.Application)
	message := "Hello World!"
	log.Info(message)
	logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
	})
	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  logrus.InfoLevel.String(),
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
	log := newTextLogger(out, app.Application)
	message := "Hello World!"
	log.WithContext(context.Background()).Info(message)
	logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
	})
	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  logrus.InfoLevel.String(),
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
	log := newTextLogger(out, app.Application)
	txn := app.StartTransaction("test txn")

	ctx := newrelic.NewContext(context.Background(), txn)
	message := "Hello World!"
	log.WithContext(ctx).Info(message)

	logcontext.ValidateDecoratedOutput(t, out, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
		TraceID:    txn.GetLinkingMetadata().TraceID,
		SpanID:     txn.GetLinkingMetadata().SpanID,
	})
	txn.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  logrus.InfoLevel.String(),
			Message:   message,
			Timestamp: internal.MatchAnyUnixMilli,
			SpanID:    txn.GetLinkingMetadata().SpanID,
			TraceID:   txn.GetLinkingMetadata().TraceID,
		},
	})

	txn.End()
}
