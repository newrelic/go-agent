package nrlogrus

import (
	"bytes"
	"context"
	"fmt"
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

type calledChecker interface {
	WasCalled() bool
}
type testEnricher struct {
	called bool
	err    error
}

func (e *testEnricher) Enrich(buf *bytes.Buffer, opts newrelic.EnricherOption) error {
	e.called = true
	return e.ErrOrNil()
}

func (e *testEnricher) WasCalled() bool {
	return e.called
}

func (e *testEnricher) ErrOrNil() error {
	return e.err
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
		// Named input parameters for target function.
		txn                    *newrelic.Transaction
		enabled                bool
		localDecoratingEnabled bool
		wantCallSpy            bool
	}{
		{
			name:                   "Logging and Local Decorating Disabled and existing txn. Should not call Enrich Log",
			txn:                    &newrelic.Transaction{},
			enabled:                false,
			localDecoratingEnabled: false,
			wantCallSpy:            false,
		},
		{
			name:                   "Logging and Local Decorating Disabled and no existing txn. Should not call Enrich Log",
			txn:                    nil,
			enabled:                false,
			localDecoratingEnabled: false,
			wantCallSpy:            false,
		},
		{
			name:                   "Logging Enabled and Local Decorating Disabled and existing txn. Should not call Enrich Log",
			txn:                    &newrelic.Transaction{},
			enabled:                true,
			localDecoratingEnabled: false,
			wantCallSpy:            false,
		},
		{
			name:                   "Logging Enabled and Local Decorating Disabled and no existing txn. Should not call Enrich Log",
			txn:                    nil,
			enabled:                true,
			localDecoratingEnabled: false,
			wantCallSpy:            false,
		},
		{
			name:                   "Logging Disabled and Local Decorating Enabled and existing txn. Should not call Enrich Log",
			txn:                    &newrelic.Transaction{},
			enabled:                false,
			localDecoratingEnabled: true,
			wantCallSpy:            false,
		},
		{
			name:                   "Logging Disabled and Local Decorating Enabled and no existing txn. Should not call Enrich Log",
			txn:                    nil,
			enabled:                false,
			localDecoratingEnabled: true,
			wantCallSpy:            false,
		},
		{
			name:                   "Logging and Local Decorating Enabled and no existing txn. Should call enrich log",
			txn:                    nil,
			enabled:                true,
			localDecoratingEnabled: true,
			wantCallSpy:            true,
		},
		{
			name:                   "Logging and Local Decorating Enabled and existing txn. Should call enrich log",
			txn:                    &newrelic.Transaction{},
			enabled:                true,
			localDecoratingEnabled: true,
			wantCallSpy:            true,
		},
	}
	formatters := map[string]logrus.Formatter{
		"Text": &logrus.TextFormatter{},
		"JSON": &logrus.JSONFormatter{},
	}
	for _, tt := range tests {
		for key, formatter := range formatters {
			testName := fmt.Sprintf("%s: tt.name", key)
			t.Run(testName, func(t *testing.T) {
				enricherSpy := &testEnricher{}
				f := ContextFormatter{
					formatter: formatter,
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
}

func TestContextFormatter_Format(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		jsonOnly                   bool
		enricher                   logEnricher
		appInitialized             bool
		logForwardingEnabled       bool
		localDecoratingEnabled     bool
		logDecoratingWithinMessage bool
		// Named input parameters for target function.
		e           *logrus.Entry
		want        []byte
		wantErr     bool
		wantCallSpy bool
	}{
		{
			name:                       "Couldn't retrieve app config and all log enabling set to true. Should return with error and nil bytes.",
			appInitialized:             false,
			logForwardingEnabled:       true,
			localDecoratingEnabled:     true,
			logDecoratingWithinMessage: true,
			enricher:                   &testEnricher{},
			e:                          &logrus.Entry{},
			want:                       nil,
			wantErr:                    true,
			wantCallSpy:                false,
		},
		{
			name:                       "Couldn't retrieve app config and all log enabling set to false. Should return with error and nil bytes.",
			appInitialized:             false,
			logForwardingEnabled:       false,
			localDecoratingEnabled:     false,
			logDecoratingWithinMessage: false,
			enricher:                   &testEnricher{},

			e:           &logrus.Entry{},
			want:        nil,
			wantErr:     true,
			wantCallSpy: false,
		},
		{
			name:                       "Couldn't retrieve app config and some log enabling set to false. Should return with error and nil bytes.",
			appInitialized:             false,
			logForwardingEnabled:       false,
			localDecoratingEnabled:     true,
			logDecoratingWithinMessage: false,
			enricher:                   &testEnricher{},

			e:           &logrus.Entry{},
			want:        nil,
			wantErr:     true,
			wantCallSpy: false,
		},
		{
			name:                       "Couldn't retrieve app config and some log enabling set to true. Should return with error and nil bytes.",
			appInitialized:             false,
			logForwardingEnabled:       true,
			localDecoratingEnabled:     true,
			logDecoratingWithinMessage: false,
			enricher:                   &testEnricher{},
			e:                          &logrus.Entry{},
			want:                       nil,
			wantErr:                    true,
			wantCallSpy:                false,
		},
		{
			name:                       "Log decorating within message set to true but enrich log returns error. Should return with error and nil bytes.",
			appInitialized:             true,
			logForwardingEnabled:       true,
			localDecoratingEnabled:     true,
			logDecoratingWithinMessage: true,
			enricher:                   &testEnricher{err: fmt.Errorf("test error")},
			e:                          &logrus.Entry{},
			want:                       nil,
			wantErr:                    true,
			wantCallSpy:                true,
		},
		{
			name:                       "Log decorating within message set to true but others set to false but enrich log returns error. Should return with nil bytes.",
			appInitialized:             true,
			logForwardingEnabled:       true,
			localDecoratingEnabled:     true,
			logDecoratingWithinMessage: true,
			enricher:                   &testEnricher{err: fmt.Errorf("test error")},
			e:                          &logrus.Entry{},
			want:                       nil,
			wantErr:                    true,
			wantCallSpy:                true,
		},
		{
			name:                       "Log decorating within message set to true and enrich log returns nil. Format returns an error (JSON only). Should return with nil bytes but should enrich.",
			jsonOnly:                   true,
			appInitialized:             true,
			logForwardingEnabled:       true,
			localDecoratingEnabled:     true,
			logDecoratingWithinMessage: true,
			enricher:                   &testEnricher{},
			e: &logrus.Entry{
				Data: logrus.Fields{"fn": func() {}}, // json encode won't work in JSONFormatter.Format()
			},
			want:        nil,
			wantErr:     true,
			wantCallSpy: true,
		},
		{
			name:                       "Log decorating within message set to true and enrich log returns err. Format returns an error (JSON only). Should return with nil bytes but should enrich.",
			jsonOnly:                   true,
			appInitialized:             true,
			logForwardingEnabled:       true,
			localDecoratingEnabled:     true,
			logDecoratingWithinMessage: true,
			enricher:                   &testEnricher{err: fmt.Errorf("test error")},
			e: &logrus.Entry{
				Data: logrus.Fields{"fn": func() {}}, // json encode won't work in JSONFormatter.Format()
			},
			want:        nil,
			wantErr:     true,
			wantCallSpy: true,
		},
		{
			name:                       "Log decorating within message set to false and enrich log returns nil. Format returns an error (JSON only). Should return with nil bytes.",
			jsonOnly:                   true,
			appInitialized:             true,
			logForwardingEnabled:       true,
			localDecoratingEnabled:     true,
			logDecoratingWithinMessage: false,
			enricher:                   &testEnricher{},
			e: &logrus.Entry{
				Data: logrus.Fields{"fn": func() {}}, // json encode won't work in JSONFormatter.Format()
			},
			want:        nil,
			wantErr:     true,
			wantCallSpy: false,
		},
		{
			name:                       "Log decorating within message set to false and enrich log returns err. Format returns an error (JSON only). Should return with nil bytes.",
			jsonOnly:                   true,
			appInitialized:             true,
			logForwardingEnabled:       true,
			localDecoratingEnabled:     true,
			logDecoratingWithinMessage: false,
			enricher:                   &testEnricher{err: fmt.Errorf("test error")},
			e: &logrus.Entry{
				Data: logrus.Fields{"fn": func() {}}, // json encode won't work in JSONFormatter.Format()
			},
			want:        nil,
			wantErr:     true,
			wantCallSpy: false,
		},
		{
			name:                       "Log decorating within message set to false and enrich log returns err. Should return with nil bytes and should call enrich.",
			appInitialized:             true,
			logForwardingEnabled:       true,
			localDecoratingEnabled:     true,
			logDecoratingWithinMessage: false,
			enricher:                   &testEnricher{err: fmt.Errorf("test error")},
			e:                          &logrus.Entry{},
			want:                       nil,
			wantErr:                    true,
			wantCallSpy:                true,
		},
		{
			name:                       "Log decorating within message set to false and enrich log returns nil. Should return with bytes and should call enrich.",
			appInitialized:             true,
			logForwardingEnabled:       true,
			localDecoratingEnabled:     true,
			logDecoratingWithinMessage: false,
			enricher:                   &testEnricher{},
			e:                          &logrus.Entry{},
			want:                       []byte{},
			wantErr:                    false,
			wantCallSpy:                true,
		},
		{
			name:                       "Log decorating within message set to true and enrich log returns nil. Should return with bytes and should call enrich.",
			appInitialized:             true,
			logForwardingEnabled:       true,
			localDecoratingEnabled:     true,
			logDecoratingWithinMessage: true,
			enricher:                   &testEnricher{},
			e:                          &logrus.Entry{},
			want:                       []byte{},
			wantErr:                    false,
			wantCallSpy:                true,
		},
		{
			name:                       "App initialized with local decorating disabled. Should return with bytes and should not call enrich.",
			appInitialized:             true,
			logForwardingEnabled:       true,
			localDecoratingEnabled:     false,
			logDecoratingWithinMessage: false,
			enricher:                   &testEnricher{},
			e:                          &logrus.Entry{},
			want:                       []byte{},
			wantErr:                    false,
			wantCallSpy:                false,
		},
	}
	formatters := map[string]logrus.Formatter{
		"Text": &logrus.TextFormatter{},
		"JSON": &logrus.JSONFormatter{},
	}
	for _, tt := range tests {
		for key, formatter := range formatters {
			if tt.jsonOnly && key == "Text" {
				continue
			}
			testName := fmt.Sprintf("%s: %s", key, tt.name)
			t.Run(testName, func(t *testing.T) {
				app := buildApp(tt.appInitialized, tt.logForwardingEnabled, tt.localDecoratingEnabled, tt.logDecoratingWithinMessage)
				f := &ContextFormatter{
					app:       app.Application,
					formatter: formatter,
					enricher:  tt.enricher,
				}
				got, gotErr := f.Format(tt.e)
				if gotErr != nil {
					if !tt.wantErr {
						t.Errorf("Format() failed: %v", gotErr)
					}
				} else if tt.wantErr {
					t.Errorf("Format() succeeded unexpectedly: %v", key)
				}
				if tt.want != nil {
					if len(got) == 0 {
						t.Errorf("Unexpected nil return -> Format() = %v, want %v", got, tt.want)
					}
				} else if len(got) > 0 {
					t.Errorf("Unexpected non-nil return -> Format() = %v, want %v", got, tt.want)
				}
				if checker, ok := tt.enricher.(calledChecker); ok {
					if tt.wantCallSpy {
						if !checker.WasCalled() {
							t.Errorf("Unexpected non-call of Enrich()")
						}
					} else if checker.WasCalled() {
						t.Errorf("Unexpected call of Enrich()")
					}
				}
			})
		}
	}
}

func buildApp(appInitialized bool, logForwardingEnabled, localDecoratingEnabled, logDecoratingWithinMessage bool) integrationsupport.ExpectApp {
	if !appInitialized {
		return integrationsupport.ExpectApp{
			Application: &newrelic.Application{},
		}
	}
	return integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(localDecoratingEnabled),
		newrelic.ConfigAppLogForwardingEnabled(logForwardingEnabled),
		newrelic.ConfigAppLogDecoratingWithinMessage(logDecoratingWithinMessage),
	)
}
