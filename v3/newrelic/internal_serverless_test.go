// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
)

func TestServerlessDistributedTracingConfigPresent(t *testing.T) {
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.DistributedTracer.Enabled = true
		cfg.ServerlessMode.AccountID = "123"
		cfg.ServerlessMode.TrustedAccountKey = "987"
		cfg.ServerlessMode.PrimaryAppID = "456"
	}
	app := testApp(nil, cfgFn, t)
	hdrs := http.Header{}
	app.StartTransaction("hello").InsertDistributedTraceHeaders(hdrs)
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: singleCount},
	})
}

func TestServerlessDistributedTracingConfigPartiallyPresent(t *testing.T) {
	// This tests that if ServerlessMode.PrimaryAppID is unset it should
	// default to "Unknown".
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.DistributedTracer.Enabled = true
		cfg.ServerlessMode.AccountID = "123"
		cfg.ServerlessMode.TrustedAccountKey = "987"
	}
	app := testApp(nil, cfgFn, t)
	hdrs := http.Header{}
	app.StartTransaction("hello").InsertDistributedTraceHeaders(hdrs)
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/Unknown/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/Unknown/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: singleCount},
	})
}

func TestServerlessDistributedTracingConfigTrustKeyAbsent(t *testing.T) {
	// Test that distributed tracing works if only AccountID has been set.
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.DistributedTracer.Enabled = true
		cfg.ServerlessMode.AccountID = "123"
	}
	app := testApp(nil, cfgFn, t)
	hdrs := http.Header{}
	app.StartTransaction("hello").InsertDistributedTraceHeaders(hdrs)
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/Unknown/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/Unknown/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: singleCount},
	})
}

func TestServerlessDistributedTracingConfigAbsent(t *testing.T) {
	// Test that payloads do not get created or accepted when distributed
	// tracing configuration is not present.
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello")
	emptyHdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(emptyHdrs)
	if len(emptyHdrs) != 0 {
		t.Error(emptyHdrs)
	}
	nonEmptyHdrs := http.Header{}
	app2 := testApp(nil, func(cfg *Config) {
		cfgFn(cfg)
		cfg.ServerlessMode.AccountID = "123"
		cfg.ServerlessMode.TrustedAccountKey = "trustkey"
		cfg.ServerlessMode.PrimaryAppID = "456"
	}, t)
	app2.StartTransaction("hello").InsertDistributedTraceHeaders(nonEmptyHdrs)
	if len(nonEmptyHdrs) == 0 {
		t.Error(nonEmptyHdrs)
	}
	txn.AcceptDistributedTraceHeaders(TransportHTTP, nonEmptyHdrs)
	app.expectNoLoggedErrors(t)
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
	})
}

func TestServerlessLowApdex(t *testing.T) {
	apdex := -1 * time.Second
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.ServerlessMode.ApdexThreshold = apdex
		cfg.DistributedTracer.Enabled = false
	}
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello")
	txn.SetWebRequestHTTP(nil) // only web gets apdex
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		// third apdex field is failed count
		{Name: "Apdex", Scope: "", Forced: true, Data: []float64{0, 0, 1, apdex.Seconds(), apdex.Seconds(), 0}},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: []float64{0, 0, 1, apdex.Seconds(), apdex.Seconds(), 0}},
	})
}

func TestServerlessHighApdex(t *testing.T) {
	apdex := 1 * time.Hour
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.ServerlessMode.ApdexThreshold = apdex
		cfg.DistributedTracer.Enabled = false
	}
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello")
	txn.SetWebRequestHTTP(nil) // only web gets apdex
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		// first apdex field is satisfied count
		{Name: "Apdex", Scope: "", Forced: true, Data: []float64{1, 0, 0, apdex.Seconds(), apdex.Seconds(), 0}},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: []float64{1, 0, 0, apdex.Seconds(), apdex.Seconds(), 0}},
	})
}

