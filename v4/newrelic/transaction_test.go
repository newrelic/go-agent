// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/v4/internal"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/api/trace/testtrace"
)

func getTraceID(s trace.Span) string {
	return s.SpanContext().TraceID.String()
}

func TestInsertDistributedTraceHeadersInvalid(t *testing.T) {
	hdrs := http.Header{}

	var txn *Transaction
	txn.InsertDistributedTraceHeaders(hdrs)

	txn = &Transaction{}
	txn.InsertDistributedTraceHeaders(hdrs)
}

func TestInsertDistributedTraceHeadersTraceparent(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	seg1 := txn.StartSegment("seg1")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	seg1.End()
	txn.End()

	traceID := getTraceID(txn.rootSpan.Span)
	seg1ID := getSpanID(seg1.StartTime.Span)

	traceparent := hdrs.Get("traceparent")
	expectedTraceparent := fmt.Sprintf("00-%s-%s-00", traceID, seg1ID)

	if traceparent != expectedTraceparent {
		t.Errorf("expected traceparent '%s', got '%s'", expectedTraceparent, traceparent)
	}
}

func TestAcceptDistributedTraceHeadersInvalid(t *testing.T) {
	hdrs := http.Header{}

	var txn *Transaction
	txn.AcceptDistributedTraceHeaders("HTTP", hdrs)

	txn = &Transaction{}
	txn.AcceptDistributedTraceHeaders("HTTP", hdrs)
}

func TestAcceptDistributedTraceHeadersTraceparent(t *testing.T) {
	remoteTraceID := "aaaa0000000000000000000000000001"
	remoteSpanID := "bbbb000000000002"

	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	seg1 := txn.StartSegment("seg1")
	seg1.End()

	hdrs := http.Header{}
	hdrs.Set("traceparent", fmt.Sprintf("00-%s-%s-01", remoteTraceID, remoteSpanID))

	txn.AcceptDistributedTraceHeaders("HTTP", hdrs)

	seg2 := txn.StartSegment("seg2")
	seg2.End()
	txn.End()

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:     "seg1",
			SpanID:   "0000000000000003",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "seg2",
			SpanID:   "0000000000000004",
			TraceID:  remoteTraceID,
			ParentID: remoteSpanID,
		},
		{
			Name:     "transaction",
			SpanID:   "0000000000000002",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000000",
		},
	})
}

func TestAcceptDistributedTraceHeadersNewGoroutine(t *testing.T) {
	remoteTraceID := "aaaa0000000000000000000000000001"
	remoteSpanID := "bbbb000000000002"

	app := newTestApp(t)
	txn := app.StartTransaction("transaction")

	seg1 := txn.StartSegment("seg1")
	seg1.End()

	txnNew := txn.NewGoroutine()
	hdrs := http.Header{}
	hdrs.Set("traceparent", fmt.Sprintf("00-%s-%s-01", remoteTraceID, remoteSpanID))

	txnNew.AcceptDistributedTraceHeaders("HTTP", hdrs)

	seg2 := txnNew.StartSegment("seg2")
	seg2.End()

	txnNew.End()
	txn.End()

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:     "seg1",
			SpanID:   "0000000000000003",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "seg2",
			SpanID:   "0000000000000004",
			TraceID:  remoteTraceID,
			ParentID: remoteSpanID,
		},
		{
			Name:     "transaction",
			SpanID:   "0000000000000002",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000000",
		},
	})
}

func TestAcceptDistributedTraceHeadersNewGoroutineNoSwitchRoot(t *testing.T) {
	remoteTraceID := "aaaa0000000000000000000000000001"
	remoteSpanID := "bbbb000000000002"

	app := newTestApp(t)
	txn := app.StartTransaction("transaction")

	txnNew := txn.NewGoroutine()
	hdrs := http.Header{}
	hdrs.Set("traceparent", fmt.Sprintf("00-%s-%s-01", remoteTraceID, remoteSpanID))

	txnNew.AcceptDistributedTraceHeaders("HTTP", hdrs)
	txn.AcceptDistributedTraceHeaders("HTTP", hdrs)

	seg1 := txnNew.StartSegment("seg1")
	seg1.End()

	txnNew.End()
	txn.End()

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:     "seg1",
			SpanID:   "0000000000000003",
			TraceID:  remoteTraceID,
			ParentID: remoteSpanID,
		},
		{
			Name:     "transaction",
			SpanID:   "0000000000000002",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000000",
		},
	})
}

func TestAcceptDistributedTraceHeadersSwitchRoot(t *testing.T) {
	remoteTraceID := "aaaa0000000000000000000000000001"
	remoteSpanID := "bbbb000000000002"

	app := newTestApp(t)
	txn := app.StartTransaction("transaction")

	hdrs := http.Header{}
	hdrs.Set("traceparent", fmt.Sprintf("00-%s-%s-01", remoteTraceID, remoteSpanID))

	txn.AcceptDistributedTraceHeaders("HTTP", hdrs)

	txn.End()

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:     "transaction",
			SpanID:   "0000000000000003",
			TraceID:  remoteTraceID,
			ParentID: remoteSpanID,
		},
	})
}

