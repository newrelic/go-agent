// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/api/trace/testtrace"
)

func getSpanID(s trace.Span) string {
	return s.SpanContext().SpanID.String()
}

func getParentID(s trace.Span) string {
	return s.(*testtrace.Span).ParentSpanID().String()
}

func spanHasEnded(s trace.Span) bool {
	return s.(*testtrace.Span).Ended()
}

func getSpanName(s trace.Span) string {
	return s.(*testtrace.Span).Name()
}

func getSpanKind(s trace.Span) trace.SpanKind {
	return s.(*testtrace.Span).SpanKind()
}

func newTestApp(t *testing.T) *Application {
	app, err := NewApplication(func(cfg *Config) {
		tp := testtrace.NewProvider()
		cfg.OpenTelemetry.Tracer = tp.Tracer("go-agent-test")
	})
	if err != nil {
		t.Fatal("unable to create app:", err)
	}
	return app
}

func TestParentingSimple(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	seg1 := txn.StartSegment("seg1")
	seg2 := txn.StartSegment("seg2")
	seg2.End()
	seg3 := txn.StartSegment("seg3")
	seg3.End()
	seg1.End()
	seg4 := txn.StartSegment("seg4")
	seg4.End()
	txn.End()

	txnID := getSpanID(txn.rootSpan.Span)
	seg1ID := getSpanID(seg1.StartTime.Span)
	seg1ParentID := getParentID(seg1.StartTime.Span)
	seg2ParentID := getParentID(seg2.StartTime.Span)
	seg3ParentID := getParentID(seg3.StartTime.Span)
	seg4ParentID := getParentID(seg4.StartTime.Span)

	if seg1ParentID != txnID {
		t.Errorf("seg1 is not a child of txn: seg1ParentID=%s, txnID=%s",
			seg1ParentID, txnID)
	}
	if seg2ParentID != seg1ID {
		t.Errorf("seg2 is not a child of seg1: seg2ParentID=%s, seg1ID=%s",
			seg2ParentID, seg1ID)
	}
	if seg3ParentID != seg1ID {
		t.Errorf("seg3 is not a child of seg1: seg3ParentID=%s, seg1ID=%s",
			seg3ParentID, seg1ID)
	}
	if seg4ParentID != txnID {
		t.Errorf("seg4 is not a child of txn: seg4ParentID=%s, txnID=%s",
			seg4ParentID, txnID)
	}
	if !spanHasEnded(txn.rootSpan.Span) {
		t.Error("txn root span wasn't ended")
	}
	if !spanHasEnded(seg1.StartTime.Span) {
		t.Error("seg1 wasn't ended")
	}
	if !spanHasEnded(seg2.StartTime.Span) {
		t.Error("seg2 wasn't ended")
	}
	if !spanHasEnded(seg3.StartTime.Span) {
		t.Error("seg3 wasn't ended")
	}
	if !spanHasEnded(seg4.StartTime.Span) {
		t.Error("seg4 wasn't ended")
	}
}

func TestParentingSegmentEndOrder(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	seg1 := txn.StartSegment("seg1")
	seg2 := txn.StartSegment("seg2")
	seg1.End()
	seg3 := txn.StartSegment("seg3")
	seg3.End()
	seg2.End()
	seg4 := txn.StartSegment("seg4")
	seg4.End()
	txn.End()

	txnID := getSpanID(txn.rootSpan.Span)
	seg1ID := getSpanID(seg1.StartTime.Span)
	seg1ParentID := getParentID(seg1.StartTime.Span)
	seg2ParentID := getParentID(seg2.StartTime.Span)
	seg3ParentID := getParentID(seg3.StartTime.Span)
	seg4ParentID := getParentID(seg4.StartTime.Span)

	if seg1ParentID != txnID {
		t.Errorf("seg1 is not a child of txn: seg1ParentID=%s, txnID=%s",
			seg1ParentID, txnID)
	}
	if seg2ParentID != seg1ID {
		t.Errorf("seg2 is not a child of seg1: seg2ParentID=%s, seg1ID=%s",
			seg2ParentID, seg1ID)
	}
	if seg3ParentID != txnID {
		t.Errorf("seg3 is not a child of txn: seg3ParentID=%s, txnID=%s",
			seg3ParentID, txnID)
	}
	if seg4ParentID != txnID {
		t.Errorf("seg4 is not a child of txn: seg4ParentID=%s, txnID=%s",
			seg4ParentID, txnID)
	}
	if !spanHasEnded(txn.rootSpan.Span) {
		t.Error("txn root span wasn't ended")
	}
	if !spanHasEnded(seg1.StartTime.Span) {
		t.Error("seg1 wasn't ended")
	}
	if !spanHasEnded(seg2.StartTime.Span) {
		t.Error("seg2 wasn't ended")
	}
	if !spanHasEnded(seg3.StartTime.Span) {
		t.Error("seg3 wasn't ended")
	}
	if !spanHasEnded(seg4.StartTime.Span) {
		t.Error("seg4 wasn't ended")
	}
}

