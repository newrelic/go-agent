// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrlambda

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func testApp(getenv func(string) string, t *testing.T) *newrelic.Application {
	if nil == getenv {
		getenv = func(string) string { return "" }
	}
	cfg := newConfigInternal(getenv)

	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		t.Fatal(err)
	}
	internal.HarvestTesting(app.Private, nil)
	return app
}

func distributedTracingEnabled(key string) string {
	switch key {
	case "NEW_RELIC_ACCOUNT_ID":
		return "1"
	case "NEW_RELIC_TRUSTED_ACCOUNT_KEY":
		return "1"
	case "NEW_RELIC_PRIMARY_APPLICATION_ID":
		return "1"
	default:
		return ""
	}
}

// bufWriterProvider is a testing implementation of writerProvider
type bufWriterProvider struct {
	buf io.Writer
}

func (bw bufWriterProvider) borrowWriter(needsWriter func(writer io.Writer)) {
	needsWriter(bw.buf)
}

func TestColdStart(t *testing.T) {
	originalHandler := func(c context.Context) {}
	app := testApp(nil, t)
	wrapped := Wrap(originalHandler, app)
	w := wrapped.(*wrappedHandler)
	w.functionName = "functionName"
	buf := &bytes.Buffer{}
	w.hasWriter = bufWriterProvider{buf}

	ctx := context.Background()
	lctx := &lambdacontext.LambdaContext{
		AwsRequestID:       "request-id",
		InvokedFunctionArn: "function-arn",
	}
	ctx = lambdacontext.NewContext(ctx, lctx)

	resp, err := wrapped.Invoke(ctx, nil)
	if nil != err || string(resp) != "null" {
		t.Error("unexpected response", err, string(resp))
	}
	app.Private.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":     "OtherTransaction/Go/functionName",
			"guid":     internal.MatchAnything,
			"priority": internal.MatchAnything,
			"sampled":  internal.MatchAnything,
			"traceId":  internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.requestId":        "request-id",
			"aws.lambda.arn":       "function-arn",
			"aws.lambda.coldStart": true,
		},
	}})
	app.Private.(internal.Expect).ExpectSpanEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "OtherTransaction/Go/functionName",
			"transaction.name": "OtherTransaction/Go/functionName",
			"guid":             internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"traceId":          internal.MatchAnything,
			"category":         "generic",
			"nr.entryPoint":    true,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.requestId":        "request-id",
			"aws.lambda.arn":       "function-arn",
			"aws.lambda.coldStart": true,
		},
	}})
	if 0 == buf.Len() {
		t.Error("no output written")
	}

	// Invoke the handler again to test the cold-start attribute absence.
	buf = &bytes.Buffer{}
	w.hasWriter = bufWriterProvider{buf}
	internal.HarvestTesting(app.Private, nil)
	resp, err = wrapped.Invoke(ctx, nil)
	if nil != err || string(resp) != "null" {
		t.Error("unexpected response", err, string(resp))
	}
	app.Private.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":     "OtherTransaction/Go/functionName",
			"guid":     internal.MatchAnything,
			"priority": internal.MatchAnything,
			"sampled":  internal.MatchAnything,
			"traceId":  internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.requestId":  "request-id",
			"aws.lambda.arn": "function-arn",
		},
	}})
	app.Private.(internal.Expect).ExpectSpanEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "OtherTransaction/Go/functionName",
			"transaction.name": "OtherTransaction/Go/functionName",
			"guid":             internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"traceId":          internal.MatchAnything,
			"category":         "generic",
			"nr.entryPoint":    true,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.requestId":  "request-id",
			"aws.lambda.arn": "function-arn",
		},
	}})
	if 0 == buf.Len() {
		t.Error("no output written")
	}
}

func TestErrorCapture(t *testing.T) {
	returnError := errors.New("problem")
	originalHandler := func() error { return returnError }
	app := testApp(nil, t)
	wrapped := Wrap(originalHandler, app)
	w := wrapped.(*wrappedHandler)
	w.functionName = "functionName"
	buf := &bytes.Buffer{}
	w.hasWriter = bufWriterProvider{buf}

	resp, err := wrapped.Invoke(context.Background(), nil)
	if err != returnError || string(resp) != "" {
		t.Error(err, string(resp))
	}
	app.Private.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/functionName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/functionName", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		// Error metrics test the error capture.
		{Name: "Errors/all", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/OtherTransaction/Go/functionName", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
	})
	app.Private.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":     "OtherTransaction/Go/functionName",
			"guid":     internal.MatchAnything,
			"priority": internal.MatchAnything,
			"sampled":  internal.MatchAnything,
			"traceId":  internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.lambda.coldStart": true,
		},
	}})
	app.Private.(internal.Expect).ExpectSpanEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "OtherTransaction/Go/functionName",
			"transaction.name": "OtherTransaction/Go/functionName",
			"guid":             internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"traceId":          internal.MatchAnything,
			"category":         "generic",
			"nr.entryPoint":    true,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.lambda.coldStart": true,
			"error.class":          "*errors.errorString",
			"error.message":        "problem",
		},
	}})
	if 0 == buf.Len() {
		t.Error("no output written")
	}
}

