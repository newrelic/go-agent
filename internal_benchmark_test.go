// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"net/http"
	"testing"
)

var (
	sampleLicense = "0123456789012345678901234567890123456789"
)

// BenchmarkMuxWithoutNewRelic acts as a control against the other mux
// benchmarks.
func BenchmarkMuxWithoutNewRelic(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc(helloPath, handler)

	w := newCompatibleResponseRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, helloRequest)
	}
}

// BenchmarkMuxWithNewRelic shows the approximate overhead of instrumenting a
// request.  The numbers here are approximate since this is a test app: rather
// than putting the transaction into a channel to be processed by another
// goroutine, the transaction is merged directly into a harvest.
func BenchmarkMuxWithNewRelic(b *testing.B) {
	app := testApp(nil, nil, b)
	mux := http.NewServeMux()
	mux.HandleFunc(WrapHandleFunc(app, helloPath, handler))

	w := newCompatibleResponseRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, helloRequest)
	}
}

// BenchmarkMuxWithNewRelic shows the overhead of instrumenting a request when
// the agent is disabled.
func BenchmarkMuxDisabledMode(b *testing.B) {
	cfg := NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	app, err := newApp(cfg)
	if nil != err {
		b.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc(WrapHandleFunc(app, helloPath, handler))

	w := newCompatibleResponseRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, helloRequest)
	}
}

// BenchmarkTraceSegmentWithDefer shows the overhead of instrumenting a segment
// using defer.  This and BenchmarkTraceSegmentNoDefer are extremely important:
// Timing functions and blocks of code should have minimal cost.
func BenchmarkTraceSegmentWithDefer(b *testing.B) {
	cfg := NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	app, err := newApp(cfg)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn", nil, nil)
	fn := func() {
		defer StartSegment(txn, "alpha").End()
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fn()
	}
}

func BenchmarkTraceSegmentNoDefer(b *testing.B) {
	cfg := NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	app, err := newApp(cfg)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn", nil, nil)
	fn := func() {
		s := StartSegment(txn, "alpha")
		s.End()
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fn()
	}
}

func BenchmarkTraceSegmentZeroSegmentThreshold(b *testing.B) {
	cfg := NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	cfg.TransactionTracer.SegmentThreshold = 0
	app, err := newApp(cfg)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn", nil, nil)
	fn := func() {
		s := StartSegment(txn, "alpha")
		s.End()
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fn()
	}
}

func BenchmarkDatastoreSegment(b *testing.B) {
	cfg := NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	app, err := newApp(cfg)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn", nil, nil)
	fn := func(txn Transaction) {
		ds := DatastoreSegment{
			StartTime:  txn.StartSegmentNow(),
			Product:    DatastoreMySQL,
			Collection: "my_table",
			Operation:  "Select",
		}
		defer ds.End()
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fn(txn)
	}
}

func BenchmarkExternalSegment(b *testing.B) {
	cfg := NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	app, err := newApp(cfg)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn", nil, nil)
	fn := func(txn Transaction) {
		es := &ExternalSegment{
			StartTime: txn.StartSegmentNow(),
			URL:       "http://example.com/",
		}
		defer es.End()
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fn(txn)
	}
}

func BenchmarkTxnWithSegment(b *testing.B) {
	app := testApp(nil, nil, b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		txn := app.StartTransaction("my txn", nil, nil)
		StartSegment(txn, "myFunction").End()
		txn.End()
	}
}

func BenchmarkTxnWithDatastore(b *testing.B) {
	app := testApp(nil, nil, b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		txn := app.StartTransaction("my txn", nil, nil)
		ds := &DatastoreSegment{
			StartTime:  txn.StartSegmentNow(),
			Product:    DatastoreMySQL,
			Collection: "my_table",
			Operation:  "Select",
		}
		ds.End()
		txn.End()
	}
}

func BenchmarkTxnWithExternal(b *testing.B) {
	app := testApp(nil, nil, b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		txn := app.StartTransaction("my txn", nil, nil)
		es := &ExternalSegment{
			StartTime: txn.StartSegmentNow(),
			URL:       "http://example.com",
		}
		es.End()
		txn.End()
	}
}
