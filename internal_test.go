// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal"
)

var (
	singleCount = []float64{1, 0, 0, 0, 0, 0, 0}
	webMetrics  = []internal.WantMetric{
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: nil},
	}
	webErrorMetrics = append([]internal.WantMetric{
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/WebTransaction/Go/hello", Scope: "", Forced: true, Data: singleCount},
	}, webMetrics...)
	backgroundMetrics = []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
	}
	backgroundMetricsUnknownCaller = append([]internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
	}, backgroundMetrics...)
	backgroundErrorMetrics = append([]internal.WantMetric{
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/OtherTransaction/Go/hello", Scope: "", Forced: true, Data: singleCount},
	}, backgroundMetrics...)
)

// compatibleResponseRecorder wraps ResponseRecorder to ensure consistent behavior
// between different versions of Go.
//
// Unfortunately, there was a behavior change in go1.6:
//
// "The net/http/httptest package's ResponseRecorder now initializes a default
// Content-Type header using the same content-sniffing algorithm as in
// http.Server."
type compatibleResponseRecorder struct {
	*httptest.ResponseRecorder
	wroteHeader bool
}

func newCompatibleResponseRecorder() *compatibleResponseRecorder {
	return &compatibleResponseRecorder{
		ResponseRecorder: httptest.NewRecorder(),
	}
}

func (rw *compatibleResponseRecorder) Header() http.Header {
	return rw.ResponseRecorder.Header()
}

func (rw *compatibleResponseRecorder) Write(buf []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(200)
		rw.wroteHeader = true
	}
	return rw.ResponseRecorder.Write(buf)
}

func (rw *compatibleResponseRecorder) WriteHeader(code int) {
	rw.wroteHeader = true
	rw.ResponseRecorder.WriteHeader(code)
}

var (
	validParams = map[string]interface{}{"zip": 1, "zap": 2}
)

var (
	helloResponse    = []byte("hello")
	helloPath        = "/hello"
	helloQueryParams = "?secret=hideme"
	helloRequest     = func() *http.Request {
		r, err := http.NewRequest("GET", helloPath+helloQueryParams, nil)
		if nil != err {
			panic(err)
		}

		r.Header.Add(`Accept`, `text/plain`)
		r.Header.Add(`Content-Type`, `text/html; charset=utf-8`)
		r.Header.Add(`Content-Length`, `753`)
		r.Header.Add(`Host`, `my_domain.com`)
		r.Header.Add(`User-Agent`, `Mozilla/5.0`)
		r.Header.Add(`Referer`, `http://en.wikipedia.org/zip?secret=password`)

		return r
	}()
	helloRequestAttributes = map[string]interface{}{
		"request.uri":                   "/hello",
		"request.headers.host":          "my_domain.com",
		"request.headers.referer":       "http://en.wikipedia.org/zip",
		"request.headers.contentLength": 753,
		"request.method":                "GET",
		"request.headers.accept":        "text/plain",
		"request.headers.User-Agent":    "Mozilla/5.0",
		"request.headers.contentType":   "text/html; charset=utf-8",
	}
)

func TestNewApplicationNil(t *testing.T) {
	cfg := NewConfig("appname", "wrong length")
	cfg.Enabled = false
	app, err := NewApplication(cfg)
	if nil == err {
		t.Error("error expected when license key is short")
	}
	if nil != app {
		t.Error("app expected to be nil when error is returned")
	}
}

func handler(w http.ResponseWriter, req *http.Request) {
	w.Write(helloResponse)
}

const (
	testLicenseKey = "0123456789012345678901234567890123456789"
)

type expectApp interface {
	internal.Expect
	Application
}

func testApp(replyfn func(*internal.ConnectReply), cfgfn func(*Config), t testing.TB) expectApp {
	cfg := NewConfig("my app", testLicenseKey)

	if nil != cfgfn {
		cfgfn(&cfg)
	}

	// Prevent spawning app goroutines in tests.
	if !cfg.ServerlessMode.Enabled {
		cfg.Enabled = false
	}

	app, err := newApp(cfg)
	if nil != err {
		t.Fatal(err)
	}

	internal.HarvestTesting(app, replyfn)

	return app.(expectApp)
}

func TestRecordCustomEventSuccess(t *testing.T) {
	app := testApp(nil, nil, t)
	err := app.RecordCustomEvent("myType", validParams)
	if nil != err {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"type":      "myType",
			"timestamp": internal.MatchAnything,
		},
		UserAttributes: validParams,
	}})
}

func TestRecordCustomEventHighSecurityEnabled(t *testing.T) {
	cfgfn := func(cfg *Config) { cfg.HighSecurity = true }
	app := testApp(nil, cfgfn, t)
	err := app.RecordCustomEvent("myType", validParams)
	if err != errHighSecurityEnabled {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{})
}

func TestRecordCustomEventSecurityPolicy(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) { reply.SecurityPolicies.CustomEvents.SetEnabled(false) }
	app := testApp(replyfn, nil, t)
	err := app.RecordCustomEvent("myType", validParams)
	if err != errSecurityPolicy {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{})
}

func TestRecordCustomEventEventsDisabled(t *testing.T) {
	cfgfn := func(cfg *Config) { cfg.CustomInsightsEvents.Enabled = false }
	app := testApp(nil, cfgfn, t)
	err := app.RecordCustomEvent("myType", validParams)
	if err != errCustomEventsDisabled {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{})
}

func TestRecordCustomEventBadInput(t *testing.T) {
	app := testApp(nil, nil, t)
	err := app.RecordCustomEvent("????", validParams)
	if err != internal.ErrEventTypeRegex {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{})
}

