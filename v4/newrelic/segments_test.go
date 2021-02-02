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
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
)

func getSpanID(s trace.Span) string {
	return s.SpanContext().SpanID.String()
}

func getParentID(s trace.Span) string {
	return s.(*oteltest.Span).ParentSpanID().String()
}

type expectApp struct {
	internal.Expect
	*Application
}

func newTestApp(t *testing.T) expectApp {
	sr := new(oteltest.StandardSpanRecorder)
	app, err := NewApplication(func(cfg *Config) {
		tr := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr)).Tracer("go-agent-test")
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
			name: "'operation' on 'collection' using 'postgresql'",
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
			name: "'operation' on 'unknown' using 'postgresql'",
		},
		{
			seg:  &DatastoreSegment{},
			name: "'unknown' on 'unknown' using 'unknown'",
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
				Library: "grpc",
			},
			name: "grpc unknown unknown",
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
			name: "unknown send",
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

func TestSegmentsAddAttribute(t *testing.T) {
	type segment interface {
		AddAttribute(string, interface{})
		End()
	}

	testcases := []struct {
		start      func(*Transaction) segment
		extraAttrs map[string]interface{}
	}{
		{
			start: func(txn *Transaction) segment {
				return txn.StartSegment("basic")
			},
		},
		{
			start: func(txn *Transaction) segment {
				return &DatastoreSegment{
					StartTime: txn.StartSegmentNow(),
				}
			},
			extraAttrs: map[string]interface{}{
				"db.collection": "unknown",
				"db.name":       "unknown",
				"db.operation":  "unknown",
				"db.statement":  "'unknown' on 'unknown' using 'unknown'",
				"db.system":     "unknown",
				"net.peer.name": "unknown",
			},
		},
		{
			start: func(txn *Transaction) segment {
				return &ExternalSegment{
					StartTime: txn.StartSegmentNow(),
				}
			},
			extraAttrs: map[string]interface{}{
				"http.component":   "http",
				"http.url":         "unknown",
				"http.status_code": int64(0),
			},
		},
		{
			start: func(txn *Transaction) segment {
				return &MessageProducerSegment{
					StartTime: txn.StartSegmentNow(),
				}
			},
			extraAttrs: map[string]interface{}{
				"messaging.system":      "unknown",
				"messaging.destination": "unknown",
			},
		},
	}

	for _, test := range testcases {
		app := newTestApp(t)
		txn := app.StartTransaction("transaction")
		seg := test.start(txn)
		seg.AddAttribute("attr-string", "this is a string")
		seg.AddAttribute("attr-float-32", float32(1.5))
		seg.AddAttribute("attr-float-64", float64(2.5))
		seg.AddAttribute("attr-int", int(2))
		seg.AddAttribute("attr-int-8", int8(3))
		seg.AddAttribute("attr-int-16", int16(4))
		seg.AddAttribute("attr-int-32", int32(5))
		seg.AddAttribute("attr-int-64", int64(6))
		seg.AddAttribute("attr-uint", uint(7))
		seg.AddAttribute("attr-uint-8", uint8(8))
		seg.AddAttribute("attr-uint-16", uint16(9))
		seg.AddAttribute("attr-uint-32", uint32(10))
		seg.AddAttribute("attr-uint-64", uint64(11))
		seg.AddAttribute("attr-uint-ptr", uintptr(12))
		seg.AddAttribute("attr-bool", true)
		seg.End()
		txn.End()

		attrs := map[string]interface{}{
			"attr-string":   "this is a string",
			"attr-float-32": float32(1.5),
			"attr-float-64": float64(2.5),
			"attr-int":      int64(2),
			"attr-int-8":    int64(3),
			"attr-int-16":   int64(4),
			"attr-int-32":   int32(5),
			"attr-int-64":   int64(6),
			"attr-uint":     uint64(7),
			"attr-uint-8":   uint64(8),
			"attr-uint-16":  uint64(9),
			"attr-uint-32":  uint32(10),
			"attr-uint-64":  uint64(11),
			"attr-uint-ptr": uint64(12),
			"attr-bool":     true,
		}
		for k, v := range test.extraAttrs {
			attrs[k] = v
		}
		app.ExpectSpanEvents(t, []internal.WantSpan{
			{
				Attributes: attrs,
			},
			{Name: "transaction"},
		})
	}
}

