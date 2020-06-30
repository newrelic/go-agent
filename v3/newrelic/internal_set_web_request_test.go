// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
)

var (
	sampleHTTPRequest = func() *http.Request {
		req, err := http.NewRequest("GET", "http://www.newrelic.com", nil)
		if nil != err {
			panic(err)
		}
		req.Header.Set("Accept", "myaccept")
		req.Header.Set("Content-Type", "mycontent")
		req.Header.Set("Content-Length", "123")
		//we should pull the host from the request field, not the headers
		req.Header.Set("Host", "wrongHost")
		req.Host = "myhost"
		return req
	}()
	sampleCustomRequest = func() WebRequest {
		u, err := url.Parse("http://www.newrelic.com")
		if nil != err {
			panic(err)
		}
		hdr := make(http.Header)
		hdr.Set("Accept", "myaccept")
		hdr.Set("Content-Type", "mycontent")
		hdr.Set("Content-Length", "123")
		//we should pull the host from the request field, not the headers
		hdr.Set("Host", "wrongHost")
		return WebRequest{
			Header:    hdr,
			URL:       u,
			Method:    "GET",
			Transport: TransportHTTP,
			Host:      "myhost",
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
	txn := app.StartTransaction("hello")
	txn.SetWebRequestHTTP(nil)
	app.expectNoLoggedErrors(t)
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

func TestSetWebRequestHTTPNil(t *testing.T) {
	// Test that calling NewWebRequestHTTP with a nil pointer and sets
	// the transaction as a web transaction.
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello")
	txn.SetWebRequestHTTP(nil)
	app.expectNoLoggedErrors(t)
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
	// Test that SetWebRequestHTTP uses the *http.Request as expected.
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello")
	txn.SetWebRequestHTTP(sampleHTTPRequest)
	app.expectNoLoggedErrors(t)
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
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
	txn := app.StartTransaction("hello")
	txn.End()
	txn.SetWebRequest(sampleCustomRequest)
	app.expectSingleLoggedError(t, "unable to set web request", map[string]interface{}{
		"reason": errAlreadyEnded.Error(),
	})
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
	// Test that the WebRequest.Transport value is used as the
	// distributed tracing transport if a distributed tracing header is
	// found in the WebRequest.Header.
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	hdrs := http.Header{}
	app.StartTransaction("hello").InsertDistributedTraceHeaders(hdrs)
	// Copy sampleCustomRequest to avoid modifying it since it is used in
	// other tests.
	req := sampleCustomRequest
	req.Header = hdrs
	txn := app.StartTransaction("hello")
	txn.SetWebRequest(req)
	app.expectNoLoggedErrors(t)
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
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: singleCount},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		AgentAttributes: map[string]interface{}{
			"request.method":       "GET",
			"request.uri":          "http://www.newrelic.com",
			"request.headers.host": "myhost",
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
	app.ExpectSpanEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"category":         "generic",
			"guid":             internal.MatchAnything,
			"name":             "WebTransaction/Go/hello",
			"nr.entryPoint":    true,
			"parentId":         internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"traceId":          internal.MatchAnything,
			"transaction.name": "WebTransaction/Go/hello",
			"trustedParentId":  internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"parent.account":           "123",
			"parent.app":               "456",
			"parent.transportDuration": internal.MatchAnything,
			"parent.transportType":     "HTTP",
			"parent.type":              "App",
			"request.method":           "GET",
			"request.uri":              "http://www.newrelic.com",
			"request.headers.host":     "myhost",
		},
	}})
}

func TestSetWebRequestIncompleteRequest(t *testing.T) {
	// Test SetWebRequest will safely handle situations where the request's
	// URL and Header values are nil.
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello")
	txn.SetWebRequest(WebRequest{Transport: TransportUnknown})
	app.expectNoLoggedErrors(t)
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
