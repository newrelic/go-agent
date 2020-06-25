// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrlambda

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

func dataShouldContain(tb testing.TB, data map[string]json.RawMessage, keys ...string) {
	if h, ok := tb.(interface {
		Helper()
	}); ok {
		h.Helper()
	}
	if len(data) != len(keys) {
		tb.Errorf("data key length mismatch, expected=%v got=%v",
			len(keys), len(data))
		return
	}
	for _, k := range keys {
		_, ok := data[k]
		if !ok {
			tb.Errorf("data does not contain key %v", k)
		}
	}
}

func testApp(getenv func(string) string, t *testing.T) newrelic.Application {
	if nil == getenv {
		getenv = func(string) string { return "" }
	}
	cfg := newConfigInternal(getenv)

	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		t.Fatal(err)
	}
	internal.HarvestTesting(app, nil)
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

func TestColdStart(t *testing.T) {
	originalHandler := func(c context.Context) {}
	app := testApp(nil, t)
	wrapped := Wrap(originalHandler, app)
	w := wrapped.(*wrappedHandler)
	w.functionName = "functionName"
	buf := &bytes.Buffer{}
	w.writer = buf

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
	app.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
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
	metadata, data, err := internal.ParseServerlessPayload(buf.Bytes())
	if err != nil {
		t.Error(err)
	}
	dataShouldContain(t, data, "metric_data", "analytic_event_data", "span_event_data")
	if v := string(metadata["arn"]); v != `"function-arn"` {
		t.Error(metadata)
	}

	// Invoke the handler again to test the cold-start attribute absence.
	buf = &bytes.Buffer{}
	w.writer = buf
	internal.HarvestTesting(app, nil)
	resp, err = wrapped.Invoke(ctx, nil)
	if nil != err || string(resp) != "null" {
		t.Error("unexpected response", err, string(resp))
	}
	app.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
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
	metadata, data, err = internal.ParseServerlessPayload(buf.Bytes())
	if err != nil {
		t.Error(err)
	}
	dataShouldContain(t, data, "metric_data", "analytic_event_data", "span_event_data")
	if v := string(metadata["arn"]); v != `"function-arn"` {
		t.Error(metadata)
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
	w.writer = buf

	resp, err := wrapped.Invoke(context.Background(), nil)
	if err != returnError || string(resp) != "" {
		t.Error(err, string(resp))
	}
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
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
	app.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
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
	_, data, err := internal.ParseServerlessPayload(buf.Bytes())
	if err != nil {
		t.Error(err)
	}
	dataShouldContain(t, data, "metric_data", "analytic_event_data", "span_event_data",
		"error_event_data", "error_data")
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
	w.writer = buf

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
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
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
	app.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
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
	_, data, err := internal.ParseServerlessPayload(buf.Bytes())
	if err != nil {
		t.Error(err)
	}
	dataShouldContain(t, data, "metric_data", "analytic_event_data", "span_event_data")
}

func makePayload(app newrelic.Application) string {
	txn := app.StartTransaction("hello", nil, nil)
	return txn.CreateDistributedTracePayload().Text()
}

func TestDistributedTracing(t *testing.T) {
	originalHandler := func(events.APIGatewayProxyRequest) {}
	app := testApp(distributedTracingEnabled, t)
	wrapped := Wrap(originalHandler, app)
	w := wrapped.(*wrappedHandler)
	w.functionName = "functionName"
	buf := &bytes.Buffer{}
	w.writer = buf

	req := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"X-Forwarded-Port":                     "4000",
			"X-Forwarded-Proto":                    "HTTPS",
			newrelic.DistributedTracePayloadHeader: makePayload(app),
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
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/functionName", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/1/1/HTTPS/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/1/1/HTTPS/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "TransportDuration/App/1/1/HTTPS/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/1/1/HTTPS/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/functionName", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/functionName", Scope: "", Forced: false, Data: nil},
	})
	app.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
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
	_, data, err := internal.ParseServerlessPayload(buf.Bytes())
	if err != nil {
		t.Error(err)
	}
	dataShouldContain(t, data, "metric_data", "analytic_event_data", "span_event_data")
}

func TestEventARN(t *testing.T) {
	originalHandler := func(events.DynamoDBEvent) {}
	app := testApp(nil, t)
	wrapped := Wrap(originalHandler, app)
	w := wrapped.(*wrappedHandler)
	w.functionName = "functionName"
	buf := &bytes.Buffer{}
	w.writer = buf

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
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/Go/functionName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/functionName", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
	})
	app.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
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
	_, data, err := internal.ParseServerlessPayload(buf.Bytes())
	if err != nil {
		t.Error(err)
	}
	dataShouldContain(t, data, "metric_data", "analytic_event_data", "span_event_data")
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
	w.writer = buf

	resp, err := wrapped.Invoke(context.Background(), nil)
	if nil != err {
		t.Error("unexpected err", err)
	}
	if !strings.Contains(string(resp), "Hello World") {
		t.Error("unexpected response", string(resp))
	}

	app.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
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
			"response.headers.contentType": "text/html",
		},
	}})
	_, data, err := internal.ParseServerlessPayload(buf.Bytes())
	if err != nil {
		t.Error(err)
	}
	dataShouldContain(t, data, "metric_data", "analytic_event_data", "span_event_data")
}

func TestCustomEvent(t *testing.T) {
	originalHandler := func(c context.Context) {
		if txn := newrelic.FromContext(c); nil != txn {
			txn.Application().RecordCustomEvent("myEvent", map[string]interface{}{
				"zip": "zap",
			})
		}
	}
	app := testApp(nil, t)
	wrapped := Wrap(originalHandler, app)
	w := wrapped.(*wrappedHandler)
	w.functionName = "functionName"
	buf := &bytes.Buffer{}
	w.writer = buf

	resp, err := wrapped.Invoke(context.Background(), nil)
	if nil != err || string(resp) != "null" {
		t.Error("unexpected response", err, string(resp))
	}
	app.(internal.Expect).ExpectCustomEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"type":      "myEvent",
			"timestamp": internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{
			"zip": "zap",
		},
		AgentAttributes: map[string]interface{}{},
	}})
	_, data, err := internal.ParseServerlessPayload(buf.Bytes())
	if err != nil {
		t.Error(err)
	}
	dataShouldContain(t, data, "metric_data", "analytic_event_data", "span_event_data", "custom_event_data")
}
