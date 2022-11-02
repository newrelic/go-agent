// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrlogrusplugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/internal/sysinfo"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sirupsen/logrus"
)

var (
	testTime      = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	matchAnything = struct{}{}
	host, _       = sysinfo.Hostname()
)

func newTestLogger(out io.Writer) *logrus.Logger {
	l := logrus.New()
	l.Formatter = ContextFormatter{}
	l.SetReportCaller(true)
	l.SetOutput(out)
	return l
}

func validateOutput(t *testing.T, out *bytes.Buffer, expected map[string]interface{}) {

	var actual map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &actual); nil != err {
		t.Fatal("failed to unmarshal log output:", err)
	}
	for k, v := range expected {
		found, ok := actual[k]
		if !ok {
			t.Errorf("key %s not found:\nactual=%s", k, actual)
		}
		if v != matchAnything && found != v {
			t.Errorf("value for key %s is incorrect:\nactual=%s\nexpected=%s", k, found, v)
		}
	}
	for k, v := range actual {
		if _, ok := expected[k]; !ok {
			t.Errorf("unexpected key found:\nkey=%s\nvalue=%s", k, v)
		}
	}
}

func BenchmarkWithOutTransaction(b *testing.B) {
	log := newTestLogger(bytes.NewBuffer([]byte("")))
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.WithContext(ctx).Info("Hello World!")
	}
}

func BenchmarkJSONFormatter(b *testing.B) {
	log := newTestLogger(bytes.NewBuffer([]byte("")))
	log.Formatter = new(logrus.JSONFormatter)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.WithContext(ctx).Info("Hello World!")
	}
}

func BenchmarkTextFormatter(b *testing.B) {
	log := newTestLogger(bytes.NewBuffer([]byte("")))
	log.Formatter = new(logrus.TextFormatter)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.WithContext(ctx).Info("Hello World!")
	}
}

func BenchmarkWithTransaction(b *testing.B) {
	app := integrationsupport.NewTestApp(nil, nil)
	txn := app.StartTransaction("TestLogDistributedTracingDisabled")
	log := newTestLogger(bytes.NewBuffer([]byte("")))
	ctx := newrelic.NewContext(context.Background(), txn)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		log.WithContext(ctx).Info("Hello World!")
	}
}

func TestLogNoContext(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	log.WithTime(testTime).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"file.name":   matchAnything,
		"line.number": matchAnything,
		"log.level":   "info",
		"message":     "Hello World!",
		"method.name": matchAnything,
		"timestamp":   float64(1417136460000),
	})
}

func TestLogNoTxn(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	log.WithTime(testTime).WithContext(context.Background()).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"file.name":   matchAnything,
		"line.number": matchAnything,
		"log.level":   "info",
		"message":     "Hello World!",
		"method.name": matchAnything,
		"timestamp":   float64(1417136460000),
	})
}

func TestLogDistributedTracingDisabled(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, nil)
	txn := app.StartTransaction("TestLogDistributedTracingDisabled")
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	ctx := newrelic.NewContext(context.Background(), txn)
	log.WithTime(testTime).WithContext(ctx).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"entity.name": integrationsupport.SampleAppName,
		"entity.type": "SERVICE",
		"file.name":   matchAnything,
		"hostname":    host,
		"line.number": matchAnything,
		"log.level":   "info",
		"message":     "Hello World!",
		"method.name": matchAnything,
		"timestamp":   float64(1417136460000),
		"trace.id":    matchAnything,
	})
}

func TestLogSampledFalse(t *testing.T) {
	app := integrationsupport.NewTestApp(
		func(reply *internal.ConnectReply) {
			reply.SetSampleNothing()
			reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
		},
		func(cfg *newrelic.Config) {
			cfg.DistributedTracer.Enabled = true
			cfg.CrossApplicationTracer.Enabled = false
		})
	txn := app.StartTransaction("TestLogSampledFalse")
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	ctx := newrelic.NewContext(context.Background(), txn)
	log.WithTime(testTime).WithContext(ctx).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"entity.name": integrationsupport.SampleAppName,
		"entity.type": "SERVICE",
		"file.name":   matchAnything,
		"hostname":    host,
		"line.number": matchAnything,
		"log.level":   "info",
		"message":     "Hello World!",
		"method.name": matchAnything,
		"timestamp":   float64(1417136460000),
		"trace.id":    "1ae969564b34a33ecd1af05fe6923d6d",
	})
}

func TestLogSampledTrue(t *testing.T) {
	app := integrationsupport.NewTestApp(
		func(reply *internal.ConnectReply) {
			reply.SetSampleEverything()
			reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
		},
		func(cfg *newrelic.Config) {
			cfg.DistributedTracer.Enabled = true
			cfg.CrossApplicationTracer.Enabled = false
		})
	txn := app.StartTransaction("TestLogSampledTrue")
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	ctx := newrelic.NewContext(context.Background(), txn)
	log.WithTime(testTime).WithContext(ctx).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"entity.name": integrationsupport.SampleAppName,
		"entity.type": "SERVICE",
		"file.name":   matchAnything,
		"hostname":    host,
		"line.number": matchAnything,
		"log.level":   "info",
		"message":     "Hello World!",
		"method.name": matchAnything,
		"span.id":     "e71870997d57214c",
		"timestamp":   float64(1417136460000),
		"trace.id":    "1ae969564b34a33ecd1af05fe6923d6d",
	})
}