func TestRecordCustomEventRemoteDisable(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) { reply.CollectCustomEvents = false }
	app := testApp(replyfn, nil, t)
	err := app.RecordCustomEvent("myType", validParams)
	if err != errCustomEventsRemoteDisabled {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{})
}

func TestRecordCustomMetricSuccess(t *testing.T) {
	app := testApp(nil, nil, t)
	err := app.RecordCustomMetric("myMetric", 123.0)
	if nil != err {
		t.Error(err)
	}
	expectData := []float64{1, 123.0, 123.0, 123.0, 123.0, 123.0 * 123.0}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Custom/myMetric", Scope: "", Forced: false, Data: expectData},
	})
}

func TestRecordCustomMetricNameEmpty(t *testing.T) {
	app := testApp(nil, nil, t)
	err := app.RecordCustomMetric("", 123.0)
	if err != errMetricNameEmpty {
		t.Error(err)
	}
}

func TestRecordCustomMetricNaN(t *testing.T) {
	app := testApp(nil, nil, t)
	err := app.RecordCustomMetric("myMetric", math.NaN())
	if err != errMetricNaN {
		t.Error(err)
	}
}

func TestRecordCustomMetricPositiveInf(t *testing.T) {
	app := testApp(nil, nil, t)
	err := app.RecordCustomMetric("myMetric", math.Inf(0))
	if err != errMetricInf {
		t.Error(err)
	}
}

func TestRecordCustomMetricNegativeInf(t *testing.T) {
	app := testApp(nil, nil, t)
	err := app.RecordCustomMetric("myMetric", math.Inf(-1))
	if err != errMetricInf {
		t.Error(err)
	}
}

type sampleResponseWriter struct {
	code    int
	written int
	header  http.Header
}

func (w *sampleResponseWriter) Header() http.Header       { return w.header }
func (w *sampleResponseWriter) Write([]byte) (int, error) { return w.written, nil }
func (w *sampleResponseWriter) WriteHeader(x int)         { w.code = x }

func TestTxnResponseWriter(t *testing.T) {
	// NOTE: Eventually when the ResponseWriter is instrumented, this test
	// should be expanded to make sure that calling ResponseWriter methods
	// after the transaction has ended is not problematic.
	w := &sampleResponseWriter{
		header: make(http.Header),
	}
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", w, nil)
	w.header.Add("zip", "zap")
	if out := txn.Header(); out.Get("zip") != "zap" {
		t.Error(out.Get("zip"))
	}
	w.written = 123
	if out, _ := txn.Write(nil); out != 123 {
		t.Error(out)
	}
	if txn.WriteHeader(503); w.code != 503 {
		t.Error(w.code)
	}
}

func TestTransactionEventWeb(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	err := txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "S",
		},
	}})
}

func TestTransactionEventBackground(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
	}})
}

func TestTransactionEventLocallyDisabled(t *testing.T) {
	cfgFn := func(cfg *Config) { cfg.TransactionEvents.Enabled = false }
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	err := txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectTxnEvents(t, []internal.WantEvent{})
}

func TestTransactionEventRemotelyDisabled(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) { reply.CollectAnalyticsEvents = false }
	app := testApp(replyfn, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	err := txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectTxnEvents(t, []internal.WantEvent{})
}

func myErrorHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("my response"))
	if txn, ok := w.(Transaction); ok {
		txn.NoticeError(myError{})
	}
}

func TestWrapHandleFunc(t *testing.T) {
	app := testApp(nil, nil, t)
	mux := http.NewServeMux()
	mux.HandleFunc(WrapHandleFunc(app, helloPath, myErrorHandler))
	w := newCompatibleResponseRecorder()
	mux.ServeHTTP(w, helloRequest)

	out := w.Body.String()
	if "my response" != out {
		t.Error(out)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "WebTransaction/Go/hello",
		},
		AgentAttributes: mergeAttributes(helloRequestAttributes, map[string]interface{}{
			"httpResponseCode": "200",
		}),
	}})
	app.ExpectMetrics(t, webErrorMetrics)
}

func TestWrapHandle(t *testing.T) {
	app := testApp(nil, nil, t)
	mux := http.NewServeMux()
	mux.Handle(WrapHandle(app, helloPath, http.HandlerFunc(myErrorHandler)))
	w := newCompatibleResponseRecorder()
	mux.ServeHTTP(w, helloRequest)

	out := w.Body.String()
	if "my response" != out {
		t.Error(out)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "WebTransaction/Go/hello",
		},
		AgentAttributes: mergeAttributes(helloRequestAttributes, map[string]interface{}{
			"httpResponseCode": "200",
		}),
	}})
	app.ExpectMetrics(t, webErrorMetrics)
}

func TestWrapHandleNilApp(t *testing.T) {
	var app Application
	mux := http.NewServeMux()
	mux.Handle(WrapHandle(app, helloPath, http.HandlerFunc(myErrorHandler)))
	w := newCompatibleResponseRecorder()
	mux.ServeHTTP(w, helloRequest)

	out := w.Body.String()
	if "my response" != out {
		t.Error(out)
	}
}

func TestSetName(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("one", nil, nil)
	if err := txn.SetName("hello"); nil != err {
		t.Error(err)
	}
	txn.End()
	if err := txn.SetName("three"); err != errAlreadyEnded {
		t.Error(err)
	}

	app.ExpectMetrics(t, backgroundMetrics)
}

func deferEndPanic(txn Transaction, panicMe interface{}) (r interface{}) {
	defer func() {
		r = recover()
	}()

	defer txn.End()

	panic(panicMe)
}

