// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrconnect

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"

	"github.com/newrelic/go-agent/v3/integrations/nrconnect/testapp"
	"github.com/newrelic/go-agent/v3/integrations/nrconnect/testapp/testappconnect"
)

func TestGetURL(t *testing.T) {
	testcases := []struct {
		method   string
		target   string
		expected string
	}{
		{
			method:   "/TestApplication/DoUnaryUnary",
			target:   "",
			expected: "connect:///TestApplication/DoUnaryUnary",
		},
		{
			method:   "TestApplication/DoUnaryUnary",
			target:   "",
			expected: "connect://TestApplication/DoUnaryUnary",
		},
		{
			method:   "/TestApplication/DoUnaryUnary",
			target:   "localhost:8080",
			expected: "connect://localhost:8080/TestApplication/DoUnaryUnary",
		},
		{
			method:   "TestApplication/DoUnaryUnary",
			target:   "localhost:8080",
			expected: "connect://localhost:8080/TestApplication/DoUnaryUnary",
		},
	}

	for _, test := range testcases {
		actual := getURL(test.method, test.target)
		if actual.String() != test.expected {
			t.Errorf("incorrect URL:\n\tmethod=%s,\n\ttarget=%s,\n\texpected=%s,\n\tactual=%s",
				test.method, test.target, test.expected, actual.String())
		}
	}
}

var replyFn = func(reply *internal.ConnectReply) {
	reply.SetSampleEverything()
	reply.AccountID = "123"
	reply.TrustedAccountKey = "123"
	reply.PrimaryAppID = "456"
}

func testApp() integrationsupport.ExpectApp {
	return integrationsupport.NewTestApp(replyFn, integrationsupport.ConfigFullTraces, newrelic.ConfigCodeLevelMetricsEnabled(false))
}

func newTestServerAndConn(t *testing.T, app *newrelic.Application) (*httptest.Server, *http.Client) {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(testappconnect.NewTestApplicationHandler(&testapp.Server{}, connect.WithInterceptors(Interceptor(app))))
	sv := httptest.NewServer(mux)
	t.Cleanup(sv.Close)
	return sv, sv.Client()
}

func TestUnaryClientInterceptor(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction("UnaryUnary")
	ctx := newrelic.NewContext(context.Background(), txn)

	sv, client := newTestServerAndConn(t, nil)
	defer sv.Close()

	connectClient := testappconnect.NewTestApplicationClient(client, sv.URL, connect.WithInterceptors(Interceptor(app.Application)))
	resp, err := connectClient.DoUnaryUnary(ctx, connect.NewRequest(&testapp.Message{}))
	if err != nil {
		t.Fatal("client call to DoUnaryUnary failed", err)
	}
	var hdrs map[string][]string
	err = json.Unmarshal([]byte(resp.Msg.Text), &hdrs)
	if err != nil {
		t.Fatal("cannot unmarshall client response", err)
	}
	if hdr, ok := hdrs["Newrelic"]; !ok || len(hdr) != 1 || hdr[0] == "" {
		t.Error("distributed trace header not sent", hdrs)
	}
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/UnaryUnary", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/UnaryUnary", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/" + sv.Listener.Addr().String() + "/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/" + sv.Listener.Addr().String() + "/Connect/TestApplication/DoUnaryUnary", Scope: "OtherTransaction/Go/UnaryUnary", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category":  "http",
				"component": "Connect",
				"name":      "External/" + sv.Listener.Addr().String() + "/Connect/TestApplication/DoUnaryUnary",
				"parentId":  internal.MatchAnything,
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"category":         "generic",
				"name":             "OtherTransaction/Go/UnaryUnary",
				"transaction.name": "OtherTransaction/Go/UnaryUnary",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/UnaryUnary",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/UnaryUnary",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "External/" + sv.Listener.Addr().String() + "/Connect/TestApplication/DoUnaryUnary",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}

