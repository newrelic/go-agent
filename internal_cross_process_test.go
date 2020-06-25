// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"errors"

	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/cat"
)

var (
	crossProcessReplyFn = func(reply *internal.ConnectReply) {
		reply.EncodingKey = "encoding_key"
		reply.CrossProcessID = "12345#67890"
		reply.TrustedAccounts = map[int]struct{}{
			12345: {},
		}
	}
	catIntrinsics = map[string]interface{}{
		"name":                        "WebTransaction/Go/hello",
		"nr.pathHash":                 "fa013f2a",
		"nr.guid":                     internal.MatchAnything,
		"nr.referringTransactionGuid": internal.MatchAnything,
		"nr.referringPathHash":        "41c04f7d",
		"nr.apdexPerfZone":            "S",
		"client_cross_process_id":     "12345#67890",
		"nr.tripId":                   internal.MatchAnything,
	}
)

func inboundCrossProcessRequestFactory() *http.Request {
	cfgFn := func(cfg *Config) { cfg.CrossApplicationTracer.Enabled = true }
	app := testApp(crossProcessReplyFn, cfgFn, nil)
	clientTxn := app.StartTransaction("client", nil, nil)
	req, err := http.NewRequest("GET", "newrelic.com", nil)
	StartExternalSegment(clientTxn, req)
	if "" == req.Header.Get(cat.NewRelicIDName) {
		panic("missing cat header NewRelicIDName: " + req.Header.Get(cat.NewRelicIDName))
	}
	if "" == req.Header.Get(cat.NewRelicTxnName) {
		panic("missing cat header NewRelicTxnName: " + req.Header.Get(cat.NewRelicTxnName))
	}
	if nil != err {
		panic(err)
	}
	return req
}

func outboundCrossProcessResponse() http.Header {
	cfgFn := func(cfg *Config) { cfg.CrossApplicationTracer.Enabled = true }
	app := testApp(crossProcessReplyFn, cfgFn, nil)
	rw := httptest.NewRecorder()
	txn := app.StartTransaction("txn", rw, inboundCrossProcessRequestFactory())
	txn.WriteHeader(200)
	return rw.HeaderMap
}

func TestCrossProcessWriteHeaderSuccess(t *testing.T) {
	// Test that the CAT response header is present when the consumer uses
	// txn.WriteHeader.
	cfgFn := func(cfg *Config) { cfg.CrossApplicationTracer.Enabled = true }
	app := testApp(crossProcessReplyFn, cfgFn, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, inboundCrossProcessRequestFactory())
	txn.WriteHeader(200)
	txn.End()

	if "" == w.Header().Get(cat.NewRelicAppDataName) {
		t.Error(w.Header().Get(cat.NewRelicAppDataName))
	}

	app.ExpectMetrics(t, webMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: catIntrinsics,
		AgentAttributes: map[string]interface{}{
			"request.method":   "GET",
			"httpResponseCode": 200,
			"request.uri":      "newrelic.com",
		},
		UserAttributes: map[string]interface{}{},
	}})
}

func TestCrossProcessWriteSuccess(t *testing.T) {
	// Test that the CAT response header is present when the consumer uses
	// txn.Write.
	cfgFn := func(cfg *Config) { cfg.CrossApplicationTracer.Enabled = true }
	app := testApp(crossProcessReplyFn, cfgFn, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, inboundCrossProcessRequestFactory())
	txn.Write([]byte("response text"))
	txn.End()

	if "" == w.Header().Get(cat.NewRelicAppDataName) {
		t.Error(w.Header().Get(cat.NewRelicAppDataName))
	}

	app.ExpectMetrics(t, webMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: catIntrinsics,
		// Do not test attributes here:  In Go 1.5
		// response.headers.contentType will be not be present.
		AgentAttributes: nil,
		UserAttributes:  map[string]interface{}{},
	}})
}

func TestCATRoundTripper(t *testing.T) {
	cfgFn := func(cfg *Config) { cfg.CrossApplicationTracer.Enabled = true }
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello", nil, nil)
	url := "http://example.com/"
	client := &http.Client{}
	inner := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		// TODO test that request headers have been set here.
		if r.URL.String() != url {
			t.Error(r.URL.String())
		}
		return nil, errors.New("hello")
	})
	client.Transport = NewRoundTripper(txn, inner)
	resp, err := client.Get(url)
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
			"nr.guid":           internal.MatchAnything,
			"nr.tripId":         internal.MatchAnything,
			"nr.pathHash":       internal.MatchAnything,
		},
	}})
}

func TestCrossProcessLocallyDisabled(t *testing.T) {
	// Test that the CAT can be disabled by local configuration.
	cfgFn := func(cfg *Config) { cfg.CrossApplicationTracer.Enabled = false }
	app := testApp(crossProcessReplyFn, cfgFn, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, inboundCrossProcessRequestFactory())
	txn.Write([]byte("response text"))
	txn.End()

	if "" != w.Header().Get(cat.NewRelicAppDataName) {
		t.Error(w.Header().Get(cat.NewRelicAppDataName))
	}

	app.ExpectMetrics(t, webMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "S",
		},
		// Do not test attributes here:  In Go 1.5
		// response.headers.contentType will be not be present.
		AgentAttributes: nil,
		UserAttributes:  map[string]interface{}{},
	}})
}

func TestCrossProcessDisabledByServerSideConfig(t *testing.T) {
	// Test that the CAT can be disabled by server-side-config.
	cfgFn := func(cfg *Config) {}
	replyfn := func(reply *internal.ConnectReply) {
		crossProcessReplyFn(reply)
		json.Unmarshal([]byte(`{"agent_config":{"cross_application_tracer.enabled":false}}`), reply)
	}
	app := testApp(replyfn, cfgFn, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, inboundCrossProcessRequestFactory())
	txn.Write([]byte("response text"))
	txn.End()

	if "" != w.Header().Get(cat.NewRelicAppDataName) {
		t.Error(w.Header().Get(cat.NewRelicAppDataName))
	}

	app.ExpectMetrics(t, webMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "S",
		},
		// Do not test attributes here:  In Go 1.5
		// response.headers.contentType will be not be present.
		AgentAttributes: nil,
		UserAttributes:  map[string]interface{}{},
	}})
}

func TestCrossProcessEnabledByServerSideConfig(t *testing.T) {
	// Test that the CAT can be enabled by server-side-config.
	cfgFn := func(cfg *Config) { cfg.CrossApplicationTracer.Enabled = false }
	replyfn := func(reply *internal.ConnectReply) {
		crossProcessReplyFn(reply)
		json.Unmarshal([]byte(`{"agent_config":{"cross_application_tracer.enabled":true}}`), reply)
	}
	app := testApp(replyfn, cfgFn, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, inboundCrossProcessRequestFactory())
	txn.Write([]byte("response text"))
	txn.End()

	if "" == w.Header().Get(cat.NewRelicAppDataName) {
		t.Error(w.Header().Get(cat.NewRelicAppDataName))
	}

	app.ExpectMetrics(t, webMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: catIntrinsics,
		// Do not test attributes here:  In Go 1.5
		// response.headers.contentType will be not be present.
		AgentAttributes: nil,
		UserAttributes:  map[string]interface{}{},
	}})
}