func TestPanicError(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)

	e := myError{}
	r := deferEndPanic(txn, e)
	if r != e {
		t.Error("panic not propagated", r)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   internal.PanicErrorKlass,
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     internal.PanicErrorKlass,
			"error.message":   "my msg",
			"transactionName": "OtherTransaction/Go/hello",
		},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestPanicString(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)

	e := "my string"
	r := deferEndPanic(txn, e)
	if r != e {
		t.Error("panic not propagated", r)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     "my string",
		Klass:   internal.PanicErrorKlass,
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     internal.PanicErrorKlass,
			"error.message":   "my string",
			"transactionName": "OtherTransaction/Go/hello",
		},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestPanicInt(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)

	e := 22
	r := deferEndPanic(txn, e)
	if r != e {
		t.Error("panic not propagated", r)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     "22",
		Klass:   internal.PanicErrorKlass,
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     internal.PanicErrorKlass,
			"error.message":   "22",
			"transactionName": "OtherTransaction/Go/hello",
		},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestPanicNil(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)

	r := deferEndPanic(txn, nil)
	if nil != r {
		t.Error(r)
	}

	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, backgroundMetrics)
}

func TestResponseCodeError(t *testing.T) {
	app := testApp(nil, nil, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)

	txn.WriteHeader(http.StatusBadRequest)   // 400
	txn.WriteHeader(http.StatusUnauthorized) // 401

	txn.End()

	if http.StatusBadRequest != w.Code {
		t.Error(w.Code)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "Bad Request",
		Klass:   "400",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "400",
			"error.message":   "Bad Request",
			"transactionName": "WebTransaction/Go/hello",
		},
		AgentAttributes: mergeAttributes(helloRequestAttributes, map[string]interface{}{
			"httpResponseCode": "400",
		}),
	}})
	app.ExpectMetrics(t, webErrorMetrics)
}

func TestResponseCode404Filtered(t *testing.T) {
	app := testApp(nil, nil, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)

	txn.WriteHeader(http.StatusNotFound)

	txn.End()

	if http.StatusNotFound != w.Code {
		t.Error(w.Code)
	}

	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, webMetrics)
}

func TestResponseCodeCustomFilter(t *testing.T) {
	cfgFn := func(cfg *Config) {
		cfg.ErrorCollector.IgnoreStatusCodes =
			append(cfg.ErrorCollector.IgnoreStatusCodes, 405)
	}
	app := testApp(nil, cfgFn, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)

	txn.WriteHeader(405)

	txn.End()

	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, webMetrics)
}

func TestResponseCodeServerSideFilterObserved(t *testing.T) {
	// Test that server-side ignore_status_codes are observed.
	cfgFn := func(cfg *Config) {
		cfg.ErrorCollector.IgnoreStatusCodes = nil
	}
	replyfn := func(reply *internal.ConnectReply) {
		json.Unmarshal([]byte(`{"agent_config":{"error_collector.ignore_status_codes":[405]}}`), reply)
	}
	app := testApp(replyfn, cfgFn, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)

	txn.WriteHeader(405)

	txn.End()

	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, webMetrics)
}

func TestResponseCodeServerSideOverwriteLocal(t *testing.T) {
	// Test that server-side ignore_status_codes are used in place of local
	// Config.ErrorCollector.IgnoreStatusCodes.
	cfgFn := func(cfg *Config) {
	}
	replyfn := func(reply *internal.ConnectReply) {
		json.Unmarshal([]byte(`{"agent_config":{"error_collector.ignore_status_codes":[402]}}`), reply)
	}
	app := testApp(replyfn, cfgFn, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)

	txn.WriteHeader(404)

	txn.End()

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "Not Found",
		Klass:   "404",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "404",
			"error.message":   "Not Found",
			"transactionName": "WebTransaction/Go/hello",
		},
		AgentAttributes: mergeAttributes(helloRequestAttributes, map[string]interface{}{
			"httpResponseCode": "404",
		}),
	}})
	app.ExpectMetrics(t, webErrorMetrics)
}

func TestResponseCodeAfterEnd(t *testing.T) {
	app := testApp(nil, nil, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)

	txn.End()
	txn.WriteHeader(http.StatusBadRequest)

	if http.StatusBadRequest != w.Code {
		t.Error(w.Code)
	}

	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, webMetrics)
}

func TestResponseCodeAfterWrite(t *testing.T) {
	app := testApp(nil, nil, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)

	txn.Write([]byte("zap"))
	txn.WriteHeader(http.StatusBadRequest)

	txn.End()

	if out := w.Body.String(); "zap" != out {
		t.Error(out)
	}

	if http.StatusOK != w.Code {
		t.Error(w.Code)
	}

	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, webMetrics)
}

func TestQueueTime(t *testing.T) {
	app := testApp(nil, nil, t)
	req, err := http.NewRequest("GET", helloPath+helloQueryParams, nil)
	req.Header.Add("X-Queue-Start", "1465793282.12345")
	if nil != err {
		t.Fatal(err)
	}
	txn := app.StartTransaction("hello", nil, req)
	txn.NoticeError(myError{})
	txn.End()

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "WebTransaction/Go/hello",
			"queueDuration":   internal.MatchAnything,
		},
		AgentAttributes: map[string]interface{}{
			"request.uri":    "/hello",
			"request.method": "GET",
		},
	}})
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "WebFrontend/QueueTime", Scope: "", Forced: true, Data: nil},
	}, webErrorMetrics...))
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "F",
			"queueDuration":    internal.MatchAnything,
		},
		AgentAttributes: nil,
	}})
}

func TestIgnore(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	txn.NoticeError(myError{})
	err := txn.Ignore()
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, []internal.WantMetric{})
	app.ExpectTxnEvents(t, []internal.WantEvent{})
}

func TestIgnoreAlreadyEnded(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	txn.NoticeError(myError{})
	txn.End()
	err := txn.Ignore()
	if err != errAlreadyEnded {
		t.Error(err)
	}
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "OtherTransaction/Go/hello",
		},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
	}})
}

