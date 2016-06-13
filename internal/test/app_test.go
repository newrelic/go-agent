package test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	newrelic "github.com/newrelic/go-sdk"
	"github.com/newrelic/go-sdk/api"
	ats "github.com/newrelic/go-sdk/attributes"
	"github.com/newrelic/go-sdk/internal"
)

func TestNewApplicationNonNil(t *testing.T) {
	cfg := api.NewConfig("appname", "short license key")
	cfg.Development = true
	app, err := newrelic.NewApplication(cfg)
	if nil == err {
		t.Error("error expected when license key is short")
	}
	if nil != app {
		t.Error("app expected to be nil when error is returned")
	}
}

var (
	sampleLicense = "0123456789012345678901234567890123456789"
	validParams   = map[string]interface{}{"zip": 1, "zap": 2}
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
)

func handler(w http.ResponseWriter, req *http.Request) {
	w.Write(helloResponse)
}

func BenchmarkMuxWithoutNewRelic(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc(helloPath, handler)

	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, helloRequest)
	}
}

func BenchmarkMuxWithNewRelic(b *testing.B) {
	app := testApp(nil, nil, b)
	mux := http.NewServeMux()
	mux.HandleFunc(newrelic.WrapHandleFunc(app, helloPath, handler))

	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, helloRequest)
	}
}

func BenchmarkMuxDevelopmentMode(b *testing.B) {
	cfg := newrelic.NewConfig("my app", sampleLicense)
	cfg.Development = true
	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		b.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc(newrelic.WrapHandleFunc(app, helloPath, handler))

	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(w, helloRequest)
	}
}

func testApp(replyfn func(*internal.ConnectReply), cfgfn func(*api.Config), t testing.TB) internal.ExpectApp {
	cfg := api.NewConfig("my app", "0123456789012345678901234567890123456789")

	if nil != cfgfn {
		cfgfn(&cfg)
	}

	app, err := internal.NewTestApp(replyfn, cfg)
	if nil != err {
		t.Fatal(err)
	}
	return app
}

func TestRecordCustomEventSuccess(t *testing.T) {
	app := testApp(nil, nil, t)
	err := app.RecordCustomEvent("myType", validParams)
	if nil != err {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantCustomEvent{{"myType", validParams}})
}

func TestRecordCustomEventHighSecurityEnabled(t *testing.T) {
	cfgfn := func(cfg *api.Config) { cfg.HighSecurity = true }
	app := testApp(nil, cfgfn, t)
	err := app.RecordCustomEvent("myType", validParams)
	if err != internal.ErrHighSecurityEnabled {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantCustomEvent{})
}

func TestRecordCustomEventEventsDisabled(t *testing.T) {
	cfgfn := func(cfg *api.Config) { cfg.CustomInsightsEvents.Enabled = false }
	app := testApp(nil, cfgfn, t)
	err := app.RecordCustomEvent("myType", validParams)
	if err != internal.ErrCustomEventsDisabled {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantCustomEvent{})
}

func TestRecordCustomEventBadInput(t *testing.T) {
	app := testApp(nil, nil, t)
	err := app.RecordCustomEvent("????", validParams)
	if err != internal.ErrEventTypeRegex {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantCustomEvent{})
}

func TestRecordCustomEventRemoteDisable(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) { reply.CollectCustomEvents = false }
	app := testApp(replyfn, nil, t)
	err := app.RecordCustomEvent("myType", validParams)
	if err != internal.ErrCustomEventsRemoteDisabled {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantCustomEvent{})
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
	txn := app.StartTransaction("myName", w, nil)
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

type myError struct{}

func (e myError) Error() string { return "my msg" }

func TestNoticeErrorBackground(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	err := txn.NoticeError(myError{})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "test.myError",
		Caller:  "test.TestNoticeErrorBackground",
		URL:     "",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "test.myError",
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/OtherTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
}

func TestNoticeErrorWeb(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	err := txn.NoticeError(myError{})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "test.myError",
		Caller:  "test.TestNoticeErrorWeb",
		URL:     "/hello",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "WebTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "test.myError",
	}})
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
}

func TestNoticeErrorTxnEnded(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	txn.End()
	err := txn.NoticeError(myError{})
	if err != internal.ErrAlreadyEnded {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
	})
}