func TestNilSegmentAddAttribute(t *testing.T) {
	// AddAttribute APIs don't panic when used on nil seg
	var basic *Segment
	basic.AddAttribute("color", "purple")
	basic = new(Segment)
	basic.AddAttribute("color", "purple")

	var external *ExternalSegment
	external.AddAttribute("color", "purple")
	external = new(ExternalSegment)
	external.AddAttribute("color", "purple")

	var datastore *DatastoreSegment
	datastore.AddAttribute("color", "purple")
	datastore = new(DatastoreSegment)
	datastore.AddAttribute("color", "purple")

	var message *MessageProducerSegment
	message.AddAttribute("color", "purple")
	message = new(MessageProducerSegment)
	message.AddAttribute("color", "purple")
}

func TestDatastoreSegmentAttributes(t *testing.T) {
	testcases := []struct {
		name  string
		seg   *DatastoreSegment
		attrs map[string]interface{}
	}{
		{
			name: "empty segment",
			seg:  &DatastoreSegment{},
			attrs: map[string]interface{}{
				"db.collection": "unknown",
				"db.name":       "unknown",
				"db.operation":  "unknown",
				"db.statement":  "'unknown' on 'unknown' using 'unknown'",
				"db.system":     "unknown",
				"net.peer.name": "unknown",
			},
		},
		{
			name: "complete segment",
			seg: &DatastoreSegment{
				Product:            DatastorePostgres,
				Collection:         "mycollection",
				Operation:          "myoperation",
				ParameterizedQuery: "myparameterizedquery",
				QueryParameters: map[string]interface{}{
					"query": "params",
				},
				Host:         "newrelic.com",
				PortPathOrID: "1234",
				DatabaseName: "mydbname",
			},
			attrs: map[string]interface{}{
				"db.collection": "mycollection",
				"db.name":       "mydbname",
				"db.operation":  "myoperation",
				"db.statement":  "myparameterizedquery",
				"db.system":     "postgresql",
				"net.peer.name": "newrelic.com",
				"net.peer.port": 1234,
			},
		},
		{
			name: "cassandra product",
			seg: &DatastoreSegment{
				Product:      DatastoreCassandra,
				DatabaseName: "mydbname",
			},
			attrs: map[string]interface{}{
				"db.collection":         "unknown",
				"db.cassandra.keyspace": "mydbname",
				"db.operation":          "unknown",
				"db.statement":          "'unknown' on 'unknown' using 'cassandra'",
				"db.system":             "cassandra",
				"net.peer.name":         "unknown",
			},
		},
		{
			name: "redis product",
			seg: &DatastoreSegment{
				Product:      DatastoreRedis,
				DatabaseName: "mydbname",
			},
			attrs: map[string]interface{}{
				"db.collection":           "unknown",
				"db.operation":            "unknown",
				"db.redis.database_index": "mydbname",
				"db.statement":            "'unknown' on 'unknown' using 'redis'",
				"db.system":               "redis",
				"net.peer.name":           "unknown",
			},
		},
		{
			name: "mongodb product",
			seg: &DatastoreSegment{
				Product:      DatastoreMongoDB,
				DatabaseName: "mydbname",
			},
			attrs: map[string]interface{}{
				"db.collection":         "unknown",
				"db.mongodb.collection": "mydbname",
				"db.operation":          "unknown",
				"db.statement":          "'unknown' on 'unknown' using 'mongodb'",
				"db.system":             "mongodb",
				"net.peer.name":         "unknown",
			},
		},
		{
			name: "ipv4 host",
			seg: &DatastoreSegment{
				Host: "1.2.3.4",
			},
			attrs: map[string]interface{}{
				"db.collection": "unknown",
				"db.name":       "unknown",
				"db.operation":  "unknown",
				"db.statement":  "'unknown' on 'unknown' using 'unknown'",
				"db.system":     "unknown",
				"net.peer.ip":   "1.2.3.4",
			},
		},
		{
			name: "ipv6 host",
			seg: &DatastoreSegment{
				Host: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			},
			attrs: map[string]interface{}{
				"db.collection": "unknown",
				"db.name":       "unknown",
				"db.operation":  "unknown",
				"db.statement":  "'unknown' on 'unknown' using 'unknown'",
				"db.system":     "unknown",
				"net.peer.ip":   "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			},
		},
		{
			name: "host is localhost",
			seg: &DatastoreSegment{
				Host: "localhost",
			},
			attrs: map[string]interface{}{
				"db.collection": "unknown",
				"db.name":       "unknown",
				"db.operation":  "unknown",
				"db.statement":  "'unknown' on 'unknown' using 'unknown'",
				"db.system":     "unknown",
				"net.peer.name": thisHost,
			},
		},
		{
			name: "host is 127.0.0.1",
			seg: &DatastoreSegment{
				Host: "127.0.0.1",
			},
			attrs: map[string]interface{}{
				"db.collection": "unknown",
				"db.name":       "unknown",
				"db.operation":  "unknown",
				"db.statement":  "'unknown' on 'unknown' using 'unknown'",
				"db.system":     "unknown",
				"net.peer.name": thisHost,
			},
		},
		{
			name: "port is a path",
			seg: &DatastoreSegment{
				PortPathOrID: "/this/is/a/path/to/a/socket.sock",
			},
			attrs: map[string]interface{}{
				"db.collection": "unknown",
				"db.name":       "unknown",
				"db.operation":  "unknown",
				"db.statement":  "'unknown' on 'unknown' using 'unknown'",
				"db.system":     "unknown",
				"net.peer.name": "unknown",
			},
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			attrs := make(map[string]interface{})
			test.seg.addRequiredAttributes(func(attributes ...label.KeyValue) {
				k := string(attributes[0].Key)
				v := attributes[0].Value
				attrs[k] = v
			})

			if len(attrs) != len(test.attrs) {
				t.Errorf("Incorrect number of attrs created:\n\texpect=%d actual=%d",
					len(test.attrs), len(attrs))
			}
			for expK, expV := range test.attrs {
				actV, ok := attrs[expK]
				if !ok {
					t.Errorf("Attribute '%s' not found", expK)
				} else if actV != expV {
					t.Errorf("Incorrect value for attribute '%s':\n\texpect=%s actual=%s",
						expK, expV, actV)
				}
			}
		})
	}
}