func TestExternalSegmentMethod(t *testing.T) {
	req, err := http.NewRequest("POST", "http://request.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	responsereq, err := http.NewRequest("POST", "http://response.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	response := &http.Response{Request: responsereq}

	// empty segment
	m := externalSegmentMethod(&ExternalSegment{})
	if "" != m {
		t.Error(m)
	}

	// empty request
	m = externalSegmentMethod(&ExternalSegment{
		Request: nil,
	})
	if "" != m {
		t.Error(m)
	}

	// segment containing request and response
	m = externalSegmentMethod(&ExternalSegment{
		Request:  req,
		Response: response,
	})
	if "POST" != m {
		t.Error(m)
	}

	// Procedure field overrides request and response.
	m = externalSegmentMethod(&ExternalSegment{
		Procedure: "GET",
		Request:   req,
		Response:  response,
	})
	if "GET" != m {
		t.Error(m)
	}

	req, err = http.NewRequest("", "http://request.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	responsereq, err = http.NewRequest("", "http://response.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	response = &http.Response{Request: responsereq}

	// empty string method means a client GET request
	m = externalSegmentMethod(&ExternalSegment{
		Request:  req,
		Response: response,
	})
	if "GET" != m {
		t.Error(m)
	}

}

func TestExternalSegmentURL(t *testing.T) {
	rawURL := "http://url.com"
	req, err := http.NewRequest("GET", "http://request.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	responsereq, err := http.NewRequest("GET", "http://response.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	response := &http.Response{Request: responsereq}

	// empty segment
	u, err := externalSegmentURL(&ExternalSegment{})
	host := internal.HostFromURL(u)
	if nil != err || nil != u || "" != host {
		t.Error(u, err, internal.HostFromURL(u))
	}
	// segment only containing url
	u, err = externalSegmentURL(&ExternalSegment{URL: rawURL})
	host = internal.HostFromURL(u)
	if nil != err || host != "url.com" {
		t.Error(u, err, internal.HostFromURL(u))
	}
	// segment only containing request
	u, err = externalSegmentURL(&ExternalSegment{Request: req})
	host = internal.HostFromURL(u)
	if nil != err || "request.com" != host {
		t.Error(host)
	}
	// segment only containing response
	u, err = externalSegmentURL(&ExternalSegment{Response: response})
	host = internal.HostFromURL(u)
	if nil != err || "response.com" != host {
		t.Error(host)
	}
	// segment containing request and response
	u, err = externalSegmentURL(&ExternalSegment{
		Request:  req,
		Response: response,
	})
	host = internal.HostFromURL(u)
	if nil != err || "response.com" != host {
		t.Error(host)
	}
	// segment containing url, request, and response
	u, err = externalSegmentURL(&ExternalSegment{
		URL:      rawURL,
		Request:  req,
		Response: response,
	})
	host = internal.HostFromURL(u)
	if nil != err || "url.com" != host {
		t.Error(err, host)
	}
}

func TestZeroSegmentsSafe(t *testing.T) {
	s := Segment{}
	s.End()

	StartSegmentNow(nil)

	ds := DatastoreSegment{}
	ds.End()

	es := ExternalSegment{}
	es.End()

	StartSegment(nil, "").End()

	StartExternalSegment(nil, nil).End()
}

func TestTraceSegmentDefer(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	func() {
		defer StartSegment(txn, "segment").End()
	}()
	txn.End()
	scope := "WebTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Custom/segment", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/segment", Scope: scope, Forced: false, Data: nil},
	}, webMetrics...))
}

func TestTraceSegmentNilErr(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	err := StartSegment(txn, "segment").End()
	if nil != err {
		t.Error(err)
	}
	txn.End()
	scope := "WebTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Custom/segment", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/segment", Scope: scope, Forced: false, Data: nil},
	}, webMetrics...))
}

func TestTraceSegmentOutOfOrder(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	s1 := StartSegment(txn, "s1")
	s2 := StartSegment(txn, "s1")
	err1 := s1.End()
	err2 := s2.End()
	if nil != err1 {
		t.Error(err1)
	}
	if nil == err2 {
		t.Error(err2)
	}
	txn.End()
	scope := "WebTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Custom/s1", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/s1", Scope: scope, Forced: false, Data: nil},
	}, webMetrics...))
}

func TestTraceSegmentEndedBeforeStartSegment(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	txn.End()
	s := StartSegment(txn, "segment")
	err := s.End()
	if err != errAlreadyEnded {
		t.Error(err)
	}
	app.ExpectMetrics(t, webMetrics)
}

func TestTraceSegmentEndedBeforeEndSegment(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	s := StartSegment(txn, "segment")
	txn.End()
	err := s.End()
	if err != errAlreadyEnded {
		t.Error(err)
	}

	app.ExpectMetrics(t, webMetrics)
}

func TestTraceSegmentPanic(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	func() {
		defer func() {
			recover()
		}()

		func() {
			defer StartSegment(txn, "f1").End()

			func() {
				t := StartSegment(txn, "f2")

				func() {
					defer StartSegment(txn, "f3").End()

					func() {
						StartSegment(txn, "f4")

						panic(nil)
					}()
				}()

				t.End()
			}()
		}()
	}()

	txn.End()
	scope := "WebTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Custom/f1", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/f1", Scope: scope, Forced: false, Data: nil},
		{Name: "Custom/f3", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/f3", Scope: scope, Forced: false, Data: nil},
	}, webMetrics...))
}

func TestTraceSegmentNilTxn(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	s := Segment{Name: "hello"}
	err := s.End()
	if err != nil {
		t.Error(err)
	}
	txn.End()
	app.ExpectMetrics(t, webMetrics)
}

