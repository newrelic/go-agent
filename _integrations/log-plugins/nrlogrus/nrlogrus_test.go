package nrlogrus

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/sysinfo"
)

var (
	testTime      = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	matchAnything = struct{}{}
)

func newTestLogger(out io.Writer) *logrus.Logger {
	l := logrus.New()
	l.Formatter = NewFormatter()
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

func testApp(t *testing.T, cfgFn func(*newrelic.Config), replyFn func(*internal.ConnectReply)) newrelic.Application {
	cfg := newrelic.NewConfig("AppName", "0123456789012345678901234567890123456789")
	cfg.Enabled = false
	if nil != cfgFn {
		cfgFn(&cfg)
	}

	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		t.Fatal(err)
	}

	internal.HarvestTesting(app, replyFn)
	return app
}

func TestLogNoContext(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	log.WithTime(testTime).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"file.name":   matchAnything,
		"log.level":   "info",
		"message":     "Hello World!",
		"method.name": "github.com/newrelic/go-agent/_integrations/log-plugins/nrlogrus.TestLogNoContext",
		"timestamp":   float64(1417136460000),
	})
}

func TestLogNoTxn(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	log.WithTime(testTime).WithContext(context.Background()).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"file.name":   matchAnything,
		"log.level":   "info",
		"message":     "Hello World!",
		"method.name": "github.com/newrelic/go-agent/_integrations/log-plugins/nrlogrus.TestLogNoTxn",
		"timestamp":   float64(1417136460000),
	})
}

func TestLogDistributedTracingDisabled(t *testing.T) {
	app := testApp(t, nil, nil)
	txn := app.StartTransaction("TestLogDistributedTracingDisabled", nil, nil)
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	ctx := newrelic.NewContext(context.Background(), txn)
	host, _ := sysinfo.Hostname()
	log.WithTime(testTime).WithContext(ctx).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"entity.name": "AppName",
		"entity.type": "SERVICE",
		"file.name":   matchAnything,
		"hostname":    host,
		"log.level":   "info",
		"message":     "Hello World!",
		"method.name": "github.com/newrelic/go-agent/_integrations/log-plugins/nrlogrus.TestLogDistributedTracingDisabled",
		"timestamp":   float64(1417136460000),
	})
}

func TestLogSampledFalse(t *testing.T) {
	app := testApp(t,
		func(cfg *newrelic.Config) {
			cfg.DistributedTracer.Enabled = true
			cfg.CrossApplicationTracer.Enabled = false
		},
		func(reply *internal.ConnectReply) {
			reply.AdaptiveSampler = internal.SampleNothing{}
			reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
		})
	txn := app.StartTransaction("TestLogSampledFalse", nil, nil)
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	ctx := newrelic.NewContext(context.Background(), txn)
	host, _ := sysinfo.Hostname()
	log.WithTime(testTime).WithContext(ctx).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"entity.name": "AppName",
		"entity.type": "SERVICE",
		"file.name":   matchAnything,
		"hostname":    host,
		"log.level":   "info",
		"message":     "Hello World!",
		"method.name": "github.com/newrelic/go-agent/_integrations/log-plugins/nrlogrus.TestLogSampledFalse",
		"timestamp":   float64(1417136460000),
		"trace.id":    "d9466896a525ccbf",
	})
}

func TestLogSampledTrue(t *testing.T) {
	app := testApp(t,
		func(cfg *newrelic.Config) {
			cfg.DistributedTracer.Enabled = true
			cfg.CrossApplicationTracer.Enabled = false
		},
		func(reply *internal.ConnectReply) {
			reply.AdaptiveSampler = internal.SampleEverything{}
			reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
		})
	txn := app.StartTransaction("TestLogSampledTrue", nil, nil)
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	ctx := newrelic.NewContext(context.Background(), txn)
	host, _ := sysinfo.Hostname()
	log.WithTime(testTime).WithContext(ctx).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"entity.name": "AppName",
		"entity.type": "SERVICE",
		"file.name":   matchAnything,
		"hostname":    host,
		"log.level":   "info",
		"message":     "Hello World!",
		"method.name": "github.com/newrelic/go-agent/_integrations/log-plugins/nrlogrus.TestLogSampledTrue",
		"span.id":     "bcfb32e050b264b8",
		"timestamp":   float64(1417136460000),
		"trace.id":    "d9466896a525ccbf",
	})
}