func TestParentingEarlyEndingTxn(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	seg1 := txn.StartSegment("seg1")
	txn.End()
	seg1.End()
	seg2 := txn.StartSegment("seg2")
	seg2.End()

	txnID := getSpanID(txn.rootSpan.Span)
	seg1ParentID := getParentID(seg1.StartTime.Span)

	if seg1ParentID != txnID {
		t.Errorf("seg1 is not a child of txn: seg1ParentID=%s, txnID=%s",
			seg1ParentID, txnID)
	}
	if seg2.StartTime.span != nil {
		t.Error("seg2 incorrectly created a span:", seg2.StartTime.span)
	}
	if !spanHasEnded(txn.rootSpan.Span) {
		t.Error("txn root span wasn't ended")
	}
	if !spanHasEnded(seg1.StartTime.Span) {
		t.Error("seg1 wasn't ended")
	}
}

func TestParentingSegmentSiblings(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	seg1 := txn.StartSegment("seg1")
	seg1.End()
	seg2 := txn.StartSegment("seg2")
	seg2.End()
	txn.End()

	txnID := getSpanID(txn.rootSpan.Span)
	seg1ParentID := getParentID(seg1.StartTime.Span)
	seg2ParentID := getParentID(seg2.StartTime.Span)

	if seg1ParentID != txnID {
		t.Errorf("seg1 is not a child of txn: seg1ParentID=%s, txnID=%s",
			seg1ParentID, txnID)
	}
	if seg2ParentID != txnID {
		t.Errorf("seg2 is not a child of txn: seg2ParentID=%s, txnID=%s",
			seg2ParentID, txnID)
	}
	if !spanHasEnded(txn.rootSpan.Span) {
		t.Error("txn root span wasn't ended")
	}
	if !spanHasEnded(seg1.StartTime.Span) {
		t.Error("seg1 wasn't ended")
	}
	if !spanHasEnded(seg2.StartTime.Span) {
		t.Error("seg2 wasn't ended")
	}
}

func TestParentingNewGoroutine(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")

	txn1 := txn.NewGoroutine()
	seg1 := txn1.StartSegment("seg1")
	txn2 := txn.NewGoroutine()
	seg2 := txn2.StartSegment("seg2")
	seg3 := txn.StartSegment("seg3")
	seg1.End()
	seg2.End()
	seg3.End()
	txn.End()

	txnID := getSpanID(txn.rootSpan.Span)
	seg1ParentID := getParentID(seg1.StartTime.Span)
	seg2ParentID := getParentID(seg2.StartTime.Span)
	seg3ParentID := getParentID(seg3.StartTime.Span)

	if seg1ParentID != txnID {
		t.Errorf("seg1 is not a child of txn: seg1ParentID=%s, txnID=%s",
			seg1ParentID, txnID)
	}
	if seg2ParentID != txnID {
		t.Errorf("seg2 is not a child of txn: seg2ParentID=%s, txnID=%s",
			seg2ParentID, txnID)
	}
	if seg3ParentID != txnID {
		t.Errorf("seg3 is not a child of txn: seg3ParentID=%s, txnID=%s",
			seg3ParentID, txnID)
	}
	if !spanHasEnded(txn.rootSpan.Span) {
		t.Error("txn root span wasn't ended")
	}
	if !spanHasEnded(seg1.StartTime.Span) {
		t.Error("seg1 wasn't ended")
	}
	if !spanHasEnded(seg2.StartTime.Span) {
		t.Error("seg2 wasn't ended")
	}
	if !spanHasEnded(seg3.StartTime.Span) {
		t.Error("seg3 wasn't ended")
	}
}

