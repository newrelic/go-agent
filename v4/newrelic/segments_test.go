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
}

func TestParentingEarlyEndingTxn(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	txn.End()
	seg := txn.StartSegment("seg")
	seg.End()

	if seg.StartTime.span != nil {
		t.Error("seg incorrectly created a span:", seg.StartTime.span)
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

	txn.End()
}