func TestExternalSegmentAttributes(t *testing.T) {
	testcases := []struct {
		name  string
		seg   *ExternalSegment
		attrs map[string]interface{}
	}{
		{
			name: "empty segment",
			seg:  &ExternalSegment{},
			attrs: map[string]interface{}{
				"http.component":   "http",
				"http.url":         "unknown",
				"http.status_code": int64(0),
			},
		},
		{
			name: "method from procedure",
			seg: &ExternalSegment{
				Procedure: "myprocedure",
				Request: &http.Request{
					Method: "GET",
					URL:    &url.URL{},
				},
			},
			attrs: map[string]interface{}{
				"http.component":   "http",
				"http.method":      "myprocedure",
				"http.scheme":      "http",
				"http.status_code": int64(0),
				"http.url":         "unknown",
			},
		},
		{
			name: "method from request",
			seg: &ExternalSegment{
				Request: &http.Request{
					Method: "GET",
					URL:    &url.URL{},
				},
			},
			attrs: map[string]interface{}{
				"http.component":   "http",
				"http.method":      "GET",
				"http.url":         "unknown",
				"http.status_code": int64(0),
				"http.scheme":      "http",
			},
		},
		{
			name: "method from response",
			seg: &ExternalSegment{
				Request: &http.Request{
					Method: "GET",
					URL:    &url.URL{},
				},
				Response: &http.Response{
					Request: &http.Request{
						Method: "PUT",
						URL:    &url.URL{},
					},
				},
			},
			attrs: map[string]interface{}{
				"http.component":   "http",
				"http.method":      "PUT",
				"http.status_code": int64(0),
				"http.url":         "unknown",
				"http.scheme":      "http",
			},
		},
		{
			name: "url from URL field",
			seg: &ExternalSegment{
				URL: "http://example.com",
				Request: &http.Request{
					URL: &url.URL{
						Scheme:   "http",
						Host:     "newrelic.com",
						Path:     "/hello/world",
						RawQuery: "hello=world",
					},
				},
			},
			attrs: map[string]interface{}{
				"http.component":   "http",
				"http.method":      "GET",
				"http.url":         "http://example.com",
				"http.status_code": int64(0),
				"http.scheme":      "http",
			},
		},
		{
			name: "empty request url",
			seg: &ExternalSegment{
				Request: &http.Request{
					URL: &url.URL{},
				},
			},
			attrs: map[string]interface{}{
				"http.component":   "http",
				"http.method":      "GET",
				"http.url":         "unknown",
				"http.status_code": int64(0),
				"http.scheme":      "http",
			},
		},
		{
			name: "url from request",
			seg: &ExternalSegment{
				Request: &http.Request{
					URL: &url.URL{
						Scheme:   "http",
						Host:     "newrelic.com",
						Path:     "/hello/world",
						RawQuery: "hello=world",
					},
				},
			},
			attrs: map[string]interface{}{
				"http.component":   "http",
				"http.method":      "GET",
				"http.url":         "http://newrelic.com/hello/world",
				"http.status_code": int64(0),
				"http.scheme":      "http",
			},
		},
		{
			name: "empty response url",
			seg: &ExternalSegment{
				Request: &http.Request{
					URL: &url.URL{
						Scheme:   "http",
						Host:     "newrelic.com",
						Path:     "/hello/world",
						RawQuery: "hello=world",
					},
				},
				Response: &http.Response{
					Request: &http.Request{
						URL: &url.URL{},
					},
				},
			},
			attrs: map[string]interface{}{
				"http.component":   "http",
				"http.method":      "GET",
				"http.status_code": int64(0),
				"http.url":         "unknown",
				"http.scheme":      "http",
			},
		},
		{
			name: "url from response",
			seg: &ExternalSegment{
				Request: &http.Request{
					URL: &url.URL{
						Scheme:   "http",
						Host:     "example.com",
						Path:     "/hello/world",
						RawQuery: "hello=world",
					},
				},
				Response: &http.Response{
					Request: &http.Request{
						URL: &url.URL{
							Scheme:   "http",
							Host:     "newrelic.com",
							Path:     "/goodbye/world",
							RawQuery: "goodbye=world",
						},
					},
				},
			},
			attrs: map[string]interface{}{
				"http.component":   "http",
				"http.method":      "GET",
				"http.status_code": int64(0),
				"http.url":         "http://newrelic.com/goodbye/world",
				"http.scheme":      "http",
			},
		},
		{
			name: "honors component",
			seg: &ExternalSegment{
				Library: "grpc",
			},
			attrs: map[string]interface{}{
				"http.component":   "grpc",
				"http.url":         "unknown",
				"http.status_code": int64(0),
			},
		},
		{
			name: "status code from API call",
			seg: &ExternalSegment{
				statusCode: func() *int {
					n := 42
					return &n
				}(),
				Response: &http.Response{
					StatusCode: 18,
				},
			},
			attrs: map[string]interface{}{
				"http.component":   "http",
				"http.status_code": int64(42),
				"http.url":         "unknown",
			},
		},
		{
			name: "status code from response",
			seg: &ExternalSegment{
				Response: &http.Response{
					StatusCode: 42,
				},
			},
			attrs: map[string]interface{}{
				"http.component":   "http",
				"http.status_code": int64(42),
				"http.url":         "unknown",
			},
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			attrs := make(map[string]interface{})
			test.seg.addRequiredAttributes(func(keyValues ...label.KeyValue) {
				for _, keyValue := range keyValues {
					attrs[string(keyValue.Key)] = keyValue.Value.AsInterface()
				}
			})

			if len(attrs) != len(test.attrs) {
				t.Errorf("Incorrect number of attrs created:\n\texpect=%d actual=%d",
					len(test.attrs), len(attrs))
			}
			for expK, expV := range test.attrs {
				actV, ok := attrs[expK]
				if !ok {
					t.Errorf("Attribute '%s' not found", expK)
				} else if actV != expV {
					t.Errorf("Incorrect value for attribute '%s':\n\texpect=%s actual=%s",
						expK, expV, actV)
				}
			}
		})
	}
}