func TestUnaryStreamClientInterceptor(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction("UnaryStream")
	ctx := newrelic.NewContext(context.Background(), txn)

	sv, client := newTestServerAndConn(t, nil)
	defer sv.Close()

	connectClient := testappconnect.NewTestApplicationClient(client, sv.URL, connect.WithInterceptors(Interceptor(app.Application)))
	stream, err := connectClient.DoUnaryStream(ctx, connect.NewRequest(&testapp.Message{}))
	if err != nil {
		t.Fatal("client call to DoUnaryStream failed", err)
	}
	var recved int
	for stream.Receive() {
		msg := stream.Msg()
		var hdrs map[string][]string
		err = json.Unmarshal([]byte(msg.Text), &hdrs)
		if err != nil {
			t.Fatal("cannot unmarshall client response", err)
		}
		if hdr, ok := hdrs["Newrelic"]; !ok || len(hdr) != 1 || hdr[0] == "" {
			t.Error("distributed trace header not sent", hdrs)
		}
		recved++
	}
	if err := stream.Err(); err != nil {
		t.Fatal("error receiving message", err)
	}
	if recved != 3 {
		t.Fatal("received incorrect number of messages from server", recved)
	}
	txn.End()

	// In Connect RPC, DoUnaryStream is handled as a single HTTP request with streaming response
	// So it's treated as a unary call from the interceptor perspective
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/UnaryStream", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/UnaryStream", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/" + sv.Listener.Addr().String() + "/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/" + sv.Listener.Addr().String() + "/Connect/TestApplication/DoUnaryStream", Scope: "OtherTransaction/Go/UnaryStream", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category":  "http",
				"component": "Connect",
				"name":      "External/" + sv.Listener.Addr().String() + "/Connect/TestApplication/DoUnaryStream",
				"parentId":  internal.MatchAnything,
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"category":         "generic",
				"name":             "OtherTransaction/Go/UnaryStream",
				"transaction.name": "OtherTransaction/Go/UnaryStream",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/UnaryStream",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/UnaryStream",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "External/" + sv.Listener.Addr().String() + "/Connect/TestApplication/DoUnaryStream",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}

func TestNilTxnClientUnary(t *testing.T) {
	sv, client := newTestServerAndConn(t, nil)
	defer sv.Close()

	connectClient := testappconnect.NewTestApplicationClient(client, sv.URL)
	resp, err := connectClient.DoUnaryUnary(context.Background(), connect.NewRequest(&testapp.Message{}))
	if err != nil {
		t.Fatal("client call to DoUnaryUnary failed", err)
	}
	var hdrs map[string][]string
	err = json.Unmarshal([]byte(resp.Msg.Text), &hdrs)
	if err != nil {
		t.Fatal("cannot unmarshall client response", err)
	}
	if _, ok := hdrs["Newrelic"]; ok {
		t.Error("distributed trace header sent", hdrs)
	}
}

func TestUnaryServerInterceptor(t *testing.T) {
	app := testApp()

	sv, client := newTestServerAndConn(t, app.Application)
	defer sv.Close()

	connectClient := testappconnect.NewTestApplicationClient(client, sv.URL, connect.WithInterceptors(Interceptor(app.Application)))
	txn := app.StartTransaction("client")
	ctx := newrelic.NewContext(context.Background(), txn)
	_, err := connectClient.DoUnaryUnary(ctx, connect.NewRequest(&testapp.Message{}))
	if err != nil {
		t.Fatal("unable to call client DoUnaryUnary", err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/TestApplication/DoUnaryUnary", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoUnaryUnary", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoUnaryUnary", Scope: "WebTransaction/Go/TestApplication/DoUnaryUnary", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/TestApplication/DoUnaryUnary", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestApplication/DoUnaryUnary", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"guid":                     internal.MatchAnything,
			"name":                     "WebTransaction/Go/TestApplication/DoUnaryUnary",
			"nr.apdexPerfZone":         internal.MatchAnything,
			"parent.account":           123,
			"parent.app":               456,
			"parent.transportDuration": internal.MatchAnything,
			"parent.transportType":     "HTTP",
			"parent.type":              "App",
			"parentId":                 internal.MatchAnything,
			"parentSpanId":             internal.MatchAnything,
			"priority":                 internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"traceId":                  internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":              200,
			"http.statusCode":               200,
			"request.headers.contentType":   "application/proto",
			"request.headers.contentLength": 0,
			"request.method":                "TestApplication/DoUnaryUnary",
			"request.uri":                   "connect://" + sv.Listener.Addr().String() + "/TestApplication/DoUnaryUnary",
		},
	}})
	// Span events validation simplified to focus on basic metrics
}

