// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
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

func newTestApp(t *testing.T) *Application {
	app, err := NewApplication(func(cfg *Config) {
		cfg.OpenTelemetry.Tracer = testtrace.NewTracer()
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
	seg1.End()
	txn.End()

	txnID := getSpanID(txn.rootSpan.Span)
	seg1ID := getSpanID(seg1.StartTime.Span)
	seg1ParentID := getParentID(seg1.StartTime.Span)
	seg2ParentID := getParentID(seg2.StartTime.Span)

	if seg2ParentID != seg1ID {
		t.Errorf("seg2 is not a child of seg1: seg2ParentID=%s, seg1ID=%s",
			seg2ParentID, seg1ID)
	}
	if seg1ParentID != txnID {
		t.Errorf("seg1 is not a child of txn: seg1ParentID=%s, txnID=%s",
			seg1ParentID, txnID)
	}
}

func TestParentingSegmentEndOrder(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	seg1 := txn.StartSegment("seg1")
	seg2 := txn.StartSegment("seg2")
	seg1.End()
	seg2.End()
	txn.End()

	txnID := getSpanID(txn.rootSpan.Span)
	seg1ID := getSpanID(seg1.StartTime.Span)
	seg1ParentID := getParentID(seg1.StartTime.Span)
	seg2ParentID := getParentID(seg2.StartTime.Span)

	if seg2ParentID != seg1ID {
		t.Errorf("seg2 is not a child of seg1: seg2ParentID=%s, seg1ID=%s",
			seg2ParentID, seg1ID)
	}
	if seg1ParentID != txnID {
		t.Errorf("seg1 is not a child of txn: seg1ParentID=%s, txnID=%s",
			seg1ParentID, txnID)
	}
}

func TestParentingEarlyEndingTxn(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	seg1 := txn.StartSegment("seg1")
	seg2 := txn.StartSegment("seg2")
	txn.End()
	seg2.End()
	seg1.End()

	txnID := getSpanID(txn.rootSpan.Span)
	seg1ID := getSpanID(seg1.StartTime.Span)
	seg1ParentID := getParentID(seg1.StartTime.Span)
	seg2ParentID := getParentID(seg2.StartTime.Span)

	if seg2ParentID != seg1ID {
		t.Errorf("seg2 is not a child of seg1: seg2ParentID=%s, seg1ID=%s",
			seg2ParentID, seg1ID)
	}
	if seg1ParentID != txnID {
		t.Errorf("seg1 is not a child of txn: seg1ParentID=%s, txnID=%s",
			seg1ParentID, txnID)
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
}