func TestServerlessRecordCustomMetric(t *testing.T) {
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.DistributedTracer.Enabled = false
	}
	app := testApp(nil, cfgFn, t)
	app.RecordCustomMetric("myMetric", 123.0)
	app.expectSingleLoggedError(t, "unable to record custom metric", map[string]interface{}{
		"metric-name": "myMetric",
		"reason":      errMetricServerless.Error(),
	})
}

func TestServerlessRecordCustomEvent(t *testing.T) {
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.DistributedTracer.Enabled = false
	}
	app := testApp(nil, cfgFn, t)

	attributes := map[string]interface{}{"zip": 1}
	app.RecordCustomEvent("myType", attributes)
	app.expectNoLoggedErrors(t)
	app.ExpectCustomEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"type":      "myType",
			"timestamp": internal.MatchAnything,
		},
		UserAttributes: attributes,
	}})

	buf := &bytes.Buffer{}
	internal.ServerlessWrite(app.Application.Private, "my-arn", buf)

	_, data, err := parseServerlessPayload(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	// Data should contain only custom events.  Dynamic timestamp makes exact
	// comparison difficult.
	eventData := string(data["custom_event_data"])
	if !strings.Contains(eventData, `{"zip":1}`) {
		t.Error(eventData)
	}
	if len(data) != 1 {
		t.Fatal(data)
	}
}

func TestServerlessJSON(t *testing.T) {
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.DistributedTracer.Enabled = false
	}
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello")
	txn.Private.(internal.AddAgentAttributer).AddAgentAttribute(AttributeAWSLambdaARN, "thearn", nil)
	txn.End()

	buf := &bytes.Buffer{}
	internal.ServerlessWrite(app.Application.Private, "lambda-test-arn", buf)

	metadata, data, err := parseServerlessPayload(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	// Data should contain txn event and metrics.  Timestamps make exact
	// JSON comparison tough.
	if v := data["metric_data"]; nil == v {
		t.Fatal(data)
	}
	if v := data["analytic_event_data"]; nil == v {
		t.Fatal(data)
	}
	if v := string(metadata["arn"]); v != `"lambda-test-arn"` {
		t.Fatal(v)
	}
	if v := string(metadata["agent_version"]); v != `"`+Version+`"` {
		t.Fatal(v)
	}
}

func TestServerlessConnectReply(t *testing.T) {
	cfg := config{Config: defaultConfig()}
	cfg.ServerlessMode.ApdexThreshold = 2 * time.Second
	cfg.ServerlessMode.AccountID = "the-account-id"
	cfg.ServerlessMode.TrustedAccountKey = "the-trust-key"
	cfg.ServerlessMode.PrimaryAppID = "the-primary-app"
	reply := newServerlessConnectReply(cfg)
	if reply.ApdexThresholdSeconds != 2 {
		t.Error(reply.ApdexThresholdSeconds)
	}
	if reply.AccountID != "the-account-id" {
		t.Error(reply.AccountID)
	}
	if reply.TrustedAccountKey != "the-trust-key" {
		t.Error(reply.TrustedAccountKey)
	}
	if reply.PrimaryAppID != "the-primary-app" {
		t.Error(reply.PrimaryAppID)
	}
	if reply.SamplingTargetPeriodInSeconds != 60 {
		t.Error(reply.SamplingTargetPeriodInSeconds)
	}
	if reply.SamplingTarget != 10 {
		t.Error(reply.SamplingTarget)
	}

	// Now test the defaults:
	cfg = config{Config: defaultConfig()}
	reply = newServerlessConnectReply(cfg)
	if reply.ApdexThresholdSeconds != 0.5 {
		t.Error(reply.ApdexThresholdSeconds)
	}
	if reply.AccountID != "" {
		t.Error(reply.AccountID)
	}
	if reply.TrustedAccountKey != "" {
		t.Error(reply.TrustedAccountKey)
	}
	if reply.PrimaryAppID != "Unknown" {
		t.Error(reply.PrimaryAppID)
	}
}