func TestParentingDoubleEndingSegments(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	seg1 := txn.StartSegment("seg1")
	seg2 := txn.StartSegment("seg2")
	seg2.End()
	seg1.End()
	seg3 := txn.StartSegment("seg3")
	seg2.End() // End seg2 a second time
	seg4 := txn.StartSegment("seg4")
	seg4.End()
	seg3.End()
	txn.End()
	txn.End() // End txn a second time

	txnID := getSpanID(txn.rootSpan.Span)
	seg1ID := getSpanID(seg1.StartTime.Span)
	seg3ID := getSpanID(seg3.StartTime.Span)
	seg1ParentID := getParentID(seg1.StartTime.Span)
	seg2ParentID := getParentID(seg2.StartTime.Span)
	seg3ParentID := getParentID(seg3.StartTime.Span)
	seg4ParentID := getParentID(seg4.StartTime.Span)

	if seg1ParentID != txnID {
		t.Errorf("seg1 is not a child of txn: seg1ParentID=%s, txnID=%s",
			seg1ParentID, txnID)
	}
	if seg2ParentID != seg1ID {
		t.Errorf("seg2 is not a child of seg1: seg2ParentID=%s, seg1ID=%s",
			seg2ParentID, seg1ID)
	}
	if seg3ParentID != txnID {
		t.Errorf("seg3 is not a child of txn: seg3ParentID=%s, txnID=%s",
			seg3ParentID, txnID)
	}
	if seg4ParentID != seg3ID {
		t.Errorf("seg4 is not a child of seg3: seg4ParentID=%s, seg3ID=%s",
			seg4ParentID, seg3ID)
	}
	if !spanHasEnded(txn.rootSpan.Span) {
		t.Error("txn root span wasn't ended")
	}
	if !spanHasEnded(seg1.StartTime.Span) {
		t.Error("seg1 wasn't ended")
	}
	if !spanHasEnded(seg2.StartTime.Span) {
		t.Error("seg2 wasn't ended")
	}
	if !spanHasEnded(seg3.StartTime.Span) {
		t.Error("seg3 wasn't ended")
	}
	if !spanHasEnded(seg4.StartTime.Span) {
		t.Error("seg4 wasn't ended")
	}
}

func TestParentingWithOTelAPI(t *testing.T) {
	app := newTestApp(t)
	tracer := app.tracer
	txn := app.StartTransaction("transaction")
	seg1 := txn.StartSegment("seg1")
	ctx := NewContext(context.Background(), txn)
	_, span := tracer.Start(ctx, "span")
	seg2 := txn.StartSegment("seg2")
	seg2.End()
	span.End()
	seg1.End()
	txn.End()

	txnID := getSpanID(txn.rootSpan.Span)
	seg1ID := getSpanID(seg1.StartTime.Span)
	seg1ParentID := getParentID(seg1.StartTime.Span)
	seg2ParentID := getParentID(seg2.StartTime.Span)
	spanParentID := getParentID(span)

	if seg1ParentID != txnID {
		t.Errorf("seg1 is not a child of txn: seg1ParentID=%s, txnID=%s",
			seg1ParentID, txnID)
	}
	if spanParentID != seg1ID {
		t.Errorf("span is not a child of seg1: spanParentID=%s, seg1ID=%s",
			spanParentID, seg1ID)
	}
	// NOTE: There is currently no way for a newrelic segment to be childed to
	// an opentelemetry span.
	if seg2ParentID != seg1ID {
		t.Errorf("seg2 is not a child of txn: seg2ParentID=%s, seg1ID=%s",
			seg2ParentID, seg1ID)
	}
	if !spanHasEnded(txn.rootSpan.Span) {
		t.Error("txn root span wasn't ended")
	}
	if !spanHasEnded(seg1.StartTime.Span) {
		t.Error("seg1 wasn't ended")
	}
	if !spanHasEnded(span) {
		t.Error("span wasn't ended")
	}
	if !spanHasEnded(seg2.StartTime.Span) {
		t.Error("seg2 wasn't ended")
	}
}