func TestNoticeErrorHighSecurity(t *testing.T) {
	cfgFn := func(cfg *api.Config) { cfg.HighSecurity = true }
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("myName", nil, nil)
	err := txn.NoticeError(myError{})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     internal.HighSecurityErrorMsg,
		Klass:   "test.myError",
		Caller:  "test.TestNoticeErrorHighSecurity",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     internal.HighSecurityErrorMsg,
		Klass:   "test.myError",
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/OtherTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
}

func TestNoticeErrorLocallyDisabled(t *testing.T) {
	cfgFn := func(cfg *api.Config) { cfg.ErrorCollector.Enabled = false }
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("myName", nil, nil)
	err := txn.NoticeError(myError{})
	if internal.ErrorsLocallyDisabled != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/OtherTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
}

func TestNoticeErrorRemotelyDisabled(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) { reply.CollectErrors = false }
	app := testApp(replyfn, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	err := txn.NoticeError(myError{})
	if internal.ErrorsRemotelyDisabled != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/OtherTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
}

func TestNoticeErrorNil(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	err := txn.NoticeError(nil)
	if internal.ErrNilError != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
	})
}

func TestNoticeErrorEventsLocallyDisabled(t *testing.T) {
	cfgFn := func(cfg *api.Config) { cfg.ErrorCollector.CaptureEvents = false }
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("myName", nil, nil)
	err := txn.NoticeError(myError{})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "test.myError",
		Caller:  "test.TestNoticeErrorEventsLocallyDisabled",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/OtherTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
}

func TestNoticeErrorEventsRemotelyDisabled(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) { reply.CollectErrorEvents = false }
	app := testApp(replyfn, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	err := txn.NoticeError(myError{})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "test.myError",
		Caller:  "test.TestNoticeErrorEventsRemotelyDisabled",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/OtherTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
}

func TestTransactionEventWeb(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	err := txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{
		{Name: "WebTransaction/Go/myName", Zone: "S"},
	})
}

func TestTransactionEventBackground(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	err := txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{
		{Name: "OtherTransaction/Go/myName"},
	})
}

func TestTransactionEventLocallyDisabled(t *testing.T) {
	cfgFn := func(cfg *api.Config) { cfg.TransactionEvents.Enabled = false }
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	err := txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{})
}

func TestTransactionEventRemotelyDisabled(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) { reply.CollectAnalyticsEvents = false }
	app := testApp(replyfn, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	err := txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{})
}

func myErrorHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("my response"))
	if txn, ok := w.(newrelic.Transaction); ok {
		txn.NoticeError(myError{})
	}
}

func TestWrapHandleFunc(t *testing.T) {
	app := testApp(nil, nil, t)
	mux := http.NewServeMux()
	mux.HandleFunc(newrelic.WrapHandleFunc(app, helloPath, myErrorHandler))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, helloRequest)

	out := w.Body.String()
	if "my response" != out {
		t.Error(out)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "test.myError",
		Caller:  "test.myErrorHandler",
		URL:     "/hello",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "test.myError",
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/hello", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/hello", "", false, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allWeb", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/WebTransaction/Go/hello", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
}

func TestWrapHandle(t *testing.T) {
	app := testApp(nil, nil, t)
	mux := http.NewServeMux()
	mux.Handle(newrelic.WrapHandle(app, helloPath, http.HandlerFunc(myErrorHandler)))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, helloRequest)

	out := w.Body.String()
	if "my response" != out {
		t.Error(out)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "test.myError",
		Caller:  "test.myErrorHandler",
		URL:     "/hello",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "test.myError",
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/hello", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/hello", "", false, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allWeb", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/WebTransaction/Go/hello", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
}

func TestSetName(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("one", nil, nil)
	if err := txn.SetName("two"); nil != err {
		t.Error(err)
	}
	txn.End()
	if err := txn.SetName("three"); err != internal.ErrAlreadyEnded {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/two", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
	})
}

func deferEndPanic(txn newrelic.Transaction, panicMe interface{}) (r interface{}) {
	defer func() {
		r = recover()
	}()

	defer txn.End()

	panic(panicMe)
}

func TestPanicError(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)

	e := myError{}
	r := deferEndPanic(txn, e)
	if r != e {
		t.Error("panic not propogated", r)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   internal.PanicErrorKlass,
		Caller:  "internal.(*txn).End",
		URL:     "",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   internal.PanicErrorKlass,
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/OtherTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
}

