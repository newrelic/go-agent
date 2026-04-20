package nrlogrus

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/logcontext"
	"github.com/newrelic/go-agent/v3/internal/sysinfo"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
	"github.com/sirupsen/logrus"
)

var (
	host, _ = sysinfo.Hostname()
)

type testEnricher struct {
	called bool
}

func (e *testEnricher) Enrich(buf *bytes.Buffer, opts newrelic.EnricherOption) error {
	e.called = true
	return nil
}
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

func TestBackgroundLogWithFields(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	out := bytes.NewBuffer([]byte{})
	log := newTextLogger(out, app.Application)
	message := "Hello World!"
	log.WithField("test field", []string{"a", "b"}).Info(message)
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
			Attributes: map[string]interface{}{
				"test field": []string{"a", "b"},
			},
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

func TestLogInContextWithFields(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	out := bytes.NewBuffer([]byte{})
	log := newTextLogger(out, app.Application)
	txn := app.StartTransaction("test txn")

	ctx := newrelic.NewContext(context.Background(), txn)
	message := "Hello World!"
	log.WithField("hi", 1).WithContext(ctx).Info(message)

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
			Attributes: map[string]interface{}{
				"hi": 1,
			},
		},
	})

	txn.End()
}

func TestContextFormatter_enrichLog(t *testing.T) {
	// do I need to test different types of formatters? Probably should
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		formatter logrus.Formatter
		// Named input parameters for target function.
		txn *newrelic.Transaction

		enabled                bool
		localDecoratingEnabled bool

		wantCallSpy bool
	}{
		{
			name:                   "Logging and Local Decorating Disabled and existing txn. Should not call Enrich Log",
			formatter:              &logrus.TextFormatter{},
			txn:                    &newrelic.Transaction{},
			enabled:                false,
			localDecoratingEnabled: false,
			wantCallSpy:            false,
		},
		{
			name:                   "Logging and Local Decorating Disabled and no existing txn. Should not call Enrich Log",
			formatter:              &logrus.TextFormatter{},
			txn:                    nil,
			enabled:                false,
			localDecoratingEnabled: false,
			wantCallSpy:            false,
		},
		{
			name:                   "Logging Enabled and Local Decorating Disabled and existing txn. Should not call Enrich Log",
			formatter:              &logrus.TextFormatter{},
			txn:                    &newrelic.Transaction{},
			enabled:                true,
			localDecoratingEnabled: false,
			wantCallSpy:            false,
		},
		{
			name:                   "Logging Enabled and Local Decorating Disabled and no existing txn. Should not call Enrich Log",
			formatter:              &logrus.TextFormatter{},
			txn:                    nil,
			enabled:                true,
			localDecoratingEnabled: false,
			wantCallSpy:            false,
		},
		{
			name:                   "Logging Disabled and Local Decorating Enabled and existing txn. Should not call Enrich Log",
			formatter:              &logrus.TextFormatter{},
			txn:                    &newrelic.Transaction{},
			enabled:                false,
			localDecoratingEnabled: true,
			wantCallSpy:            false,
		},
		{
			name:                   "Logging Disabled and Local Decorating Enabled and no existing txn. Should not call Enrich Log",
			formatter:              &logrus.TextFormatter{},
			txn:                    nil,
			enabled:                false,
			localDecoratingEnabled: true,
			wantCallSpy:            false,
		},
		{
			name:                   "Logging and Local Decorating Enabled and no existing txn. Should call enrich log",
			formatter:              &logrus.TextFormatter{},
			txn:                    nil,
			enabled:                true,
			localDecoratingEnabled: true,
			wantCallSpy:            true,
		},
		{
			name:                   "Logging and Local Decorating Enabled and existing txn. Should call enrich log",
			formatter:              &logrus.TextFormatter{},
			txn:                    &newrelic.Transaction{},
			enabled:                true,
			localDecoratingEnabled: true,
			wantCallSpy:            true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enricherSpy := &testEnricher{}
			f := ContextFormatter{
				formatter: tt.formatter,
				enricher:  enricherSpy,
			} // not testing any app functionality so we can set it to nil in this case
			f.enrichLog(nil, tt.txn, newrelic.Config{
				ApplicationLogging: newrelic.ApplicationLogging{
					Enabled: tt.enabled,
					LocalDecorating: struct {
						Enabled            bool
						WithinMessageField bool
					}{
						Enabled: tt.localDecoratingEnabled,
					},
				},
			})

			if enricherSpy.called != tt.wantCallSpy {
				t.Errorf("enrichLog() failed with calling newrelic.Enrich(), Got: %v want: %v", enricherSpy.called, tt.wantCallSpy)
			}

		})
	}
}