func TestNilSegment(t *testing.T) {
	// Ensure nil segments do not panic
	var seg *Segment
	seg.AddAttribute("hello", "world")
	seg.End()
}

func TestSegmentsOnNilTransaction(t *testing.T) {
	// Ensure segments on nil transactions do not panic
	var txn *Transaction
	seg := txn.StartSegment("seg")
	seg.End()
	txn.End()
}

func TestSegmentsOnEmptyTransaction(t *testing.T) {
	// Ensure segments on empty transactions do not panic
	txn := &Transaction{}
	seg := txn.StartSegment("seg")
	seg.End()
	txn.End()
}

func TestParentingDatastoreSegment(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	seg := &DatastoreSegment{
		StartTime: txn.StartSegmentNow(),
	}
	seg.End()
	txn.End()

	txnID := getSpanID(txn.rootSpan.Span)
	segParentID := getParentID(seg.StartTime.Span)

	if segParentID != txnID {
		t.Errorf("seg is not a child of txn: segParentID=%s, txnID=%s",
			segParentID, txnID)
	}
	if !spanHasEnded(txn.rootSpan.Span) {
		t.Error("txn root span wasn't ended")
	}
	if !spanHasEnded(seg.StartTime.Span) {
		t.Error("seg wasn't ended")
	}
}

func TestDatastoreSegmentNaming(t *testing.T) {
	testcases := []struct {
		seg  *DatastoreSegment
		name string
	}{
		{
			seg: &DatastoreSegment{
				Product:            DatastorePostgres,
				Collection:         "collection",
				Operation:          "operation",
				ParameterizedQuery: "parametrized query",
				QueryParameters: map[string]interface{}{
					"query": "param",
				},
				Host:         "host",
				PortPathOrID: "port",
				DatabaseName: "dbname",
			},
			name: "parametrized query",
		},
		{
			seg: &DatastoreSegment{
				Product:            DatastorePostgres,
				Collection:         "collection",
				Operation:          "operation",
				ParameterizedQuery: "",
				QueryParameters: map[string]interface{}{
					"query": "param",
				},
				Host:         "host",
				PortPathOrID: "port",
				DatabaseName: "dbname",
			},
			name: "'operation' on 'collection' using 'Postgres'",
		},
		{
			seg: &DatastoreSegment{
				Product:            DatastorePostgres,
				Collection:         "",
				Operation:          "operation",
				ParameterizedQuery: "",
				QueryParameters: map[string]interface{}{
					"query": "param",
				},
				Host:         "host",
				PortPathOrID: "port",
				DatabaseName: "dbname",
			},
			name: "'operation' on 'unknown' using 'Postgres'",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if name := tc.seg.name(); name != tc.name {
				t.Errorf(`incorrect name: actual="%s" expected="%s"`, name, tc.name)
			}
		})
	}
}

func TestParentingExternalSegment(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	seg := &ExternalSegment{
		StartTime: txn.StartSegmentNow(),
	}
	seg.End()
	txn.End()

	txnID := getSpanID(txn.rootSpan.Span)
	segParentID := getParentID(seg.StartTime.Span)

	if segParentID != txnID {
		t.Errorf("seg is not a child of txn: segParentID=%s, txnID=%s",
			segParentID, txnID)
	}
	if !spanHasEnded(txn.rootSpan.Span) {
		t.Error("txn root span wasn't ended")
	}
	if !spanHasEnded(seg.StartTime.Span) {
		t.Error("seg wasn't ended")
	}
}