func TestPanicString(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)

	e := "my string"
	r := deferEndPanic(txn, e)
	if r != e {
		t.Error("panic not propogated", r)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my string",
		Klass:   internal.PanicErrorKlass,
		Caller:  "internal.(*txn).End",
		URL:     "",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my string",
		Klass:   internal.PanicErrorKlass,
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/OtherTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
}

func TestPanicInt(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)

	e := 22
	r := deferEndPanic(txn, e)
	if r != e {
		t.Error("panic not propogated", r)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "22",
		Klass:   internal.PanicErrorKlass,
		Caller:  "internal.(*txn).End",
		URL:     "",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "22",
		Klass:   internal.PanicErrorKlass,
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/OtherTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
}

func TestPanicNil(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)

	r := deferEndPanic(txn, nil)
	if nil != r {
		t.Error(r)
	}

	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
	})
}

func TestResponseCodeError(t *testing.T) {
	app := testApp(nil, nil, t)
	w := httptest.NewRecorder()
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
		Caller:  "internal.(*txn).WriteHeader",
		URL:     "/hello",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "Bad Request",
		Klass:   "400",
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/hello", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/hello", "", false, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allWeb", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/WebTransaction/Go/hello", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
}

func TestResponseCode404Filtered(t *testing.T) {
	app := testApp(nil, nil, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)

	txn.WriteHeader(http.StatusNotFound)

	txn.End()

	if http.StatusNotFound != w.Code {
		t.Error(w.Code)
	}

	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/hello", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/hello", "", false, nil},
	})
}

func TestResponseCodeCustomFilter(t *testing.T) {
	cfgFn := func(cfg *api.Config) {
		cfg.ErrorCollector.IgnoreStatusCodes =
			append(cfg.ErrorCollector.IgnoreStatusCodes,
				http.StatusNotFound)
	}
	app := testApp(nil, cfgFn, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)

	txn.WriteHeader(http.StatusNotFound)

	txn.End()

	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/hello", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/hello", "", false, nil},
	})
}

func TestResponseCodeAfterEnd(t *testing.T) {
	app := testApp(nil, nil, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)

	txn.End()
	txn.WriteHeader(http.StatusBadRequest)

	if http.StatusBadRequest != w.Code {
		t.Error(w.Code)
	}

	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/hello", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/hello", "", false, nil},
	})
}

func TestResponseCodeAfterWrite(t *testing.T) {
	app := testApp(nil, nil, t)
	w := httptest.NewRecorder()
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
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/hello", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/hello", "", false, nil},
	})
}

func TestUserAttributeBasics(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)

	txn.NoticeError(errors.New("zap"))

	if err := txn.AddAttribute(`int\key`, 1); nil != err {
		t.Error(err)
	}
	if err := txn.AddAttribute(`str\key`, `zip\zap`); nil != err {
		t.Error(err)
	}
	err := txn.AddAttribute("invalid_value", struct{}{})
	if _, ok := err.(internal.ErrInvalidAttribute); !ok {
		t.Error(err)
	}
	txn.End()
	if err := txn.AddAttribute("already_ended", "zap"); err != internal.ErrAlreadyEnded {
		t.Error(err)
	}

	agentAttributes := map[string]interface{}{}
	userAttributes := map[string]interface{}{`int\key`: 1, `str\key`: `zip\zap`}

	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "OtherTransaction/Go/hello",
		Zone:            "",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "OtherTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestUserAttributeBasics",
		URL:             "",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "OtherTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestUserAttributeConfiguration(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.TransactionEvents.Attributes.Exclude = []string{"only_errors"}
		cfg.ErrorCollector.Attributes.Exclude = []string{"only_txn_events"}
		cfg.Attributes.Exclude = []string{"completed_excluded"}
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)

	txn.NoticeError(errors.New("zap"))

	if err := txn.AddAttribute("only_errors", 1); nil != err {
		t.Error(err)
	}
	if err := txn.AddAttribute("only_txn_events", 2); nil != err {
		t.Error(err)
	}
	if err := txn.AddAttribute("completed_excluded", 3); nil != err {
		t.Error(err)
	}
	txn.End()

	agentAttributes := map[string]interface{}{}
	errorUserAttributes := map[string]interface{}{"only_errors": 1}
	txnEventUserAttributes := map[string]interface{}{"only_txn_events": 2}

	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "OtherTransaction/Go/hello",
		Zone:            "",
		AgentAttributes: agentAttributes,
		UserAttributes:  txnEventUserAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "OtherTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestUserAttributeConfiguration",
		URL:             "",
		AgentAttributes: agentAttributes,
		UserAttributes:  errorUserAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "OtherTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  errorUserAttributes,
	}})
}