func TestTraceDatastore(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	s := DatastoreSegment{}
	s.StartTime = txn.StartSegmentNow()
	s.Product = DatastoreMySQL
	s.Collection = "my_table"
	s.Operation = "SELECT"
	err := s.End()
	if nil != err {
		t.Error(err)
	}
	txn.NoticeError(myError{})
	txn.End()
	scope := "WebTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/operation/MySQL/SELECT", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/my_table/SELECT", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/my_table/SELECT", Scope: scope, Forced: false, Data: nil},
	}, webErrorMetrics...))
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":       "newrelic.myError",
			"error.message":     "my msg",
			"transactionName":   "WebTransaction/Go/hello",
			"databaseCallCount": 1,
			"databaseDuration":  internal.MatchAnything,
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":              "WebTransaction/Go/hello",
			"nr.apdexPerfZone":  "F",
			"databaseCallCount": 1,
			"databaseDuration":  internal.MatchAnything,
		},
	}})
}

func TestTraceDatastoreBackground(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	s := DatastoreSegment{
		StartTime:  txn.StartSegmentNow(),
		Product:    DatastoreMySQL,
		Collection: "my_table",
		Operation:  "SELECT",
	}
	err := s.End()
	if nil != err {
		t.Error(err)
	}
	txn.NoticeError(myError{})
	txn.End()
	scope := "OtherTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/operation/MySQL/SELECT", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/my_table/SELECT", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/my_table/SELECT", Scope: scope, Forced: false, Data: nil},
	}, backgroundErrorMetrics...))
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":       "newrelic.myError",
			"error.message":     "my msg",
			"transactionName":   "OtherTransaction/Go/hello",
			"databaseCallCount": 1,
			"databaseDuration":  internal.MatchAnything,
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":              "OtherTransaction/Go/hello",
			"databaseCallCount": 1,
			"databaseDuration":  internal.MatchAnything,
		},
	}})
}

func TestTraceDatastoreMissingProductOperationCollection(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	s := DatastoreSegment{
		StartTime: txn.StartSegmentNow(),
	}
	err := s.End()
	if nil != err {
		t.Error(err)
	}
	txn.NoticeError(myError{})
	txn.End()
	scope := "WebTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/Unknown/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/Unknown/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/operation/Unknown/other", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/operation/Unknown/other", Scope: scope, Forced: false, Data: nil},
	}, webErrorMetrics...))
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":       "newrelic.myError",
			"error.message":     "my msg",
			"transactionName":   "WebTransaction/Go/hello",
			"databaseCallCount": 1,
			"databaseDuration":  internal.MatchAnything,
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":              "WebTransaction/Go/hello",
			"nr.apdexPerfZone":  "F",
			"databaseCallCount": 1,
			"databaseDuration":  internal.MatchAnything,
		},
	}})
}

func TestTraceDatastoreNilTxn(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	var s DatastoreSegment
	s.Product = DatastoreMySQL
	s.Collection = "my_table"
	s.Operation = "SELECT"
	err := s.End()
	if nil != err {
		t.Error(err)
	}
	txn.NoticeError(myError{})
	txn.End()
	app.ExpectMetrics(t, webErrorMetrics)
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "WebTransaction/Go/hello",
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "F",
		},
	}})
}

func TestTraceDatastoreTxnEnded(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	txn.NoticeError(myError{})
	s := DatastoreSegment{
		StartTime:  txn.StartSegmentNow(),
		Product:    DatastoreMySQL,
		Collection: "my_table",
		Operation:  "SELECT",
	}
	txn.End()
	err := s.End()
	if errAlreadyEnded != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, webErrorMetrics)
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "WebTransaction/Go/hello",
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "F",
		},
	}})
}

func TestTraceExternal(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	s := ExternalSegment{
		StartTime: txn.StartSegmentNow(),
		URL:       "http://example.com/",
	}
	err := s.End()
	if nil != err {
		t.Error(err)
	}
	txn.NoticeError(myError{})
	txn.End()
	scope := "WebTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "External/example.com/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/example.com/http", Scope: scope, Forced: false, Data: nil},
	}, webErrorMetrics...))
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":       "newrelic.myError",
			"error.message":     "my msg",
			"transactionName":   "WebTransaction/Go/hello",
			"externalCallCount": 1,
			"externalDuration":  internal.MatchAnything,
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":              "WebTransaction/Go/hello",
			"nr.apdexPerfZone":  "F",
			"externalCallCount": 1,
			"externalDuration":  internal.MatchAnything,
		},
	}})
}

func TestExternalSegmentCustomFieldsWithURL(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	s := ExternalSegment{
		StartTime: txn.StartSegmentNow(),
		URL:       "https://otherhost.com/path/zip/zap?secret=ssshhh",
		Host:      "bufnet",
		Procedure: "TestApplication/DoUnaryUnary",
		Library:   "grpc",
	}
	err := s.End()
	if nil != err {
		t.Error(err)
	}
	txn.End()
	scope := "WebTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "External/bufnet/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/bufnet/grpc/TestApplication/DoUnaryUnary", Scope: scope, Forced: false, Data: nil},
	}, webMetrics...))
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "WebTransaction/Go/hello",
				"sampled":       true,
				"category":      "generic",
				"nr.entryPoint": true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"name":      "External/bufnet/grpc/TestApplication/DoUnaryUnary",
				"category":  "http",
				"component": "grpc",
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				// "http.url" and "http.method" are not saved if
				// library is not "http".
			},
		},
	})
}

