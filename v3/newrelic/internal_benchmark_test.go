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
	mux.HandleFunc(WrapHandleFunc(app.Application, helloPath, handler))

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
	app, err := NewApplication(
		ConfigAppName("my app"),
		ConfigLicense(sampleLicense),
		ConfigEnabled(false),
	)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn")
	fn := func() {
		defer txn.StartSegment("alpha").End()
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fn()
	}
}

func BenchmarkTraceSegmentNoDefer(b *testing.B) {
	app, err := NewApplication(
		ConfigAppName("my app"),
		ConfigLicense(sampleLicense),
		ConfigEnabled(false),
	)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn")
	fn := func() {
		s := txn.StartSegment("alpha")
		s.End()
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fn()
	}
}

func BenchmarkTraceSegmentZeroSegmentThreshold(b *testing.B) {
	app, err := NewApplication(
		ConfigAppName("my app"),
		ConfigLicense(sampleLicense),
		ConfigEnabled(false),
		func(cfg *Config) {
			cfg.TransactionTracer.Segments.Threshold = 0
		},
	)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn")
	fn := func() {
		s := txn.StartSegment("alpha")
		s.End()
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fn()
	}
}

func BenchmarkDatastoreSegment(b *testing.B) {
	app, err := NewApplication(
		ConfigAppName("my app"),
		ConfigLicense(sampleLicense),
		ConfigEnabled(false),
	)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn")
	fn := func(txn *Transaction) {
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
	app, err := NewApplication(
		ConfigAppName("my app"),
		ConfigLicense(sampleLicense),
		ConfigEnabled(false),
	)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn")
	fn := func(txn *Transaction) {
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
		txn := app.StartTransaction("my txn")
		txn.StartSegment("myFunction").End()
		txn.End()
	}
}

func BenchmarkTxnWithDatastore(b *testing.B) {
	app := testApp(nil, nil, b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		txn := app.StartTransaction("my txn")
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
		txn := app.StartTransaction("my txn")
		es := &ExternalSegment{
			StartTime: txn.StartSegmentNow(),
			URL:       "http://example.com",
		}
		es.End()
		txn.End()
	}
}
