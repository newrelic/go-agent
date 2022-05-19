// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"errors"
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
)

func myErrorHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("my response"))
	// Ensure that the transaction is added to the request's context.
	txn := FromContext(req.Context())
	txn.NoticeError(myError{})
}

func TestWrapHandleFunc(t *testing.T) {
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	mux := http.NewServeMux()
	mux.HandleFunc(WrapHandleFunc(app.Application, helloPath, myErrorHandler))
	w := newCompatibleResponseRecorder()
	mux.ServeHTTP(w, helloRequest)

	out := w.Body.String()
	if "my response" != out {
		t.Error(out)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/GET /hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "WebTransaction/Go/GET /hello",
		},
		AgentAttributes: mergeAttributes(helloRequestAttributes, map[string]interface{}{
			"httpResponseCode": "200",
			"http.statusCode":  "200",
		}),
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/GET /hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/GET /hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/GET /hello", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/WebTransaction/Go/GET /hello", Scope: "", Forced: true, Data: singleCount},
	})
}

func TestWrapHandle(t *testing.T) {
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	mux := http.NewServeMux()
	mux.Handle(WrapHandle(app.Application, helloPath, http.HandlerFunc(myErrorHandler)))
	w := newCompatibleResponseRecorder()
	mux.ServeHTTP(w, helloRequest)

	out := w.Body.String()
	if "my response" != out {
		t.Error(out)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/GET /hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "WebTransaction/Go/GET /hello",
		},
		AgentAttributes: mergeAttributes(helloRequestAttributes, map[string]interface{}{
			"httpResponseCode": "200",
			"http.statusCode":  "200",
		}),
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/GET /hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/GET /hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/GET /hello", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/WebTransaction/Go/GET /hello", Scope: "", Forced: true, Data: singleCount},
	})
}

func TestWrapHandleNilApp(t *testing.T) {
	var app *Application
	mux := http.NewServeMux()
	mux.Handle(WrapHandle(app, helloPath, http.HandlerFunc(myErrorHandler)))
	w := newCompatibleResponseRecorder()
	mux.ServeHTTP(w, helloRequest)

	out := w.Body.String()
	if "my response" != out {
		t.Error(out)
	}
}

func TestRoundTripper(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello")
	url := "http://example.com/"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("zip", "zap")
	client := &http.Client{}
	inner := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		catHdr := r.Header.Get(DistributedTraceNewRelicHeader)
		if "" == catHdr {
			t.Error("cat header missing")
		}
		// Test that headers are preserved during reqest cloning:
		if z := r.Header.Get("zip"); z != "zap" {
			t.Error("missing header", z)
		}
		if r.URL.String() != url {
			t.Error(r.URL.String())
		}
		return nil, errors.New("hello")
	})
	req = RequestWithTransactionContext(req, txn)
	client.Transport = NewRoundTripper(inner)
	resp, err := client.Do(req)
	if resp != nil || err == nil {
		t.Error(resp, err.Error())
	}
	// Ensure that the request was cloned:
	catHdr := req.Header.Get(DistributedTraceNewRelicHeader)
	if "" != catHdr {
		t.Error("cat header unexpectedly present")
	}
	txn.NoticeError(myError{})
	txn.End()
	scope := "OtherTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/example.com/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/example.com/http/GET", Scope: scope, Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Data: nil},
	}, backgroundErrorMetrics...))
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":       "newrelic.myError",
			"error.message":     "my msg",
			"spanId":            "4981855ad8681d0d",
			"transactionName":   "OtherTransaction/Go/hello",
			"externalCallCount": 1,
			"externalDuration":  internal.MatchAnything,
			"guid":              internal.MatchAnything,
			"traceId":           internal.MatchAnything,
			"priority":          internal.MatchAnything,
			"sampled":           internal.MatchAnything,
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":              "OtherTransaction/Go/hello",
			"externalCallCount": 1,
			"externalDuration":  internal.MatchAnything,
			"guid":              internal.MatchAnything,
			"traceId":           internal.MatchAnything,
			"priority":          internal.MatchAnything,
			"sampled":           internal.MatchAnything,
		},
	}})
}

func TestRoundTripperOldCAT(t *testing.T) {
	cfgfn := func(c *Config) {
		c.DistributedTracer.Enabled = false
		c.CrossApplicationTracer.Enabled = true
	}

	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("hello")
	url := "http://example.com/"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{}
	inner := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		// TODO test that request headers have been set here.
		if r.URL.String() != url {
			t.Error(r.URL.String())
		}
		return nil, errors.New("hello")
	})
	req = RequestWithTransactionContext(req, txn)
	client.Transport = NewRoundTripper(inner)
	resp, err := client.Do(req)
	if resp != nil || err == nil {
		t.Error(resp, err.Error())
	}
	txn.NoticeError(myError{})
	txn.End()
	scope := "OtherTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/example.com/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/example.com/http/GET", Scope: scope, Forced: false, Data: nil},
	}, backgroundErrorMetrics...))
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":       "newrelic.myError",
			"error.message":     "my msg",
			"transactionName":   "OtherTransaction/Go/hello",
			"externalCallCount": 1,
			"externalDuration":  internal.MatchAnything,
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":              "OtherTransaction/Go/hello",
			"externalCallCount": 1,
			"externalDuration":  internal.MatchAnything,
			"nr.tripId":         internal.MatchAnything,
			"nr.guid":           internal.MatchAnything,
			"nr.pathHash":       internal.MatchAnything,
		},
	}})
}

func TestRoundTripperRace(t *testing.T) {
	// Test to detect a potential data race when using NewRoundTripper in
	// multiple goroutines.
	client := &http.Client{
		Transport: NewRoundTripper(nil),
	}
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	go client.Do(req)
	go client.Do(req)
}
