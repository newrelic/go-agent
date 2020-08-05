// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"fmt"
	"net/http"
	"testing"

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

	seg2ParentID := getParentID(seg2.StartTime.Span)
	seg2TraceID := getTraceID(seg2.StartTime.Span)
	seg1TraceID := getTraceID(seg1.StartTime.Span)

	if seg2TraceID != remoteTraceID {
		t.Errorf("seg2 does not have remote trace id: seg2TracdID=%s, remoteTraceID=%s",
			seg2TraceID, remoteTraceID)
	}
	if seg2ParentID != remoteSpanID {
		t.Errorf("seg2 is not a child of remote segment: seg2ParentID=%s, remoteSpanID=%s",
			seg2ParentID, remoteSpanID)
	}
	if seg1TraceID == remoteTraceID {
		t.Errorf("seg1 does have remote trace id: seg1TracdID=%s, remoteTraceID=%s",
			seg1TraceID, remoteTraceID)
	}
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

	seg2 := txnNew.StartSegment("seg1")
	seg2.End()

	txnNew.End()
	txn.End()

	txnRootTraceID := getTraceID(txn.rootSpan.Span)
	txnNewRootTraceID := getTraceID(txnNew.rootSpan.Span)

	if txnRootTraceID == remoteTraceID {
		t.Errorf("txn root does have remote trace id: txnRootTraceID=%s, remoteTraceID=%s",
			txnRootTraceID, remoteTraceID)
	}
	if txnNewRootTraceID != txnRootTraceID {
		t.Errorf("txn root does not have same trace id as goroutine root : txnNewRootTraceID=%s, txnRootTraceID=%s",
			txnNewRootTraceID, txnRootTraceID)
	}
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

	txnRootTraceID := getTraceID(txn.rootSpan.Span)
	txnNewRootTraceID := getTraceID(txnNew.rootSpan.Span)

	if txnRootTraceID == remoteTraceID {
		t.Errorf("txn root does have remote trace id: rootTraceID=%s, remoteTraceID=%s",
			txnRootTraceID, remoteTraceID)
	}
	if txnNewRootTraceID == remoteTraceID {
		t.Errorf("txn root does have remote trace id: rootTraceID=%s, remoteTraceID=%s",
			txnNewRootTraceID, remoteTraceID)
	}
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

	rootParentID := getParentID(txn.rootSpan.Span)
	rootTraceID := getTraceID(txn.rootSpan.Span)

	if rootTraceID != remoteTraceID {
		t.Errorf("root does not have remote trace id: rootTracdID=%s, remoteTraceID=%s",
			rootTraceID, remoteTraceID)
	}
	if rootParentID != remoteSpanID {
		t.Errorf("root is not a child of remote segment: rootParentID=%s, remoteSpanID=%s",
			rootParentID, remoteSpanID)
	}
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