func TestAgentAttributes(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.HostDisplayName = `my\host\display\name`
	}

	app := testApp(nil, cfgfn, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))

	hdr := txn.Header()
	hdr.Set("Content-Type", `text/plain; charset=us-ascii`)
	hdr.Set("Content-Length", `345`)

	txn.WriteHeader(404)
	txn.End()

	userAttributes := map[string]interface{}{}
	agentAttributes := map[string]interface{}{
		ats.HostDisplayName:              `my\host\display\name`,
		ats.ResponseCode:                 `404`,
		ats.ResponseHeadersContentType:   `text/plain; charset=us-ascii`,
		ats.ResponseHeadersContentLength: 345,
		ats.RequestMethod:                "GET",
		ats.RequestAcceptHeader:          "text/plain",
		ats.RequestContentType:           "text/html; charset=utf-8",
		ats.RequestContentLength:         753,
		ats.RequestHeadersHost:           "my_domain.com",
	}

	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})

	agentAttributes[ats.RequestHeadersUserAgent] = "Mozilla/5.0"
	agentAttributes[ats.RequestHeadersReferer] = "http://en.wikipedia.org/zip"

	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestAgentAttributes",
		URL:             "/hello",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestAttributesDisabled(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.Attributes.Enabled = false
		cfg.HostDisplayName = `my\host\display\name`
	}

	app := testApp(nil, cfgfn, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))

	hdr := txn.Header()
	hdr.Set("Content-Type", `text/plain; charset=us-ascii`)
	hdr.Set("Content-Length", `345`)

	txn.WriteHeader(404)
	txn.AddAttribute("my_attribute", "zip")
	txn.End()

	userAttributes := map[string]interface{}{}
	agentAttributes := map[string]interface{}{}

	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestAttributesDisabled",
		URL:             "/hello",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestDefaultResponseCode(t *testing.T) {
	app := testApp(nil, nil, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))
	txn.Write([]byte("hello"))
	txn.End()

	userAttributes := map[string]interface{}{}
	agentAttributes := map[string]interface{}{
		ats.ResponseCode:         `200`,
		ats.RequestMethod:        "GET",
		ats.RequestAcceptHeader:  "text/plain",
		ats.RequestContentType:   "text/html; charset=utf-8",
		ats.RequestContentLength: 753,
		ats.RequestHeadersHost:   "my_domain.com",
	}

	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})

	agentAttributes[ats.RequestHeadersUserAgent] = "Mozilla/5.0"
	agentAttributes[ats.RequestHeadersReferer] = "http://en.wikipedia.org/zip"

	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestDefaultResponseCode",
		URL:             "/hello",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestTxnEventAttributesDisabled(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.TransactionEvents.Attributes.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))
	txn.AddAttribute("myStr", "hello")
	txn.Write([]byte("hello"))
	txn.End()

	userAttributes := map[string]interface{}{
		"myStr": "hello",
	}
	agentAttributes := map[string]interface{}{
		ats.ResponseCode:         `200`,
		ats.RequestMethod:        "GET",
		ats.RequestAcceptHeader:  "text/plain",
		ats.RequestContentType:   "text/html; charset=utf-8",
		ats.RequestContentLength: 753,
		ats.RequestHeadersHost:   "my_domain.com",
	}
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  map[string]interface{}{},
	}})

	agentAttributes[ats.RequestHeadersUserAgent] = "Mozilla/5.0"
	agentAttributes[ats.RequestHeadersReferer] = "http://en.wikipedia.org/zip"

	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestTxnEventAttributesDisabled",
		URL:             "/hello",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestErrorAttributesDisabled(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.ErrorCollector.Attributes.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))
	txn.AddAttribute("myStr", "hello")
	txn.Write([]byte("hello"))
	txn.End()

	userAttributes := map[string]interface{}{
		"myStr": "hello",
	}
	agentAttributes := map[string]interface{}{
		ats.ResponseCode:         `200`,
		ats.RequestMethod:        "GET",
		ats.RequestAcceptHeader:  "text/plain",
		ats.RequestContentType:   "text/html; charset=utf-8",
		ats.RequestContentLength: 753,
		ats.RequestHeadersHost:   "my_domain.com",
	}
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestErrorAttributesDisabled",
		URL:             "/hello",
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  map[string]interface{}{},
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  map[string]interface{}{},
	}})
}

