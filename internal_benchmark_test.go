package newrelic

import (
	"net/http"
	"testing"
)

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

func BenchmarkDatastoreSegment(b *testing.B) {
	cfg := NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	app, err := newApp(cfg)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn", nil, nil)
	fn := func(txn Transaction) {
		defer DatastoreSegment{
			StartTime:  txn.StartSegmentNow(),
			Product:    DatastoreMySQL,
			Collection: "my_table",
			Operation:  "Select",
		}.End()
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
		defer ExternalSegment{
			StartTime: txn.StartSegmentNow(),
			URL:       "http://example.com/",
		}.End()
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
		DatastoreSegment{
			StartTime:  txn.StartSegmentNow(),
			Product:    DatastoreMySQL,
			Collection: "my_table",
			Operation:  "Select",
		}.End()
		txn.End()
	}
}

func BenchmarkTxnWithExternal(b *testing.B) {
	app := testApp(nil, nil, b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		txn := app.StartTransaction("my txn", nil, nil)
		ExternalSegment{
			StartTime: txn.StartSegmentNow(),
			URL:       "http://example.com",
		}.End()
		txn.End()
	}
}