func TestExternalSegmentSpanStatus(t *testing.T) {
	intptr := func(i int) *int { return &i }

	testcases := []struct {
		name string
		seg  *ExternalSegment
		code codes.Code
		str  string
	}{
		{
			name: "empty segment",
			seg:  &ExternalSegment{},
			code: codes.Code(0),
			str:  "OK",
		},
		{
			name: "grpc range code",
			seg: &ExternalSegment{
				statusCode: intptr(8),
			},
			code: codes.Code(8),
			str:  "ResourceExhausted",
		},
		{
			name: "unknown range code",
			seg: &ExternalSegment{
				statusCode: intptr(42),
			},
			code: codes.Code(2),
			str:  "Invalid HTTP status code 42",
		},
		{
			name: "http range code 418",
			seg: &ExternalSegment{
				statusCode: intptr(418),
			},
			code: codes.Code(3),
			str:  "HTTP status code: 418",
		},
		{
			name: "http range code 200",
			seg: &ExternalSegment{
				statusCode: intptr(200),
			},
			code: codes.Code(0),
			str:  "HTTP status code: 200",
		},
		{
			name: "response status code 418",
			seg: &ExternalSegment{
				Response: &http.Response{
					StatusCode: 418,
				},
			},
			code: codes.Code(3),
			str:  "HTTP status code: 418",
		},
		{
			name: "response status code 200",
			seg: &ExternalSegment{
				Response: &http.Response{
					StatusCode: 200,
				},
			},
			code: codes.Code(0),
			str:  "HTTP status code: 200",
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			var actCode codes.Code
			var actStr string
			test.seg.setSpanStatus(func(c codes.Code, s string) {
				actCode = c
				actStr = s
			})
			if actCode != test.code {
				t.Errorf("Incorrect code recorded:\n\texpect=%d actual=%d",
					test.code, actCode)
			}
			if actStr != test.str {
				t.Errorf("Incorrect string recorded:\n\texpect=%s actual=%s",
					test.str, actStr)
			}
		})
	}
}