var (
	allAgentAttributeNames = []string{
		ats.ResponseCode,
		ats.RequestMethod,
		ats.RequestAcceptHeader,
		ats.RequestContentType,
		ats.RequestContentLength,
		ats.RequestHeadersHost,
		ats.ResponseHeadersContentType,
		ats.ResponseHeadersContentLength,
		ats.HostDisplayName,
		ats.RequestHeadersUserAgent,
		ats.RequestHeadersReferer,
	}
)

func TestAgentAttributesExcluded(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.HostDisplayName = `my\host\display\name`
		cfg.Attributes.Exclude = allAgentAttributeNames
	}

	app := testApp(nil, cfgfn, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))

	hdr := txn.Header()
	hdr.Set("Content-Type", `text/plain; charset=us-ascii`)
	hdr.Set("Content-Length", `345`)

	txn.WriteHeader(404)
	txn.End()

	userAttributes := map[string]interface{}{}
	agentAttributes := map[string]interface{}{}

	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestAgentAttributesExcluded",
		URL:             "/hello",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestAgentAttributesExcludedFromErrors(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.HostDisplayName = `my\host\display\name`
		cfg.ErrorCollector.Attributes.Exclude = allAgentAttributeNames
	}

	app := testApp(nil, cfgfn, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))

	hdr := txn.Header()
	hdr.Set("Content-Type", `text/plain; charset=us-ascii`)
	hdr.Set("Content-Length", `345`)

	txn.WriteHeader(404)
	txn.End()

	userAttributes := map[string]interface{}{}
	agentAttributes := map[string]interface{}{
		ats.HostDisplayName:              `my\host\display\name`,
		ats.ResponseCode:                 `404`,
		ats.ResponseHeadersContentType:   `text/plain; charset=us-ascii`,
		ats.ResponseHeadersContentLength: 345,
		ats.RequestMethod:                "GET",
		ats.RequestAcceptHeader:          "text/plain",
		ats.RequestContentType:           "text/html; charset=utf-8",
		ats.RequestContentLength:         753,
		ats.RequestHeadersHost:           "my_domain.com",
	}
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestAgentAttributesExcludedFromErrors",
		URL:             "/hello",
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  userAttributes,
	}})
}

func TestAgentAttributesExcludedFromTxnEvents(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.HostDisplayName = `my\host\display\name`
		cfg.TransactionEvents.Attributes.Exclude = allAgentAttributeNames
	}

	app := testApp(nil, cfgfn, t)
	w := httptest.NewRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))

	hdr := txn.Header()
	hdr.Set("Content-Type", `text/plain; charset=us-ascii`)
	hdr.Set("Content-Length", `345`)

	txn.WriteHeader(404)
	txn.End()

	userAttributes := map[string]interface{}{}
	agentAttributes := map[string]interface{}{
		ats.HostDisplayName:              `my\host\display\name`,
		ats.ResponseCode:                 `404`,
		ats.ResponseHeadersContentType:   `text/plain; charset=us-ascii`,
		ats.ResponseHeadersContentLength: 345,
		ats.RequestMethod:                "GET",
		ats.RequestAcceptHeader:          "text/plain",
		ats.RequestContentType:           "text/html; charset=utf-8",
		ats.RequestContentLength:         753,
		ats.RequestHeadersHost:           "my_domain.com",
		ats.RequestHeadersUserAgent:      "Mozilla/5.0",
		ats.RequestHeadersReferer:        "http://en.wikipedia.org/zip",
	}
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestAgentAttributesExcludedFromTxnEvents",
		URL:             "/hello",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestQueueTime(t *testing.T) {
	app := testApp(nil, nil, t)
	req, err := http.NewRequest("GET", helloPath+helloQueryParams, nil)
	req.Header.Add("X-Queue-Start", "1465793282.12345")
	if nil != err {
		t.Fatal(err)
	}
	txn := app.StartTransaction("myName", nil, req)
	txn.NoticeError(myError{})
	txn.End()

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "test.myError",
		Caller:  "test.TestQueueTime",
		URL:     "/hello",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "WebTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "test.myError",
		Queuing: true,
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"WebFrontend/QueueTime", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allWeb", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/WebTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/myName",
		Zone:            "F",
		AgentAttributes: nil,
		Queuing:         true,
	}})
}