func TestExternalSegmentCustomFieldsWithRequest(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	req, _ := http.NewRequest("GET", "https://www.something.com/path/zip/zap?secret=ssshhh", nil)
	s := StartExternalSegment(txn, req)
	s.Host = "bufnet"
	s.Procedure = "TestApplication/DoUnaryUnary"
	s.Library = "grpc"
	err := s.End()
	if nil != err {
		t.Error(err)
	}
	txn.End()
	scope := "WebTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "External/bufnet/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/bufnet/grpc/TestApplication/DoUnaryUnary", Scope: scope, Forced: false, Data: nil},
	}, webMetrics...))
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "WebTransaction/Go/hello",
				"sampled":       true,
				"category":      "generic",
				"nr.entryPoint": true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"name":      "External/bufnet/grpc/TestApplication/DoUnaryUnary",
				"category":  "http",
				"component": "grpc",
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				// "http.url" and "http.method" are not saved if
				// library is not "http".
			},
		},
	})
}

func TestExternalSegmentCustomFieldsWithResponse(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	req, _ := http.NewRequest("GET", "https://www.something.com/path/zip/zap?secret=ssshhh", nil)
	resp := &http.Response{Request: req}
	s := ExternalSegment{
		StartTime: txn.StartSegmentNow(),
		Response:  resp,
		Host:      "bufnet",
		Procedure: "TestApplication/DoUnaryUnary",
		Library:   "grpc",
	}
	err := s.End()
	if nil != err {
		t.Error(err)
	}
	txn.End()
	scope := "WebTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "External/bufnet/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/bufnet/grpc/TestApplication/DoUnaryUnary", Scope: scope, Forced: false, Data: nil},
	}, webMetrics...))
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "WebTransaction/Go/hello",
				"sampled":       true,
				"category":      "generic",
				"nr.entryPoint": true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"name":      "External/bufnet/grpc/TestApplication/DoUnaryUnary",
				"category":  "http",
				"component": "grpc",
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				// "http.url" and "http.method" are not saved if
				// library is not "http".
			},
		},
	})
}

func TestTraceExternalBadURL(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	s := ExternalSegment{
		StartTime: txn.StartSegmentNow(),
		URL:       ":example.com/",
	}
	err := s.End()
	if nil == err {
		t.Error(err)
	}
	txn.NoticeError(myError{})
	txn.End()
	app.ExpectMetrics(t, webErrorMetrics)
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "WebTransaction/Go/hello",
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "F",
		},
	}})
}

func TestTraceExternalBackground(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	s := ExternalSegment{
		StartTime: txn.StartSegmentNow(),
		URL:       "http://example.com/",
	}
	err := s.End()
	if nil != err {
		t.Error(err)
	}
	txn.NoticeError(myError{})
	txn.End()
	scope := "OtherTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/example.com/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/example.com/http", Scope: scope, Forced: false, Data: nil},
	}, backgroundErrorMetrics...))
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":       "newrelic.myError",
			"error.message":     "my msg",
			"transactionName":   "OtherTransaction/Go/hello",
			"externalCallCount": 1,
			"externalDuration":  internal.MatchAnything,
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":              "OtherTransaction/Go/hello",
			"externalCallCount": 1,
			"externalDuration":  internal.MatchAnything,
		},
	}})
}

func TestTraceExternalMissingURL(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	s := ExternalSegment{
		StartTime: txn.StartSegmentNow(),
	}
	err := s.End()
	if nil != err {
		t.Error(err)
	}
	txn.NoticeError(myError{})
	txn.End()
	scope := "WebTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "External/unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/unknown/http", Scope: scope, Forced: false, Data: nil},
	}, webErrorMetrics...))
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":       "newrelic.myError",
			"error.message":     "my msg",
			"transactionName":   "WebTransaction/Go/hello",
			"externalCallCount": 1,
			"externalDuration":  internal.MatchAnything,
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":              "WebTransaction/Go/hello",
			"nr.apdexPerfZone":  "F",
			"externalCallCount": 1,
			"externalDuration":  internal.MatchAnything,
		},
	}})
}

func TestTraceExternalNilTxn(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	txn.NoticeError(myError{})
	var s ExternalSegment
	err := s.End()
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectMetrics(t, webErrorMetrics)
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "WebTransaction/Go/hello",
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "F",
		},
	}})
}

func TestTraceExternalTxnEnded(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	txn.NoticeError(myError{})
	s := ExternalSegment{
		StartTime: txn.StartSegmentNow(),
		URL:       "http://example.com/",
	}
	txn.End()
	err := s.End()
	if err != errAlreadyEnded {
		t.Error(err)
	}
	app.ExpectMetrics(t, webErrorMetrics)
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "WebTransaction/Go/hello",
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "F",
		},
	}})
}

func TestRoundTripper(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	url := "http://example.com/"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("zip", "zap")
	client := &http.Client{}
	inner := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		catHdr := r.Header.Get(DistributedTracePayloadHeader)
		if "" == catHdr {
			t.Error("cat header missing")
		}
		// Test that headers are preserved during reqest cloning:
		if z := r.Header.Get("zip"); z != "zap" {
			t.Error("missing header", z)
		}
		if r.URL.String() != url {
			t.Error(r.URL.String())
		}
		return nil, errors.New("hello")
	})
	client.Transport = NewRoundTripper(txn, inner)
	resp, err := client.Do(req)
	if resp != nil || err == nil {
		t.Error(resp, err.Error())
	}
	// Ensure that the request was cloned:
	catHdr := req.Header.Get(DistributedTracePayloadHeader)
	if "" != catHdr {
		t.Error("cat header unexpectedly present")
	}
	txn.NoticeError(myError{})
	txn.End()
	scope := "OtherTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/example.com/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/example.com/http/GET", Scope: scope, Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Data: nil},
	}, backgroundErrorMetrics...))
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":       "newrelic.myError",
			"error.message":     "my msg",
			"transactionName":   "OtherTransaction/Go/hello",
			"externalCallCount": 1,
			"externalDuration":  internal.MatchAnything,
			"guid":              internal.MatchAnything,
			"traceId":           internal.MatchAnything,
			"priority":          internal.MatchAnything,
			"sampled":           internal.MatchAnything,
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":              "OtherTransaction/Go/hello",
			"externalCallCount": 1,
			"externalDuration":  internal.MatchAnything,
			"guid":              internal.MatchAnything,
			"traceId":           internal.MatchAnything,
			"priority":          internal.MatchAnything,
			"sampled":           internal.MatchAnything,
		},
	}})
}

