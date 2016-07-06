package test

import (
	"testing"

	"github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/api"
	"github.com/newrelic/go-agent/api/datastore"
	"github.com/newrelic/go-agent/internal"
)

func BenchmarkTraceSegmentWithDefer(b *testing.B) {
	cfg := newrelic.NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	app, err := internal.NewAppInternal(cfg)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn", nil, nil)
	fn := func() {
		defer txn.EndSegment(txn.StartSegment(), "alpha")
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fn()
	}
}

func BenchmarkTraceSegmentNoDefer(b *testing.B) {
	cfg := newrelic.NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	app, err := internal.NewAppInternal(cfg)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn", nil, nil)
	fn := func() {
		token := txn.StartSegment()
		txn.EndSegment(token, "alpha")
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fn()
	}
}

func BenchmarkDatastoreSegment(b *testing.B) {
	cfg := newrelic.NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	app, err := internal.NewAppInternal(cfg)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn", nil, nil)
	fn := func(txn api.Transaction) {
		defer txn.EndDatastore(txn.StartSegment(), datastore.Segment{
			Product:    datastore.MySQL,
			Collection: "my_table",
			Operation:  "SELECT",
		})
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fn(txn)
	}
}

func BenchmarkExternalSegment(b *testing.B) {
	cfg := newrelic.NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	app, err := internal.NewAppInternal(cfg)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn", nil, nil)
	fn := func(txn api.Transaction) {
		defer txn.EndExternal(txn.StartSegment(), "http://example.com/")
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
		token := txn.StartSegment()
		txn.EndSegment(token, "myFunction")
		txn.End()
	}
}

func BenchmarkTxnWithDatastore(b *testing.B) {
	app := testApp(nil, nil, b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		txn := app.StartTransaction("my txn", nil, nil)
		token := txn.StartSegment()
		txn.EndDatastore(token, datastore.Segment{
			Product:    datastore.MySQL,
			Collection: "my_table",
			Operation:  "SELECT",
		})
		txn.End()
	}
}

func BenchmarkTxnWithExternal(b *testing.B) {
	app := testApp(nil, nil, b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		txn := app.StartTransaction("my txn", nil, nil)
		token := txn.StartSegment()
		txn.EndExternal(token, "http://example.com")
		txn.End()
	}
}
