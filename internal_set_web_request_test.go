// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/newrelic/go-agent/internal"
)

type customRequest struct {
	header    http.Header
	u         *url.URL
	method    string
	transport TransportType
}

func (r customRequest) Header() http.Header      { return r.header }
func (r customRequest) URL() *url.URL            { return r.u }
func (r customRequest) Method() string           { return r.method }
func (r customRequest) Transport() TransportType { return r.transport }

var (
	sampleHTTPRequest = func() *http.Request {
		req, err := http.NewRequest("GET", "http://www.newrelic.com", nil)
		if nil != err {
			panic(err)
		}
		req.Header.Set("Accept", "myaccept")
		req.Header.Set("Content-Type", "mycontent")
		req.Header.Set("Host", "myhost")
		req.Header.Set("Content-Length", "123")
		return req
	}()
	sampleCustomRequest = func() customRequest {
		u, err := url.Parse("http://www.newrelic.com")
		if nil != err {
			panic(err)
		}
		hdr := make(http.Header)
		hdr.Set("Accept", "myaccept")
		hdr.Set("Content-Type", "mycontent")
		hdr.Set("Host", "myhost")
		hdr.Set("Content-Length", "123")
		return customRequest{
			header:    hdr,
			u:         u,
			method:    "GET",
			transport: TransportHTTP,
		}
	}()
	sampleRequestAgentAttributes = map[string]interface{}{
		AttributeRequestMethod:        "GET",
		AttributeRequestAccept:        "myaccept",
		AttributeRequestContentType:   "mycontent",
		AttributeRequestContentLength: 123,
		AttributeRequestHost:          "myhost",
		AttributeRequestURI:           "http://www.newrelic.com",
	}
)

func TestSetWebRequestNil(t *testing.T) {
	// Test that using SetWebRequest with nil marks the transaction as a web
	// transaction.
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.SetWebRequest(nil)
	if err != nil {
		t.Error("unexpected error", err)
	}
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		AgentAttributes: map[string]interface{}{},
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"guid":             internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"traceId":          internal.MatchAnything,
			"nr.apdexPerfZone": internal.MatchAnything,
		},
	}})
}

func TestSetWebRequestNilPointer(t *testing.T) {
	// Test that calling NewWebRequest with a nil pointer is safe and
	// returns a nil interface that SetWebRequest handles safely.
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.SetWebRequest(NewWebRequest(nil))
	if err != nil {
		t.Error("unexpected error", err)
	}
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		AgentAttributes: map[string]interface{}{},
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"guid":             internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"traceId":          internal.MatchAnything,
			"nr.apdexPerfZone": internal.MatchAnything,
		},
	}})
}

func TestSetWebRequestHTTPRequest(t *testing.T) {
	// Test that NewWebRequest correctly turns an *http.Request into a
	// WebRequest that SetWebRequest uses as expected.
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.SetWebRequest(NewWebRequest(sampleHTTPRequest))
	if err != nil {
		t.Error("unexpected error", err)
	}
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		AgentAttributes: sampleRequestAgentAttributes,
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"guid":             internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"traceId":          internal.MatchAnything,
			"nr.apdexPerfZone": internal.MatchAnything,
		},
	}})
}

func TestSetWebRequestCustomRequest(t *testing.T) {
	// Test that a custom type which implements WebRequest is used by
	// SetWebRequest as expected.
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.SetWebRequest(sampleCustomRequest)
	if err != nil {
		t.Error("unexpected error", err)
	}
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		AgentAttributes: sampleRequestAgentAttributes,
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"guid":             internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"traceId":          internal.MatchAnything,
			"nr.apdexPerfZone": internal.MatchAnything,
		},
	}})
}

func TestSetWebRequestAlreadyEnded(t *testing.T) {
	// Test that SetWebRequest returns an error if called after
	// Transaction.End.
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	txn.End()
	err := txn.SetWebRequest(sampleCustomRequest)
	if err != errAlreadyEnded {
		t.Error("incorrect error", err)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		AgentAttributes: map[string]interface{}{},
		Intrinsics: map[string]interface{}{
			"name":     "OtherTransaction/Go/hello",
			"guid":     internal.MatchAnything,
			"sampled":  internal.MatchAnything,
			"priority": internal.MatchAnything,
			"traceId":  internal.MatchAnything,
		},
	}})
}

func TestSetWebRequestWithDistributedTracing(t *testing.T) {
	// Test that the WebRequest.Transport() return value is used as the
	// distributed tracing transport if a distributed tracing header is
	// found in the WebRequest.Header().
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	payload := makePayload(app, nil)
	// Copy sampleCustomRequest to avoid modifying it since it is used in
	// other tests.
	req := sampleCustomRequest
	req.header = map[string][]string{
		DistributedTracePayloadHeader: {payload.Text()},
	}
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.SetWebRequest(req)
	if nil != err {
		t.Error("unexpected error", err)
	}
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: singleCount},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		AgentAttributes: map[string]interface{}{
			"request.method": "GET",
			"request.uri":    "http://www.newrelic.com",
		},
		Intrinsics: map[string]interface{}{
			"name":                     "WebTransaction/Go/hello",
			"parent.type":              "App",
			"parent.account":           "123",
			"parent.app":               "456",
			"parent.transportType":     "HTTP",
			"parent.transportDuration": internal.MatchAnything,
			"parentId":                 internal.MatchAnything,
			"traceId":                  internal.MatchAnything,
			"parentSpanId":             internal.MatchAnything,
			"guid":                     internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"priority":                 internal.MatchAnything,
			"nr.apdexPerfZone":         internal.MatchAnything,
		},
	}})
}

type incompleteRequest struct{}

func (r incompleteRequest) Header() http.Header      { return nil }
func (r incompleteRequest) URL() *url.URL            { return nil }
func (r incompleteRequest) Method() string           { return "" }
func (r incompleteRequest) Transport() TransportType { return TransportUnknown }

func TestSetWebRequestIncompleteRequest(t *testing.T) {
	// Test SetWebRequest will safely handle situations where the request's
	// URL() and Header() methods return nil.
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.SetWebRequest(incompleteRequest{})
	if err != nil {
		t.Error("unexpected error", err)
	}
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		AgentAttributes: map[string]interface{}{},
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"guid":             internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"traceId":          internal.MatchAnything,
			"nr.apdexPerfZone": internal.MatchAnything,
		},
	}})
}
