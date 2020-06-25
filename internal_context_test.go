// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// +build go1.7

package newrelic

import (
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/internal"
)

func TestWrapHandlerContext(t *testing.T) {
	// Test that WrapHandleFunc adds the transaction to the request's
	// context, and that it is accessible through FromContext.

	app := testApp(nil, nil, t)
	_, h := WrapHandleFunc(app, "myTxn", func(rw http.ResponseWriter, r *http.Request) {
		txn := FromContext(r.Context())
		segment := StartSegment(txn, "mySegment")
		segment.End()
	})
	req, _ := http.NewRequest("GET", "", nil)
	h(nil, req)

	scope := "WebTransaction/Go/myTxn"
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/myTxn", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/myTxn", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/myTxn", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/mySegment", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/mySegment", Scope: scope, Forced: false, Data: nil},
	})
}

func TestStartExternalSegmentNilTransaction(t *testing.T) {
	// Test that StartExternalSegment pulls the transaction from the
	// request's context if it is not explicitly provided.

	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myTxn", nil, nil)

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

	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myTxn", nil, nil)

	client := &http.Client{}
	client.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{}, nil
	})
	client.Transport = NewRoundTripper(nil, client.Transport)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req = RequestWithTransactionContext(req, txn)
	client.Do(req)
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