func TestEntryUsedTwice(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	entry := log.WithTime(testTime)

	// First log has dt enabled, ensure trace.id and span.id are included
	app := testApp(t,
		func(cfg *newrelic.Config) {
			cfg.DistributedTracer.Enabled = true
			cfg.CrossApplicationTracer.Enabled = false
		},
		func(reply *internal.ConnectReply) {
			reply.AdaptiveSampler = internal.SampleEverything{}
			reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
		})
	txn := app.StartTransaction("TestEntryUsedTwice1", nil, nil)
	ctx := newrelic.NewContext(context.Background(), txn)
	host, _ := sysinfo.Hostname()
	entry.WithContext(ctx).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"entity.name": "AppName",
		"entity.type": "SERVICE",
		"file.name":   matchAnything,
		"hostname":    host,
		"log.level":   "info",
		"message":     "Hello World!",
		"method.name": "github.com/newrelic/go-agent/_integrations/log-plugins/nrlogrus.TestEntryUsedTwice",
		"span.id":     "bcfb32e050b264b8",
		"timestamp":   float64(1417136460000),
		"trace.id":    "d9466896a525ccbf",
	})

	// First log has dt enabled, ensure trace.id and span.id are included
	out.Reset()
	app = testApp(t,
		func(cfg *newrelic.Config) {
			cfg.DistributedTracer.Enabled = false
		}, nil)
	txn = app.StartTransaction("TestEntryUsedTwice2", nil, nil)
	ctx = newrelic.NewContext(context.Background(), txn)
	host, _ = sysinfo.Hostname()
	entry.WithContext(ctx).Info("Hello World! Again!")
	validateOutput(t, out, map[string]interface{}{
		"entity.name": "AppName",
		"entity.type": "SERVICE",
		"file.name":   matchAnything,
		"hostname":    host,
		"log.level":   "info",
		"message":     "Hello World! Again!",
		"method.name": "github.com/newrelic/go-agent/_integrations/log-plugins/nrlogrus.TestEntryUsedTwice",
		"timestamp":   float64(1417136460000),
	})
}

func TestEntryError(t *testing.T) {
	app := testApp(t, nil, nil)
	txn := app.StartTransaction("TestEntryError", nil, nil)
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	ctx := newrelic.NewContext(context.Background(), txn)
	host, _ := sysinfo.Hostname()
	log.WithTime(testTime).WithContext(ctx).WithField("func", func() {}).Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"entity.name":  "AppName",
		"entity.type":  "SERVICE",
		"file.name":    matchAnything,
		"hostname":     host,
		"log.level":    "info",
		"logrus_error": `can not add field "func"`,
		"message":      "Hello World!",
		"method.name":  "github.com/newrelic/go-agent/_integrations/log-plugins/nrlogrus.TestEntryError",
		"timestamp":    float64(1417136460000),
	})
}

func TestWithCustomField(t *testing.T) {
	app := testApp(t, nil, nil)
	txn := app.StartTransaction("TestWithCustomField", nil, nil)
	out := bytes.NewBuffer([]byte{})
	log := newTestLogger(out)
	ctx := newrelic.NewContext(context.Background(), txn)
	host, _ := sysinfo.Hostname()
	log.WithTime(testTime).WithContext(ctx).WithField("zip", "zap").Info("Hello World!")
	validateOutput(t, out, map[string]interface{}{
		"entity.name": "AppName",
		"entity.type": "SERVICE",
		"file.name":   matchAnything,
		"hostname":    host,
		"log.level":   "info",
		"message":     "Hello World!",
		"method.name": "github.com/newrelic/go-agent/_integrations/log-plugins/nrlogrus.TestWithCustomField",
		"timestamp":   float64(1417136460000),
		"zip":         "zap",
	})
}
