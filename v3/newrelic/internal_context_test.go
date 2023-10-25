// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
)

func TestWrapHandlerContext(t *testing.T) {
	// Test that WrapHandleFunc adds the transaction to the request's
	// context, and that it is accessible through FromContext.

	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	_, h := WrapHandleFunc(app.Application, "myTxn", func(rw http.ResponseWriter, r *http.Request) {
		txn := FromContext(r.Context())
		segment := txn.StartSegment("mySegment")
		segment.End()
	})
	req, _ := http.NewRequest("GET", "", nil)
	h(nil, req)

	scope := "WebTransaction/Go/GET myTxn"
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/GET myTxn", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/GET myTxn", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/GET myTxn", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/mySegment", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/mySegment", Scope: scope, Forced: false, Data: nil},
	})
}

func TestStartExternalSegmentNilTransaction(t *testing.T) {
	// Test that StartExternalSegment pulls the transaction from the
	// request's context if it is not explicitly provided.

	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	txn := app.StartTransaction("myTxn")

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req = RequestWithTransactionContext(req, txn)
	segment := StartExternalSegment(nil, req)
	segment.End()
	txn.End()

	scope := "OtherTransaction/Go/myTxn"
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/myTxn", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/myTxn", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/example.com/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/example.com/http/GET", Scope: scope, Forced: false, Data: nil},
	})
}

func TestNewRoundTripperNilTransaction(t *testing.T) {
	// Test that NewRoundTripper pulls the transaction from the
	// request's context if it is not explicitly provided.

	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("myTxn")

	client := &http.Client{}
	client.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 202,
		}, nil
	})
	client.Transport = NewRoundTripper(client.Transport)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req = RequestWithTransactionContext(req, txn)
	client.Do(req)
	txn.End()

	scope := "OtherTransaction/Go/myTxn"
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/example.com/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/example.com/http/GET", Scope: scope, Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/myTxn", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/myTxn", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category":  "http",
				"component": "http",
				"name":      "External/example.com/http/GET",
				"parentId":  internal.MatchAnything,
				"span.kind": "client",
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"http.method":     "GET",
				"http.statusCode": 202,
				"http.url":        "http://example.com",
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"category":         "generic",
				"name":             "OtherTransaction/Go/myTxn",
				"transaction.name": "OtherTransaction/Go/myTxn",
				"nr.entryPoint":    true,
				"sampled":          true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}