func TestWrapNilApp(t *testing.T) {
	originalHandler := func() (int, error) {
		return 123, nil
	}
	wrapped := Wrap(originalHandler, nil)
	ctx := context.Background()
	resp, err := wrapped.Invoke(ctx, nil)
	if nil != err || string(resp) != "123" {
		t.Error("unexpected response", err, string(resp))
	}
}

func TestSetWebRequest(t *testing.T) {
	originalHandler := func(events.APIGatewayProxyRequest) {}
	app := testApp(nil, t)
	wrapped := Wrap(originalHandler, app)
	w := wrapped.(*wrappedHandler)
	w.functionName = "functionName"
	buf := &bytes.Buffer{}
	w.hasWriter = bufWriterProvider{buf}

	req := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"X-Forwarded-Port":  "4000",
			"X-Forwarded-Proto": "HTTPS",
		},
	}
	reqbytes, err := json.Marshal(req)
	if err != nil {
		t.Error("unable to marshal json", err)
	}

	resp, err := wrapped.Invoke(context.Background(), reqbytes)
	if err != nil {
		t.Error(err, string(resp))
	}
	app.Private.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/functionName", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/functionName", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/functionName", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: nil},
	})
	app.Private.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/functionName",
			"nr.apdexPerfZone": "S",
			"guid":             internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"traceId":          internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.lambda.coldStart": true,
			"request.uri":          "//:4000",
		},
	}})
	app.Private.(internal.Expect).ExpectSpanEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/functionName",
			"transaction.name": "WebTransaction/Go/functionName",
			"guid":             internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"traceId":          internal.MatchAnything,
			"category":         "generic",
			"nr.entryPoint":    true,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.lambda.coldStart": true,
			"request.uri":          "//:4000",
		},
	}})
	if 0 == buf.Len() {
		t.Error("no output written")
	}
}

func TestDistributedTracing(t *testing.T) {
	originalHandler := func(events.APIGatewayProxyRequest) {}
	app := testApp(distributedTracingEnabled, t)
	wrapped := Wrap(originalHandler, app)
	w := wrapped.(*wrappedHandler)
	w.functionName = "functionName"
	buf := &bytes.Buffer{}
	w.hasWriter = bufWriterProvider{buf}

	dtHdr := http.Header{}
	app.StartTransaction("hello").InsertDistributedTraceHeaders(dtHdr)
	hdr := map[string]string{
		"X-Forwarded-Port":  "4000",
		"X-Forwarded-Proto": "HTTPS",
	}
	for k := range dtHdr {
		if v := dtHdr.Get(k); v != "" {
			hdr[k] = v
		}
	}
	req := events.APIGatewayProxyRequest{Headers: hdr}
	reqbytes, err := json.Marshal(req)
	if err != nil {
		t.Error("unable to marshal json", err)
	}

	resp, err := wrapped.Invoke(context.Background(), reqbytes)
	if err != nil {
		t.Error(err, string(resp))
	}
	app.Private.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/functionName", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/1/1/HTTPS/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/1/1/HTTPS/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: nil},
		{Name: "TransportDuration/App/1/1/HTTPS/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/1/1/HTTPS/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/functionName", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/functionName", Scope: "", Forced: false, Data: nil},
	})
	app.Private.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                     "WebTransaction/Go/functionName",
			"nr.apdexPerfZone":         "S",
			"parent.account":           "1",
			"parent.app":               "1",
			"parent.transportType":     "HTTPS",
			"parent.type":              "App",
			"guid":                     internal.MatchAnything,
			"parent.transportDuration": internal.MatchAnything,
			"parentId":                 internal.MatchAnything,
			"parentSpanId":             internal.MatchAnything,
			"priority":                 internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"traceId":                  internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.lambda.coldStart": true,
			"request.uri":          "//:4000",
		},
	}})
	app.Private.(internal.Expect).ExpectSpanEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/functionName",
			"transaction.name": "WebTransaction/Go/functionName",
			"guid":             internal.MatchAnything,
			"parentId":         internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"traceId":          internal.MatchAnything,
			"trustedParentId":  internal.MatchAnything,
			"category":         "generic",
			"nr.entryPoint":    true,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.lambda.coldStart":     true,
			"parent.account":           "1",
			"parent.app":               "1",
			"parent.transportDuration": internal.MatchAnything,
			"parent.transportType":     "HTTPS",
			"parent.type":              "App",
			"request.uri":              "//:4000",
		},
	}})
	if 0 == buf.Len() {
		t.Error("no output written")
	}
}

