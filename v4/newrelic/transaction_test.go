// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"fmt"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel/api/trace"
)

func getTraceID(s trace.Span) string {
	return s.SpanContext().TraceID.String()
}

func TestInsertDistributedTraceHeadersNil(t *testing.T) {
	hdrs := http.Header{}

	var txn *Transaction
	txn.InsertDistributedTraceHeaders(hdrs)
}

func TestInsertDistributedTraceHeadersTracestate(t *testing.T) {
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

func TestAcceptDistributedTraceHeadersNil(t *testing.T) {
	hdrs := http.Header{}

	var txn *Transaction
	txn.AcceptDistributedTraceHeaders("HTTP", hdrs)
}

func TestAcceptDistributedTraceHeadersTracestate(t *testing.T) {
	remoteTraceID := "aaaa0000000000000000000000000001"
	remoteSpanID := "bbbb000000000002"

	app := newTestApp(t)
	txn := app.StartTransaction("transaction")

	hdrs := http.Header{}
	hdrs.Set("traceparent", fmt.Sprintf("00-%s-%s-01", remoteTraceID, remoteSpanID))

	txn.AcceptDistributedTraceHeaders("HTTP", hdrs)

	seg1 := txn.StartSegment("seg1")
	seg1.End()
	txn.End()

	seg1ParentID := getParentID(seg1.StartTime.Span)
	seg1TraceID := getTraceID(seg1.StartTime.Span)

	if seg1TraceID != remoteTraceID {
		t.Errorf("seg1 is does not have remote trace id: seg1TracdID=%s, remoteTraceID=%s",
			seg1TraceID, remoteTraceID)
	}
	if seg1ParentID != remoteSpanID {
		t.Errorf("seg1 is not a child of remote segment: seg1ParentID=%s, remoteSpanID=%s",
			seg1ParentID, remoteSpanID)
	}
}