func TestMessageProducerSegmentAttributes(t *testing.T) {
	testcases := []struct {
		name  string
		seg   *MessageProducerSegment
		attrs map[string]interface{}
	}{
		{
			name: "empty segment",
			seg:  &MessageProducerSegment{},
			attrs: map[string]interface{}{
				"messaging.system":      "unknown",
				"messaging.destination": "unknown",
			},
		},
		{
			name: "complete segment",
			seg: &MessageProducerSegment{
				Library:              "kafka",
				DestinationType:      MessageQueue,
				DestinationName:      "mydestination",
				DestinationTemporary: true,
			},
			attrs: map[string]interface{}{
				"messaging.system":           "kafka",
				"messaging.destination":      "mydestination",
				"messaging.destination_kind": "queue",
				"messaging.temp_destination": true,
			},
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			attrs := make(map[string]interface{})
			test.seg.addRequiredAttributes(func(keyValues ...label.KeyValue) {
				k := string(keyValues[0].Key)
				v := keyValues[0].Value
				attrs[k] = v
			})

			if len(attrs) != len(test.attrs) {
				t.Errorf("Incorrect number of attrs created:\n\texpect=%d actual=%d",
					len(test.attrs), len(attrs))
			}
			for expK, expV := range test.attrs {
				actV, ok := attrs[expK]
				if !ok {
					t.Errorf("Attribute '%s' not found", expK)
				} else if actV != expV {
					t.Errorf("Incorrect value for attribute '%s':\n\texpect=%s actual=%s",
						expK, expV, actV)
				}
			}
		})
	}
}