func TestPropagateTracestate(t *testing.T) {
	remoteTraceID := "aaaa0000000000000000000000000001"
	remoteSpanID := "bbbb000000000002"
	remoteTracestate := "123@nr=0-0-123-456-1234567890123456-6543210987654321-0-0.24689-0"

	app := newTestApp(t)
	txn := app.StartTransaction("transaction")

	inboundHdrs := http.Header{}
	inboundHdrs.Set("traceparent", fmt.Sprintf("00-%s-%s-01", remoteTraceID, remoteSpanID))
	inboundHdrs.Set("tracestate", remoteTracestate)
	txn.AcceptDistributedTraceHeaders("HTTP", inboundHdrs)

	seg1 := txn.StartSegment("seg1")
	outboundHdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(outboundHdrs)
	seg1.End()

	txn.End()

	seg1ID := getSpanID(seg1.StartTime.Span)

	traceparent := outboundHdrs.Get("traceparent")
	tracestate := outboundHdrs.Get("tracestate")
	expectedTraceparent := fmt.Sprintf("00-%s-%s-01", remoteTraceID, seg1ID)

	if traceparent != expectedTraceparent {
		t.Errorf("expected traceparent '%s', got '%s'", expectedTraceparent, traceparent)
	}
	if tracestate != remoteTracestate {
		t.Errorf("expected traceparent '%s', got '%s'", remoteTracestate, tracestate)
	}
}

func TestInsertDistributedTraceHeadersB3(t *testing.T) {
	app, err := NewApplication(func(cfg *Config) {
		tp := testtrace.NewProvider()
		cfg.OpenTelemetry.Tracer = tp.Tracer("go-agent-test")
		cfg.OpenTelemetry.Propagators = propagation.New(
			propagation.WithInjectors(trace.B3{}),
			propagation.WithExtractors(trace.B3{}))
	})
	if err != nil {
		t.Fatal("unable to create app:", err)
	}
	txn := app.StartTransaction("transaction")
	seg1 := txn.StartSegment("seg1")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	seg1.End()
	txn.End()

	traceID := getTraceID(txn.rootSpan.Span)
	seg1ID := getSpanID(seg1.StartTime.Span)

	b3TraceID := hdrs.Get("X-B3-Traceid")
	b3SpanID := hdrs.Get("X-B3-Spanid")

	if b3TraceID != traceID {
		t.Errorf("expected X-B3-Traceid '%s', got '%s'", traceID, b3TraceID)
	}
	if b3SpanID != seg1ID {
		t.Errorf("expected X-B3-Spanid '%s', got '%s'", seg1ID, b3SpanID)
	}
}

func TestSetWebRequestAcceptDTHeaders(t *testing.T) {
	remoteTraceID := "aaaa0000000000000000000000000001"
	remoteSpanID := "bbbb000000000002"

	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	hdrs := http.Header{}
	hdrs.Set("traceparent", fmt.Sprintf("00-%s-%s-01", remoteTraceID, remoteSpanID))
	txn.SetWebRequest(WebRequest{
		Header:    hdrs,
		Transport: TransportHTTPS,
	})
	txn.End()

	txnTraceID := getTraceID(txn.rootSpan.Span)
	txnParentID := getParentID(txn.rootSpan.Span)

	if txnTraceID != remoteTraceID {
		t.Errorf("txn does not have remote trace id: txnTraceID=%s, remoteTraceID=%s",
			txnTraceID, remoteTraceID)
	}
	if txnParentID != remoteSpanID {
		t.Errorf("txn is not a child of remote segment: txnParentID=%s, remoteSpanID=%s",
			txnParentID, remoteSpanID)
	}
}

func TestSetWebRequestHTTPAcceptDTHeaders(t *testing.T) {
	remoteTraceID := "aaaa0000000000000000000000000001"
	remoteSpanID := "bbbb000000000002"

	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	hdrs := http.Header{}
	hdrs.Set("traceparent", fmt.Sprintf("00-%s-%s-01", remoteTraceID, remoteSpanID))
	txn.SetWebRequestHTTP(&http.Request{
		Header: hdrs,
	})
	txn.End()

	txnTraceID := getTraceID(txn.rootSpan.Span)
	txnParentID := getParentID(txn.rootSpan.Span)

	if txnTraceID != remoteTraceID {
		t.Errorf("txn does not have remote trace id: txnTraceID=%s, remoteTraceID=%s",
			txnTraceID, remoteTraceID)
	}
	if txnParentID != remoteSpanID {
		t.Errorf("txn is not a child of remote segment: txnParentID=%s, remoteSpanID=%s",
			txnParentID, remoteSpanID)
	}
}
