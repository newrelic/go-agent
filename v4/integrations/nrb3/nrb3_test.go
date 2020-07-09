// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrb3

import (
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func TestNewRoundTripperNil(t *testing.T) {
	rt := NewRoundTripper(nil)
	if orig := rt.(*b3Transport).original; orig != http.DefaultTransport {
		t.Error("original is not as expected:", orig)
	}
}

type roundTripperFn func(*http.Request) (*http.Response, error)

func (fn roundTripperFn) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestRoundTripperNoTxn(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, integrationsupport.DTEnabledCfgFn)
	txn := app.StartTransaction("test")

	var count int
	rt := NewRoundTripper(roundTripperFn(func(req *http.Request) (*http.Response, error) {
		count++
		return &http.Response{
			StatusCode: 200,
		}, nil
	}))
	client := &http.Client{Transport: rt}

	req, err := http.NewRequest("GET", "http://example.com", nil)
	if nil != err {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if nil != err {
		t.Fatal(err)
	}
	txn.End()

	if count != 1 {
		t.Error("incorrect call count to RoundTripper:", count)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/test", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/test", Scope: "", Forced: false, Data: nil},
	})
}

func TestRoundTripperWithTxnSampled(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
		reply.TraceIDGenerator = internal.NewTraceIDGenerator(123)
	}
	app := integrationsupport.NewTestApp(replyfn, integrationsupport.DTEnabledCfgFn)
	txn := app.StartTransaction("test")

	var count int
	var sent *http.Request
	rt := NewRoundTripper(roundTripperFn(func(req *http.Request) (*http.Response, error) {
		count++
		sent = req
		return &http.Response{
			StatusCode: 200,
		}, nil
	}))
	rt.(*b3Transport).idGen = internal.NewTraceIDGenerator(456)
	client := &http.Client{Transport: rt}

	req, err := http.NewRequest("GET", "http://example.com", nil)
	if nil != err {
		t.Fatal(err)
	}
	req = newrelic.RequestWithTransactionContext(req, txn)
	_, err = client.Do(req)
	if nil != err {
		t.Fatal(err)
	}
	txn.End()

	if count != 1 {
		t.Error("incorrect call count to RoundTripper:", count)
	}
	// original request is not modified
	if hdr := req.Header.Get("X-B3-TraceId"); hdr != "" {
		t.Error("original request was modified, X-B3-TraceId header set:", hdr)
	}
	// b3 headers added
	if hdr := sent.Header.Get("X-B3-TraceId"); hdr != "f1405ced8b9968baf9109259515bf702" {
		t.Error("unexpected value for X-B3-TraceId header:", hdr)
	}
	if hdr := sent.Header.Get("X-B3-SpanId"); hdr != "2e6fb48a8d962779" {
		t.Error("unexpected value for X-B3-SpanId header:", hdr)
	}
	if hdr := sent.Header.Get("X-B3-ParentSpanId"); hdr != "5a291b00ff0f4b36" {
		t.Error("unexpected value for X-B3-ParentSpanId header:", hdr)
	}
	if hdr := sent.Header.Get("X-B3-Sampled"); hdr != "1" {
		t.Error("unexpected value for X-B3-Sampled header:", hdr)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/example.com/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/example.com/http/GET", Scope: "OtherTransaction/Go/test", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/test", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/test", Scope: "", Forced: false, Data: nil},
	})
}

func TestRoundTripperWithTxnNotSampled(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleNothing()
	}
	app := integrationsupport.NewTestApp(replyfn, integrationsupport.DTEnabledCfgFn)
	txn := app.StartTransaction("test")

	var sent *http.Request
	rt := NewRoundTripper(roundTripperFn(func(req *http.Request) (*http.Response, error) {
		sent = req
		return &http.Response{
			StatusCode: 200,
		}, nil
	}))
	client := &http.Client{Transport: rt}

	req, err := http.NewRequest("GET", "http://example.com", nil)
	if nil != err {
		t.Fatal(err)
	}
	req = newrelic.RequestWithTransactionContext(req, txn)
	_, err = client.Do(req)
	if nil != err {
		t.Fatal(err)
	}
	txn.End()

	if hdr := sent.Header.Get("X-B3-Sampled"); hdr != "0" {
		t.Error("unexpected value for X-B3-Sampled header:", hdr)
	}
}