func TestExternalSegmentNaming(t *testing.T) {
	testcases := []struct {
		seg  *ExternalSegment
		name string
	}{
		{
			seg:  &ExternalSegment{},
			name: "http unknown unknown",
		},
		{
			seg: &ExternalSegment{
				Host: "myhost:1234",
			},
			name: "http unknown myhost:1234",
		},
		{
			seg: &ExternalSegment{
				URL: "http://myhost:1234/path",
			},
			name: "http unknown myhost:1234",
		},
		{
			seg: &ExternalSegment{
				URL: "this is not a url",
			},
			name: "http unknown unknown",
		},
		{
			seg: &ExternalSegment{
				Procedure: "procedure",
			},
			name: "http procedure unknown",
		},
		{
			seg: &ExternalSegment{
				Library: "gRPC",
			},
			name: "gRPC unknown unknown",
		},
		{
			seg: &ExternalSegment{
				Request: &http.Request{},
			},
			name: "http GET unknown",
		},
		{
			seg: &ExternalSegment{
				Request: &http.Request{
					Method: "POST",
				},
			},
			name: "http POST unknown",
		},
		{
			seg: &ExternalSegment{
				Request: &http.Request{
					Method: "POST",
				},
				Response: &http.Response{},
			},
			name: "http POST unknown",
		},
		{
			seg: &ExternalSegment{
				Request: &http.Request{
					Method: "POST",
				},
				Response: &http.Response{
					Request: &http.Request{
						Method: "PUT",
					},
				},
			},
			name: "http PUT unknown",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if name := tc.seg.name(); name != tc.name {
				t.Errorf(`incorrect name: actual="%s" expected="%s"`, name, tc.name)
			}
		})
	}
}

func TestParentingMessageProducerSegment(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	seg := &MessageProducerSegment{
		StartTime: txn.StartSegmentNow(),
	}
	seg.End()
	txn.End()

	txnID := getSpanID(txn.rootSpan.Span)
	segParentID := getParentID(seg.StartTime.Span)

	if segParentID != txnID {
		t.Errorf("seg is not a child of txn: segParentID=%s, txnID=%s",
			segParentID, txnID)
	}
	if !spanHasEnded(txn.rootSpan.Span) {
		t.Error("txn root span wasn't ended")
	}
	if !spanHasEnded(seg.StartTime.Span) {
		t.Error("seg wasn't ended")
	}
}

func TestMessageProducerSegmentNaming(t *testing.T) {
	testcases := []struct {
		seg  *MessageProducerSegment
		name string
	}{
		{
			seg: &MessageProducerSegment{
				DestinationName:      "destination",
				DestinationTemporary: false,
			},
			name: "destination send",
		},
		{
			seg: &MessageProducerSegment{
				DestinationName:      "destination",
				DestinationTemporary: true,
			},
			name: "(temporary) send",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if name := tc.seg.name(); name != tc.name {
				t.Errorf(`incorrect name: actual="%s" expected="%s"`, name, tc.name)
			}
		})
	}
}

func TestSpanKind(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	segBasic := txn.StartSegment("basic")
	segBasic.End()
	segDatastore := &DatastoreSegment{
		StartTime: txn.StartSegmentNow(),
	}
	segDatastore.End()
	segExternal := &ExternalSegment{
		StartTime: txn.StartSegmentNow(),
	}
	segExternal.End()
	segMessage := &MessageProducerSegment{
		StartTime: txn.StartSegmentNow(),
	}
	segMessage.End()
	txn.End()

	if kind := getSpanKind(txn.rootSpan.Span); kind != trace.SpanKindInternal {
		t.Errorf("txn has incorrect SpanKind: %s", kind)
	}
	if kind := getSpanKind(segBasic.StartTime.Span); kind != trace.SpanKindInternal {
		t.Errorf("segBasic has incorrect SpanKind: %s", kind)
	}
	if kind := getSpanKind(segDatastore.StartTime.Span); kind != trace.SpanKindInternal {
		t.Errorf("segDatastore has incorrect SpanKind: %s", kind)
	}
	if kind := getSpanKind(segExternal.StartTime.Span); kind != trace.SpanKindInternal {
		t.Errorf("segExternal has incorrect SpanKind: %s", kind)
	}
	if kind := getSpanKind(segMessage.StartTime.Span); kind != trace.SpanKindInternal {
		t.Errorf("segMessage has incorrect SpanKind: %s", kind)
	}
}

func TestStartExternalSegment(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")

	req, _ := http.NewRequest("GET", "http://request.com/", nil)
	seg1 := StartExternalSegment(txn, req)
	seg1.End()

	txn.End()

	traceID := getTraceID(txn.rootSpan.Span)
	seg1ID := getSpanID(seg1.StartTime.Span)

	traceparent := req.Header.Get("traceparent")
	expectedTraceparent := fmt.Sprintf("00-%s-%s-00", traceID, seg1ID)

	if traceparent != expectedTraceparent {
		t.Errorf("expected traceparent '%s', got '%s'", expectedTraceparent, traceparent)
	}
}
