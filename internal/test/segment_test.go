package test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/api"
	"github.com/newrelic/go-agent/api/datastore"
	"github.com/newrelic/go-agent/internal"
)

func TestTraceSegment(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	func() {
		defer txn.EndSegment(txn.StartSegment(), "segment")
	}()
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Custom/segment", "", false, nil},
		{"Custom/segment", "WebTransaction/Go/myName", false, nil},
	})
}

func TestTraceSegmentEndedBeforeStartSegment(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	txn.End()
	func() {
		defer txn.EndSegment(txn.StartSegment(), "segment")
	}()
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
	})
}

func TestTraceSegmentEndedBeforeEndSegment(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	token := txn.StartSegment()
	txn.End()
	txn.EndSegment(token, "segment")

	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
	})
}

func TestTraceSegmentPanic(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	func() {
		defer func() {
			recover()
		}()

		func() {
			defer txn.EndSegment(txn.StartSegment(), "f1")

			func() {
				t := txn.StartSegment()

				func() {
					defer txn.EndSegment(txn.StartSegment(), "f3")

					func() {
						txn.StartSegment()

						panic(nil)
					}()
				}()

				txn.EndSegment(t, "f2")
			}()
		}()
	}()

	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Custom/f1", "", false, nil},
		{"Custom/f1", "WebTransaction/Go/myName", false, nil},
		{"Custom/f3", "", false, nil},
		{"Custom/f3", "WebTransaction/Go/myName", false, nil},
	})
}

func TestTraceSegmentInvalidToken(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	token := txn.StartSegment()
	token++
	txn.EndSegment(token, "segment")
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
	})
}

func TestTraceSegmentDefaultToken(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	var token api.Token
	txn.EndSegment(token, "segment")
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
	})
}

func TestTraceDatastore(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	func() {
		defer txn.EndDatastore(txn.StartSegment(), datastore.Segment{
			Product:    datastore.MySQL,
			Collection: "my_table",
			Operation:  "SELECT",
		})
	}()
	txn.NoticeError(myError{})
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allWeb", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/WebTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Datastore/all", "", true, nil},
		{"Datastore/allWeb", "", true, nil},
		{"Datastore/MySQL/all", "", true, nil},
		{"Datastore/MySQL/allWeb", "", true, nil},
		{"Datastore/operation/MySQL/SELECT", "", false, nil},
		{"Datastore/statement/MySQL/my_table/SELECT", "", false, nil},
		{"Datastore/statement/MySQL/my_table/SELECT", "WebTransaction/Go/myName", false, nil},
	})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:            "WebTransaction/Go/myName",
		Msg:                "my msg",
		Klass:              "test.myError",
		DatastoreCallCount: 1,
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:               "WebTransaction/Go/myName",
		Zone:               "F",
		DatastoreCallCount: 1,
	}})
}

func TestTraceDatastoreBackground(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	func() {
		defer txn.EndDatastore(txn.StartSegment(), datastore.Segment{
			Product:    datastore.MySQL,
			Collection: "my_table",
			Operation:  "SELECT",
		})
	}()
	txn.NoticeError(myError{})
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/OtherTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Datastore/all", "", true, nil},
		{"Datastore/allOther", "", true, nil},
		{"Datastore/MySQL/all", "", true, nil},
		{"Datastore/MySQL/allOther", "", true, nil},
		{"Datastore/operation/MySQL/SELECT", "", false, nil},
		{"Datastore/statement/MySQL/my_table/SELECT", "", false, nil},
		{"Datastore/statement/MySQL/my_table/SELECT", "OtherTransaction/Go/myName", false, nil},
	})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:            "OtherTransaction/Go/myName",
		Msg:                "my msg",
		Klass:              "test.myError",
		DatastoreCallCount: 1,
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:               "OtherTransaction/Go/myName",
		Zone:               "",
		DatastoreCallCount: 1,
	}})
}

func TestTraceDatastoreMissingProductOperationCollection(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	func() {
		defer txn.EndDatastore(txn.StartSegment(), datastore.Segment{})
	}()
	txn.NoticeError(myError{})
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allWeb", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/WebTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Datastore/all", "", true, nil},
		{"Datastore/allWeb", "", true, nil},
		{"Datastore/Unknown/all", "", true, nil},
		{"Datastore/Unknown/allWeb", "", true, nil},
		{"Datastore/operation/Unknown/other", "", false, nil},
		{"Datastore/operation/Unknown/other", "WebTransaction/Go/myName", false, nil},
	})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:            "WebTransaction/Go/myName",
		Msg:                "my msg",
		Klass:              "test.myError",
		DatastoreCallCount: 1,
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:               "WebTransaction/Go/myName",
		Zone:               "F",
		DatastoreCallCount: 1,
	}})
}

