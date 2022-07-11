package nrlogrus

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
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

type expectVals struct {
	decorationDisabled bool
	entityGUID         string
	entityName         string
	hostname           string
	traceID            string
	spanID             string
}

// metadata indexes
const (
	entityguid = 1
	hostname   = 2
	traceid    = 3
	spanid     = 4
)

func entityname(vals []string) string {
	if len(vals) < 2 {
		return ""
	}

	return vals[len(vals)-2]
}

func validateOutput(t *testing.T, out *bytes.Buffer, expect *expectVals) {
	actual := out.String()
	split := strings.Split(actual, "NR-LINKING")

	if expect.decorationDisabled && len(split) != 2 {
		t.Errorf("expected log decoration, but NR-LINKING data was missing: %s", actual)
	}

	linkingData := strings.Split(split[1], "|")

	if len(linkingData) < 5 {
		t.Errorf("linking data is missing required fields: %s", split[1])
	}

	if linkingData[entityguid] != expect.entityGUID {
		t.Errorf("incorrect entity GUID; expect: %s actual: %s", expect.entityGUID, linkingData[entityguid])
	}

	if linkingData[hostname] != expect.hostname {
		t.Errorf("incorrect hostname; expect: %s actual: %s", expect.hostname, linkingData[hostname])
	}

	if entityname(linkingData) != expect.entityName {
		t.Errorf("incorrect entity name; expect: %s actual: %s", expect.entityName, entityname(linkingData))
	}

	if expect.traceID != "" && expect.spanID != "" {
		if len(linkingData) < 7 {
			t.Errorf("transaction metadata is missing from linking data: %s", split[1])
		}

		if linkingData[traceid] != expect.traceID {
			t.Errorf("incorrect traceID; expect: %s actual: %s", expect.traceID, linkingData[traceid])
		}

		if linkingData[spanid] != expect.spanID {
			t.Errorf("incorrect hostname; expect: %s actual: %s", expect.spanID, linkingData[spanid])
		}
	}
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

func BenchmarkTextFormatter(b *testing.B) {
	log := newTextLogger(bytes.NewBuffer([]byte("")), nil)
	log.Formatter = new(logrus.TextFormatter)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.WithContext(ctx).Info("Hello World!")
	}
}

func BenchmarkWithOutTransaction(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	log := newTextLogger(bytes.NewBuffer([]byte("")), app.Application)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.WithContext(ctx).Info("Hello World!")
	}
}

func BenchmarkWithTransaction(b *testing.B) {
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
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	out := bytes.NewBuffer([]byte{})
	log := newTextLogger(out, app.Application)
	log.Info("Hello World!")
	validateOutput(t, out, &expectVals{
		entityGUID: integrationsupport.TestEntityGUID,
		hostname:   host,
		entityName: integrationsupport.SampleAppName,
	})
}

func TestJSONBackgroundLog(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	out := bytes.NewBuffer([]byte{})
	log := newJSONLogger(out, app.Application)
	log.Info("Hello World!")
	validateOutput(t, out, &expectVals{
		entityGUID: integrationsupport.TestEntityGUID,
		hostname:   host,
		entityName: integrationsupport.SampleAppName,
	})
}

func TestLogEmptyContext(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	out := bytes.NewBuffer([]byte{})
	log := newTextLogger(out, app.Application)
	log.WithContext(context.Background()).Info("Hello World!")
	validateOutput(t, out, &expectVals{
		entityGUID: integrationsupport.TestEntityGUID,
		hostname:   host,
		entityName: integrationsupport.SampleAppName,
	})
}

func TestLogInContext(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	out := bytes.NewBuffer([]byte{})
	log := newTextLogger(out, app.Application)
	txn := app.StartTransaction("test txn")
	defer txn.End()

	ctx := newrelic.NewContext(context.Background(), txn)
	log.WithContext(ctx).Info("Hello World!")

	validateOutput(t, out, &expectVals{
		entityGUID: integrationsupport.TestEntityGUID,
		hostname:   host,
		entityName: integrationsupport.SampleAppName,
		traceID:    txn.GetLinkingMetadata().TraceID,
		spanID:     txn.GetLinkingMetadata().SpanID,
	})
}
