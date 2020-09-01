// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/v4/internal"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/api/trace/testtrace"
	"google.golang.org/grpc/codes"
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

func TestTransactionAddAttribute(t *testing.T) {
	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	txn.AddAttribute("attr-string", "this is a string")
	txn.AddAttribute("attr-float-32", float32(1.5))
	txn.AddAttribute("attr-float-64", float64(2.5))
	txn.AddAttribute("attr-int", int(2))
	txn.AddAttribute("attr-int-8", int8(3))
	txn.AddAttribute("attr-int-16", int16(4))
	txn.AddAttribute("attr-int-32", int32(5))
	txn.AddAttribute("attr-int-64", int64(6))
	txn.AddAttribute("attr-uint", uint(7))
	txn.AddAttribute("attr-uint-8", uint8(8))
	txn.AddAttribute("attr-uint-16", uint16(9))
	txn.AddAttribute("attr-uint-32", uint32(10))
	txn.AddAttribute("attr-uint-64", uint64(11))
	txn.AddAttribute("attr-uint-ptr", uintptr(12))
	txn.AddAttribute("attr-bool", true)
	txn.End()

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name: "transaction",
			Attributes: map[string]interface{}{
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
			},
		},
	})
}

func TestNilTransactionAddAttribute(t *testing.T) {
	// AddAttribute APIs don't panic when used on nil txn
	var txn *Transaction
	txn.AddAttribute("color", "purple")
	txn = new(Transaction)
	txn.AddAttribute("color", "purple")
}

func TestAddTxnRequestAttributes(t *testing.T) {
	testcases := []struct {
		name  string
		req   *http.Request
		attrs map[string]interface{}
	}{
		{
			name: "empty request",
			req:  &http.Request{},
			attrs: map[string]interface{}{
				"http.method":   "",
				"http.scheme":   "http",
				"http.target":   "",
				"net.transport": "IP.TCP",
			},
		},
		{
			name: "complete request",
			req: func() *http.Request {
				req, err := http.NewRequest("POST", "http://example.com:80/the/path?the=query",
					bytes.NewBufferString("hello world"))
				if err != nil {
					panic(err)
				}
				req.SetBasicAuth("Harry Potter", "Caput Draconis")
				req.RequestURI = "/what/path"
				req.Header.Add("User-Agent", "curl/7.64.1")
				return req
			}(),
			attrs: map[string]interface{}{
				"enduser.id":                  "Harry Potter",
				"http.flavor":                 "1.1",
				"http.host":                   "example.com:80",
				"http.method":                 "POST",
				"http.request_content_length": int64(11),
				"http.scheme":                 "http",
				"http.target":                 "/what/path",
				"http.user_agent":             "curl/7.64.1",
				"net.host.name":               "example.com",
				"net.host.port":               int64(80),
				"net.transport":               "IP.TCP",
			},
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			attrs := make(map[string]interface{})
			addTxnHTTPRequestAttributes(test.req, func(keyValues ...kv.KeyValue) {
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

func TestAddTxnStatusCodeAttributes(t *testing.T) {
	testcases := []struct {
		name  string
		code  int
		attrs map[string]interface{}
	}{
		{
			name: "non-error code",
			code: 0,
			attrs: map[string]interface{}{
				"http.status_code": int64(0),
			},
		},
		{
			name: "OK code",
			code: 200,
			attrs: map[string]interface{}{
				"http.status_code": int64(200),
				"http.status_text": "OK",
			},
		},
		{
			name: "error level code",
			code: 500,
			attrs: map[string]interface{}{
				"http.status_code": int64(500),
				"http.status_text": "Internal Server Error",
			},
		},
		{
			name: "absurd code",
			code: 999,
			attrs: map[string]interface{}{
				"http.status_code": int64(999),
			},
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			attrs := make(map[string]interface{})
			addTxnStatusCodeAttributes(test.code, func(keyValues ...kv.KeyValue) {
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

func TestSetTxnSpanStatus(t *testing.T) {
	var actCode codes.Code
	var actStr string
	setTxnSpanStatus(500, func(c codes.Code, s string) {
		actCode = c
		actStr = s
	})

	if expCode := codes.Code(13); actCode != expCode {
		t.Errorf("Incorrect code recorded:\n\texpect=%d actual=%d",
			expCode, actCode)
	}
	if expStr := "HTTP status code: 500"; actStr != expStr {
		t.Errorf("Incorrect string recorded:\n\texpect=%s actual=%s",
			expStr, actStr)
	}
}