func TestTraceDatastoreInvalidToken(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	token := txn.StartSegment()
	token++
	txn.EndDatastore(token, datastore.Segment{
		Product:    datastore.MySQL,
		Collection: "my_table",
		Operation:  "SELECT",
	})
	txn.NoticeError(myError{})
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allWeb", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/WebTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "WebTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "test.myError",
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name: "WebTransaction/Go/myName",
		Zone: "F",
	}})
}

func TestTraceDatastoreTxnEnded(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	txn.NoticeError(myError{})
	token := txn.StartSegment()
	txn.End()
	txn.EndDatastore(token, datastore.Segment{
		Product:    datastore.MySQL,
		Collection: "my_table",
		Operation:  "SELECT",
	})

	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allWeb", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/WebTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "WebTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "test.myError",
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name: "WebTransaction/Go/myName",
		Zone: "F",
	}})
}

func TestTraceExternal(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	func() {
		defer txn.EndExternal(txn.StartSegment(), "http://example.com/")
	}()
	txn.NoticeError(myError{})
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allWeb", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/WebTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"External/all", "", true, nil},
		{"External/allWeb", "", true, nil},
		{"External/example.com/all", "", false, nil},
		{"External/example.com/all", "WebTransaction/Go/myName", false, nil},
	})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:           "WebTransaction/Go/myName",
		Msg:               "my msg",
		Klass:             "test.myError",
		ExternalCallCount: 1,
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:              "WebTransaction/Go/myName",
		Zone:              "F",
		ExternalCallCount: 1,
	}})
}

func TestTraceExternalBackground(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	func() {
		defer txn.EndExternal(txn.StartSegment(), "http://example.com/")
	}()
	txn.NoticeError(myError{})
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/OtherTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"External/all", "", true, nil},
		{"External/allOther", "", true, nil},
		{"External/example.com/all", "", false, nil},
		{"External/example.com/all", "OtherTransaction/Go/myName", false, nil},
	})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:           "OtherTransaction/Go/myName",
		Msg:               "my msg",
		Klass:             "test.myError",
		ExternalCallCount: 1,
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:              "OtherTransaction/Go/myName",
		Zone:              "",
		ExternalCallCount: 1,
	}})
}

func TestTraceExternalMissingURL(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	func() {
		defer txn.EndExternal(txn.StartSegment(), "")
	}()
	txn.NoticeError(myError{})
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allWeb", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/WebTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"External/all", "", true, nil},
		{"External/allWeb", "", true, nil},
		{"External/unknown/all", "", false, nil},
		{"External/unknown/all", "WebTransaction/Go/myName", false, nil},
	})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:           "WebTransaction/Go/myName",
		Msg:               "my msg",
		Klass:             "test.myError",
		ExternalCallCount: 1,
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:              "WebTransaction/Go/myName",
		Zone:              "F",
		ExternalCallCount: 1,
	}})
}

func TestTraceExternalInvalidToken(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	txn.NoticeError(myError{})
	token := txn.StartSegment()
	token++
	txn.EndExternal(token, "http://example.com/")
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allWeb", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/WebTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "WebTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "test.myError",
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name: "WebTransaction/Go/myName",
		Zone: "F",
	}})
}

func TestTraceExternalTxnEnded(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	txn.NoticeError(myError{})
	token := txn.StartSegment()
	txn.End()
	txn.EndExternal(token, "http://example.com/")

	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allWeb", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/WebTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "WebTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "test.myError",
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name: "WebTransaction/Go/myName",
		Zone: "F",
	}})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRoundTripper(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	url := "http://example.com/"
	client := &http.Client{}
	inner := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		// TODO test that request headers have been set here.
		if r.URL.String() != url {
			t.Error(r.URL.String())
		}
		return nil, errors.New("hello")
	})
	client.Transport = newrelic.NewRoundTripper(txn, inner)
	resp, err := client.Get(url)
	if resp != nil || err == nil {
		t.Error(resp, err.Error())
	}
	txn.NoticeError(myError{})
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/OtherTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"External/all", "", true, nil},
		{"External/allOther", "", true, nil},
		{"External/example.com/all", "", false, nil},
		{"External/example.com/all", "OtherTransaction/Go/myName", false, nil},
	})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:           "OtherTransaction/Go/myName",
		Msg:               "my msg",
		Klass:             "test.myError",
		ExternalCallCount: 1,
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:              "OtherTransaction/Go/myName",
		Zone:              "",
		ExternalCallCount: 1,
	}})
}