func TestEventARN(t *testing.T) {
	originalHandler := func(events.DynamoDBEvent) {}
	app := testApp(nil, t)
	wrapped := Wrap(originalHandler, app)
	w := wrapped.(*wrappedHandler)
	w.functionName = "functionName"
	buf := &bytes.Buffer{}
	w.hasWriter = bufWriterProvider{buf}

	req := events.DynamoDBEvent{
		Records: []events.DynamoDBEventRecord{{
			EventSourceArn: "ARN",
		}},
	}

	reqbytes, err := json.Marshal(req)
	if err != nil {
		t.Error("unable to marshal json", err)
	}

	resp, err := wrapped.Invoke(context.Background(), reqbytes)
	if err != nil {
		t.Error(err, string(resp))
	}
	app.Private.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/Go/functionName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/functionName", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
	})
	app.Private.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":     "OtherTransaction/Go/functionName",
			"guid":     internal.MatchAnything,
			"priority": internal.MatchAnything,
			"sampled":  internal.MatchAnything,
			"traceId":  internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.lambda.coldStart":       true,
			"aws.lambda.eventSource.arn": "ARN",
		},
	}})
	app.Private.(internal.Expect).ExpectSpanEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "OtherTransaction/Go/functionName",
			"transaction.name": "OtherTransaction/Go/functionName",
			"guid":             internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"traceId":          internal.MatchAnything,
			"nr.entryPoint":    true,
			"category":         "generic",
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.lambda.coldStart":       true,
			"aws.lambda.eventSource.arn": "ARN",
		},
	}})
	if 0 == buf.Len() {
		t.Error("no output written")
	}
}

func TestAPIGatewayProxyResponse(t *testing.T) {
	originalHandler := func() (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{
			Body:       "Hello World",
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "text/html",
			},
		}, nil
	}

	app := testApp(nil, t)
	wrapped := Wrap(originalHandler, app)
	w := wrapped.(*wrappedHandler)
	w.functionName = "functionName"
	buf := &bytes.Buffer{}
	w.hasWriter = bufWriterProvider{buf}

	resp, err := wrapped.Invoke(context.Background(), nil)
	if nil != err {
		t.Error("unexpected err", err)
	}
	if !strings.Contains(string(resp), "Hello World") {
		t.Error("unexpected response", string(resp))
	}

	app.Private.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":     "OtherTransaction/Go/functionName",
			"guid":     internal.MatchAnything,
			"priority": internal.MatchAnything,
			"sampled":  internal.MatchAnything,
			"traceId":  internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.lambda.coldStart":         true,
			"httpResponseCode":             "200",
			"http.statusCode":              "200",
			"response.headers.contentType": "text/html",
		},
	}})
	app.Private.(internal.Expect).ExpectSpanEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "OtherTransaction/Go/functionName",
			"transaction.name": "OtherTransaction/Go/functionName",
			"guid":             internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"traceId":          internal.MatchAnything,
			"category":         "generic",
			"nr.entryPoint":    true,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.lambda.coldStart":         true,
			"httpResponseCode":             "200",
			"http.statusCode":              200,
			"response.headers.contentType": "text/html",
		},
	}})
	if 0 == buf.Len() {
		t.Error("no output written")
	}
}

func TestCustomEvent(t *testing.T) {
	originalHandler := func(c context.Context) {
		txn := newrelic.FromContext(c)
		txn.Application().RecordCustomEvent("myEvent", map[string]interface{}{
			"zip": "zap",
		})
	}
	app := testApp(nil, t)
	wrapped := Wrap(originalHandler, app)
	w := wrapped.(*wrappedHandler)
	w.functionName = "functionName"
	buf := &bytes.Buffer{}
	w.hasWriter = bufWriterProvider{buf}

	resp, err := wrapped.Invoke(context.Background(), nil)
	if nil != err || string(resp) != "null" {
		t.Error("unexpected response", err, string(resp))
	}
	app.Private.(internal.Expect).ExpectCustomEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"type":      "myEvent",
			"timestamp": internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{
			"zip": "zap",
		},
		AgentAttributes: map[string]interface{}{},
	}})
	if 0 == buf.Len() {
		t.Error("no output written")
	}
}

func TestDefaultWriterProvider(t *testing.T) {
	dwp := defaultWriterProvider{}
	dwp.borrowWriter(func(writer io.Writer) {
		if writer != os.Stdout {
			t.Error("Expected stdout")
		}
	})

	const telemetryFile = "/tmp/newrelic-telemetry"
	defer os.Remove(telemetryFile)
	file, err := os.Create(telemetryFile)
	if err != nil {
		t.Error("Unexpected error creating telemetry file", err)
	}

	err = file.Close()
	if err != nil {
		t.Error("Error closing telemetry file", err)
	}

	dwp.borrowWriter(func(writer io.Writer) {
		if writer == os.Stdout {
			t.Error("Expected telemetry file, got stdout")
		}
	})
}
