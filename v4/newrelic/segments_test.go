// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/newrelic/go-agent/v4/internal"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/api/trace/testtrace"
)

func getSpanID(s trace.Span) string {
	return s.SpanContext().SpanID.String()
}

func getParentID(s trace.Span) string {
	return s.(*testtrace.Span).ParentSpanID().String()
}

type expectApp struct {
	internal.Expect
	*Application
}

func newTestApp(t *testing.T) expectApp {
	sr := new(testtrace.StandardSpanRecorder)
	app, err := NewApplication(func(cfg *Config) {
		tr := testtrace.NewProvider(testtrace.WithSpanRecorder(sr)).Tracer("go-agent-test")
		cfg.OpenTelemetry.Tracer = tr
	})
	if err != nil {
		t.Fatal("unable to create app:", err)
	}

	return expectApp{
		Expect: &internal.OpenTelemetryExpect{
			Spans: sr,
		},
		Application: app,
	}
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

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:     "seg2",
			SpanID:   "0000000000000004",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000003",
		},
		{
			Name:     "seg3",
			SpanID:   "0000000000000005",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000003",
		},
		{
			Name:     "seg1",
			SpanID:   "0000000000000003",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "seg4",
			SpanID:   "0000000000000006",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "transaction",
			SpanID:   "0000000000000002",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000000",
		},
	})
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

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:     "seg1",
			SpanID:   "0000000000000003",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "seg3",
			SpanID:   "0000000000000005",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "seg2",
			SpanID:   "0000000000000004",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000003",
		},
		{
			Name:     "seg4",
			SpanID:   "0000000000000006",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "transaction",
			SpanID:   "0000000000000002",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000000",
		},
	})
}

func TestParentingEarlyEndingTxn(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	seg1 := txn.StartSegment("seg1")
	txn.End()
	seg1.End()
	seg2 := txn.StartSegment("seg2")
	seg2.End()

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:     "transaction",
			SpanID:   "0000000000000002",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000000",
		},
		{
			Name:     "seg1",
			SpanID:   "0000000000000003",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
	})
}

func TestParentingSegmentSiblings(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	seg1 := txn.StartSegment("seg1")
	seg1.End()
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
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "transaction",
			SpanID:   "0000000000000002",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000000",
		},
	})
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
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "seg3",
			SpanID:   "0000000000000005",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "transaction",
			SpanID:   "0000000000000002",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000000",
		},
	})
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

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:     "seg2",
			SpanID:   "0000000000000004",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000003",
		},
		{
			Name:     "seg1",
			SpanID:   "0000000000000003",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "seg4",
			SpanID:   "0000000000000006",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000005",
		},
		{
			Name:     "seg3",
			SpanID:   "0000000000000005",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "transaction",
			SpanID:   "0000000000000002",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000000",
		},
	})
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

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:    "seg2",
			SpanID:  "0000000000000005",
			TraceID: "00000000000000020000000000000000",
			// NOTE: There is currently no way for a newrelic segment to be
			// childed to an opentelemetry span.
			ParentID: "0000000000000003",
		},
		{
			Name:     "span",
			SpanID:   "0000000000000004",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000003",
		},
		{
			Name:     "seg1",
			SpanID:   "0000000000000003",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "transaction",
			SpanID:   "0000000000000002",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000000",
		},
	})
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

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			SpanID:   "0000000000000003",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "transaction",
			SpanID:   "0000000000000002",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000000",
		},
	})
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

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			SpanID:   "0000000000000003",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "transaction",
			SpanID:   "0000000000000002",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000000",
		},
	})
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
				Request: &http.Request{
					URL: &url.URL{
						Host: "myhost",
					},
				},
			},
			name: "http GET myhost",
		},
		{
			seg: &ExternalSegment{
				Request: &http.Request{
					URL: &url.URL{
						Host: "requestHost",
					},
				},
				Response: &http.Response{
					Request: &http.Request{
						URL: &url.URL{
							Host: "responseHost",
						},
					},
				},
			},
			name: "http GET responseHost",
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

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			SpanID:   "0000000000000003",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000002",
		},
		{
			Name:     "transaction",
			SpanID:   "0000000000000002",
			TraceID:  "00000000000000020000000000000000",
			ParentID: "0000000000000000",
		},
	})
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
		{
			seg: &MessageProducerSegment{
				DestinationName: "",
			},
			name: "Unknown send",
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

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Kind: "internal",
		},
		{
			Kind: "internal",
		},
		{
			Kind: "internal",
		},
		{
			Kind: "internal",
		},
		{
			Kind: "internal",
		},
	})
}

func TestStartExternalSegmentInvalid(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://request.com/", nil)

	var txn *Transaction
	StartExternalSegment(txn, req)

	txn = &Transaction{}
	StartExternalSegment(txn, req)

	app := newTestApp(t)
	txn = app.StartTransaction("transaction")
	defer txn.End()

	StartExternalSegment(txn, nil)
	StartExternalSegment(txn, &http.Request{})
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

func TestStartExternalSegmentWithTxnContext(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	ctx := NewContext(context.Background(), txn)

	req, _ := http.NewRequest("GET", "http://request.com/", nil)
	req = req.WithContext(ctx)
	seg1 := StartExternalSegment(nil, req)
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