func TestEntryUsedTwice(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	entry := log.WithTime(testTime)

	// First log has dt enabled, ensure trace.id and span.id are included
	app := integrationsupport.NewTestApp(
		func(reply *internal.ConnectReply) {
			reply.SetSampleEverything()
			reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
		},
		func(cfg *newrelic.Config) {
			cfg.DistributedTracer.Enabled = true
			cfg.CrossApplicationTracer.Enabled = false
		})
	txn := app.StartTransaction("TestEntryUsedTwice1")
	ctx := newrelic.NewContext(context.Background(), txn)
	entry.WithContext(ctx).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"entity.name": integrationsupport.SampleAppName,
		"entity.type": "SERVICE",
		"file.name":   matchAnything,
		"hostname":    host,
		"line.number": matchAnything,
		"log.level":   "info",
		"message":     "Hello World!",
		"method.name": matchAnything,
		"span.id":     "e71870997d57214c",
		"timestamp":   float64(1417136460000),
		"trace.id":    "1ae969564b34a33ecd1af05fe6923d6d",
	})

	// First log has dt enabled, ensure trace.id and span.id are included
	out.Reset()
	app = integrationsupport.NewTestApp(nil,
		func(cfg *newrelic.Config) {
			cfg.DistributedTracer.Enabled = false
		})
	txn = app.StartTransaction("TestEntryUsedTwice2")
	ctx = newrelic.NewContext(context.Background(), txn)
	entry.WithContext(ctx).Info("Hello World! Again!")
	validateOutput(t, out, map[string]interface{}{
		"entity.name": integrationsupport.SampleAppName,
		"entity.type": "SERVICE",
		"file.name":   matchAnything,
		"hostname":    host,
		"line.number": matchAnything,
		"log.level":   "info",
		"message":     "Hello World! Again!",
		"method.name": matchAnything,
		"timestamp":   float64(1417136460000),
	})
}

func TestEntryError(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, nil)
	txn := app.StartTransaction("TestEntryError")
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	ctx := newrelic.NewContext(context.Background(), txn)
	log.WithTime(testTime).WithContext(ctx).WithField("func", func() {}).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"entity.name": integrationsupport.SampleAppName,
		"entity.type": "SERVICE",
		"file.name":   matchAnything,
		"hostname":    host,
		"line.number": matchAnything,
		"log.level":   "info",
		// Since the err field on the Entry is private we cannot record it.
		//"logrus_error": `can not add field "func"`,
		"message":     "Hello World!",
		"method.name": matchAnything,
		"timestamp":   float64(1417136460000),
		"trace.id":    matchAnything,
	})
}

func TestWithCustomField(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, nil)
	txn := app.StartTransaction("TestWithCustomField")
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	ctx := newrelic.NewContext(context.Background(), txn)
	log.WithTime(testTime).WithContext(ctx).WithField("zip", "zap").Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"entity.name": integrationsupport.SampleAppName,
		"entity.type": "SERVICE",
		"file.name":   matchAnything,
		"hostname":    host,
		"line.number": matchAnything,
		"log.level":   "info",
		"message":     "Hello World!",
		"method.name": matchAnything,
		"timestamp":   float64(1417136460000),
		"zip":         "zap",
		"trace.id":    matchAnything,
	})
}

func TestCustomFieldTypes(t *testing.T) {
	out := bytes.NewBuffer([]byte{})

	testcases := []struct {
		input  interface{}
		output string
	}{
		{input: true, output: "true"},
		{input: false, output: "false"},
		{input: uint8(42), output: "42"},
		{input: uint16(42), output: "42"},
		{input: uint32(42), output: "42"},
		{input: uint(42), output: "42"},
		{input: uintptr(42), output: "42"},
		{input: int8(42), output: "42"},
		{input: int16(42), output: "42"},
		{input: int32(42), output: "42"},
		{input: int64(42), output: "42"},
		{input: float32(42), output: "42"},
		{input: float64(42), output: "42"},
		{input: errors.New("Ooops an error"), output: `"Ooops an error"`},
		{input: []int{1, 2, 3}, output: `"[]int{1, 2, 3}"`},
	}

	for _, test := range testcases {
		out.Reset()
		writeValue(out, test.input)
		if out.String() != test.output {
			t.Errorf("Incorrect output written:\nactual=%s\nexpected=%s",
				out.String(), test.output)
		}
	}
}

func TestUnsetCaller(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	log.SetReportCaller(false)
	log.WithTime(testTime).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"log.level": "info",
		"message":   "Hello World!",
		"timestamp": float64(1417136460000),
	})
}

func TestCustomFieldNameCollision(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	log.SetReportCaller(false)
	log.WithTime(testTime).WithField("timestamp", "Yesterday").Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"log.level": "info",
		"message":   "Hello World!",
		// Reserved keys will be overwritten
		"timestamp": float64(1417136460000),
	})
}

type gopher struct {
	name string
}

func (g *gopher) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.name)
}

func TestCustomJSONMarshaller(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	log.SetReportCaller(false)
	log.WithTime(testTime).WithField("gopher", &gopher{name: "sam"}).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"gopher":    "sam",
		"log.level": "info",
		"message":   "Hello World!",
		"timestamp": float64(1417136460000),
	})
}