func TestRoundTripperOldCAT(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	url := "http://example.com/"
	client := &http.Client{}
	inner := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		// TODO test that request headers have been set here.
		if r.URL.String() != url {
			t.Error(r.URL.String())
		}
		return nil, errors.New("hello")
	})
	client.Transport = NewRoundTripper(txn, inner)
	resp, err := client.Get(url)
	if resp != nil || err == nil {
		t.Error(resp, err.Error())
	}
	txn.NoticeError(myError{})
	txn.End()
	scope := "OtherTransaction/Go/hello"
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/example.com/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/example.com/http/GET", Scope: scope, Forced: false, Data: nil},
	}, backgroundErrorMetrics...))
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":       "newrelic.myError",
			"error.message":     "my msg",
			"transactionName":   "OtherTransaction/Go/hello",
			"externalCallCount": 1,
			"externalDuration":  internal.MatchAnything,
		},
	}})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":              "OtherTransaction/Go/hello",
			"externalCallCount": 1,
			"externalDuration":  internal.MatchAnything,
			"nr.tripId":         internal.MatchAnything,
			"nr.guid":           internal.MatchAnything,
			"nr.pathHash":       internal.MatchAnything,
		},
	}})
}

func TestTraceBelowThreshold(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{})
}

func TestTraceBelowThresholdBackground(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{})
}

func TestTraceNoSegments(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
		cfg.TransactionTracer.SegmentThreshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName:  "WebTransaction/Go/hello",
		NumSegments: 0,
	}})
}

func TestTraceDisabledLocally(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
		cfg.TransactionTracer.SegmentThreshold = 0
		cfg.TransactionTracer.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{})
}

func TestTraceDisabledByServerSideConfig(t *testing.T) {
	// Test that server-side-config trace-enabled-setting can disable transaction
	// traces.
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
		cfg.TransactionTracer.SegmentThreshold = 0
	}
	replyfn := func(reply *internal.ConnectReply) {
		json.Unmarshal([]byte(`{"agent_config":{"transaction_tracer.enabled":false}}`), reply)
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{})
}

func TestTraceEnabledByServerSideConfig(t *testing.T) {
	// Test that server-side-config trace-enabled-setting can enable
	// transaction traces (and hence server-side-config has priority).
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
		cfg.TransactionTracer.SegmentThreshold = 0
		cfg.TransactionTracer.Enabled = false
	}
	replyfn := func(reply *internal.ConnectReply) {
		json.Unmarshal([]byte(`{"agent_config":{"transaction_tracer.enabled":true}}`), reply)
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName:  "WebTransaction/Go/hello",
		NumSegments: 0,
	}})
}

func TestTraceDisabledRemotelyOverridesServerSideConfig(t *testing.T) {
	// Test that the connect reply "collect_traces" setting overrides the
	// "transaction_tracer.enabled" server side config setting.
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
		cfg.TransactionTracer.SegmentThreshold = 0
		cfg.TransactionTracer.Enabled = true
	}
	replyfn := func(reply *internal.ConnectReply) {
		json.Unmarshal([]byte(`{"agent_config":{"transaction_tracer.enabled":true},"collect_traces":false}`), reply)
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{})
}

func TestTraceDisabledRemotely(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
		cfg.TransactionTracer.SegmentThreshold = 0
	}
	replyfn := func(reply *internal.ConnectReply) {
		reply.CollectTraces = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{})
}

func TestTraceWithSegments(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
		cfg.TransactionTracer.SegmentThreshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	s1 := StartSegment(txn, "s1")
	s1.End()
	s2 := ExternalSegment{
		StartTime: StartSegmentNow(txn),
		URL:       "http://example.com",
	}
	s2.End()
	s3 := DatastoreSegment{
		StartTime:  StartSegmentNow(txn),
		Product:    DatastoreMySQL,
		Collection: "my_table",
		Operation:  "SELECT",
	}
	s3.End()
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName:  "WebTransaction/Go/hello",
		NumSegments: 3,
	}})
}

func TestTraceSegmentsBelowThreshold(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
		cfg.TransactionTracer.SegmentThreshold = 1 * time.Hour
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	s1 := StartSegment(txn, "s1")
	s1.End()
	s2 := ExternalSegment{
		StartTime: StartSegmentNow(txn),
		URL:       "http://example.com",
	}
	s2.End()
	s3 := DatastoreSegment{
		StartTime:  StartSegmentNow(txn),
		Product:    DatastoreMySQL,
		Collection: "my_table",
		Operation:  "SELECT",
	}
	s3.End()
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName:  "WebTransaction/Go/hello",
		NumSegments: 0,
	}})
}

func TestNoticeErrorTxnEvents(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(myError{})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":  "OtherTransaction/Go/hello",
			"error": true,
		},
	}})
}

func TestTransactionApplication(t *testing.T) {
	txn := testApp(nil, nil, t).StartTransaction("hello", nil, nil)
	app := txn.Application()
	err := app.RecordCustomMetric("myMetric", 123.0)
	if nil != err {
		t.Error(err)
	}
	expectData := []float64{1, 123.0, 123.0, 123.0, 123.0, 123.0 * 123.0}
	app.(expectApp).ExpectMetrics(t, []internal.WantMetric{
		{Name: "Custom/myMetric", Scope: "", Forced: false, Data: expectData},
	})
}