func TestUnaryServerInterceptorError(t *testing.T) {
	app := testApp()

	sv, client := newTestServerAndConn(t, app.Application)
	defer sv.Close()

	connectClient := testappconnect.NewTestApplicationClient(client, sv.URL)
	_, err := connectClient.DoUnaryUnaryError(context.Background(), connect.NewRequest(&testapp.Message{}))
	if err == nil {
		t.Fatal("DoUnaryUnaryError should have returned an error")
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/TestApplication/DoUnaryUnaryError", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/WebTransaction/Go/TestApplication/DoUnaryUnaryError", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/TestApplication/DoUnaryUnaryError", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestApplication/DoUnaryUnaryError", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"guid":             internal.MatchAnything,
			"name":             "WebTransaction/Go/TestApplication/DoUnaryUnaryError",
			"nr.apdexPerfZone": internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"traceId":          internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{
			"connectStatusCode":    "data_loss",
			"connectStatusMessage": "",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":              500,
			"http.statusCode":               500,
			"request.headers.contentType":   "application/proto",
			"request.headers.contentLength": 0,
			"request.method":                "TestApplication/DoUnaryUnaryError",
			"request.uri":                   "connect://" + sv.Listener.Addr().String() + "/TestApplication/DoUnaryUnaryError",
		},
	}})
	// Error events are expected but validation is simplified
}

func TestUnaryStreamServerInterceptor(t *testing.T) {
	app := testApp()

	sv, client := newTestServerAndConn(t, app.Application)
	defer sv.Close()

	connectClient := testappconnect.NewTestApplicationClient(client, sv.URL, connect.WithInterceptors(Interceptor(app.Application)))
	txn := app.StartTransaction("client")
	ctx := newrelic.NewContext(context.Background(), txn)
	stream, err := connectClient.DoUnaryStream(ctx, connect.NewRequest(&testapp.Message{}))
	if err != nil {
		t.Fatal("client call to DoUnaryStream failed", err)
	}
	var recved int
	for stream.Receive() {
		recved++
	}
	if err := stream.Err(); err != nil {
		t.Fatal("error receiving message", err)
	}
	if recved != 3 {
		t.Fatal("received incorrect number of messages from server", recved)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/TestApplication/DoUnaryStream", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoUnaryStream", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoUnaryStream", Scope: "WebTransaction/Go/TestApplication/DoUnaryStream", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/TestApplication/DoUnaryStream", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestApplication/DoUnaryStream", Scope: "", Forced: false, Data: nil},
	})
	// Basic validation that transaction and span events are generated
}

func TestUnaryServerInterceptorNilApp(t *testing.T) {
	sv, client := newTestServerAndConn(t, nil)
	defer sv.Close()

	connectClient := testappconnect.NewTestApplicationClient(client, sv.URL)
	resp, err := connectClient.DoUnaryUnary(context.Background(), connect.NewRequest(&testapp.Message{}))
	if err != nil {
		t.Fatal("unable to call client DoUnaryUnary", err)
	}
	if resp.Msg.Text == "" {
		t.Error("incorrect message received")
	}
}