func TestNilSegmentPointerEnd(t *testing.T) {
	var basicSegment *Segment
	var datastoreSegment *DatastoreSegment
	var externalSegment *ExternalSegment

	// These calls on nil pointer receivers should not panic.
	basicSegment.End()
	datastoreSegment.End()
	externalSegment.End()
}

type flushWriter struct{}

func (f flushWriter) WriteHeader(int)           {}
func (f flushWriter) Write([]byte) (int, error) { return 0, nil }
func (f flushWriter) Header() http.Header       { return nil }
func (f flushWriter) Flush()                    {}

func TestAsync(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", flushWriter{}, nil)
	if _, ok := txn.(http.Flusher); !ok {
		t.Error("transaction should have flush")
	}
	s1 := StartSegment(txn, "mainThread")
	asyncThread := txn.NewGoroutine()
	// Test that the async transaction reference has the correct optional
	// interface behavior.
	if _, ok := asyncThread.(http.Flusher); !ok {
		t.Error("async transaction reference should have flush")
	}
	s2 := StartSegment(asyncThread, "asyncThread")
	// End segments in interleaved order.
	s1.End()
	s2.End()
	// Test that the async transaction reference has the expected
	// transaction method behavior.
	asyncThread.AddAttribute("zip", "zap")
	// Test that the transaction ends when the async transaction is ended.
	if err := asyncThread.End(); nil != err {
		t.Error(err)
	}
	threadAfterEnd := asyncThread.NewGoroutine()
	if _, ok := threadAfterEnd.(http.Flusher); !ok {
		t.Error("after end transaction reference should have flush")
	}
	if err := threadAfterEnd.End(); err != errAlreadyEnded {
		t.Error(err)
	}
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
		UserAttributes: map[string]interface{}{
			"zip": "zap",
		},
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/mainThread", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/mainThread", Scope: "OtherTransaction/Go/hello", Forced: false, Data: nil},
		{Name: "Custom/asyncThread", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/asyncThread", Scope: "OtherTransaction/Go/hello", Forced: false, Data: nil},
	})
}

func TestMessageProducerSegmentBasic(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	s := MessageProducerSegment{
		StartTime:       StartSegmentNow(txn),
		Library:         "RabbitMQ",
		DestinationType: MessageQueue,
		DestinationName: "myQueue",
	}
	err := s.End()
	if err != nil {
		t.Error(err)
	}
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "MessageBroker/RabbitMQ/Queue/Produce/Named/myQueue", Scope: "", Forced: false, Data: nil},
		{Name: "MessageBroker/RabbitMQ/Queue/Produce/Named/myQueue", Scope: "OtherTransaction/Go/hello", Forced: false, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "OtherTransaction/Go/hello",
				"sampled":       true,
				"category":      "generic",
				"nr.entryPoint": true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"parentId": internal.MatchAnything,
				"name":     "MessageBroker/RabbitMQ/Queue/Produce/Named/myQueue",
				"category": "generic",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestMessageProducerSegmentMissingDestinationType(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	s := MessageProducerSegment{
		StartTime:       StartSegmentNow(txn),
		Library:         "RabbitMQ",
		DestinationName: "myQueue",
	}
	err := s.End()
	if err != nil {
		t.Error(err)
	}
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "MessageBroker/RabbitMQ/Queue/Produce/Named/myQueue", Scope: "", Forced: false, Data: nil},
		{Name: "MessageBroker/RabbitMQ/Queue/Produce/Named/myQueue", Scope: "OtherTransaction/Go/hello", Forced: false, Data: nil},
	})
}

func TestMessageProducerSegmentTemp(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	s := MessageProducerSegment{
		StartTime:            StartSegmentNow(txn),
		Library:              "RabbitMQ",
		DestinationType:      MessageQueue,
		DestinationTemporary: true,
		DestinationName:      "myQueue0123456789",
	}
	err := s.End()
	if err != nil {
		t.Error(err)
	}
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "MessageBroker/RabbitMQ/Queue/Produce/Temp", Scope: "", Forced: false, Data: nil},
		{Name: "MessageBroker/RabbitMQ/Queue/Produce/Temp", Scope: "OtherTransaction/Go/hello", Forced: false, Data: nil},
	})
}

func TestMessageProducerSegmentNoName(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	s := MessageProducerSegment{
		StartTime:       StartSegmentNow(txn),
		Library:         "RabbitMQ",
		DestinationType: MessageQueue,
	}
	err := s.End()
	if err != nil {
		t.Error(err)
	}
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "MessageBroker/RabbitMQ/Queue/Produce/Named/Unknown", Scope: "", Forced: false, Data: nil},
		{Name: "MessageBroker/RabbitMQ/Queue/Produce/Named/Unknown", Scope: "OtherTransaction/Go/hello", Forced: false, Data: nil},
	})
}

func TestMessageProducerSegmentTxnEnded(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	s := MessageProducerSegment{
		StartTime:            StartSegmentNow(txn),
		Library:              "RabbitMQ",
		DestinationType:      MessageQueue,
		DestinationTemporary: true,
		DestinationName:      "myQueue0123456789",
	}
	txn.End()
	err := s.End()
	if err != errAlreadyEnded {
		t.Error("expected already ended error", err)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
	})
}

func TestMessageProducerSegmentNilTxn(t *testing.T) {
	s := MessageProducerSegment{
		StartTime:            StartSegmentNow(nil),
		Library:              "RabbitMQ",
		DestinationType:      MessageQueue,
		DestinationTemporary: true,
		DestinationName:      "myQueue0123456789",
	}
	s.End()
}

func TestMessageProducerSegmentNilSegment(t *testing.T) {
	var s *MessageProducerSegment
	s.End()
}
