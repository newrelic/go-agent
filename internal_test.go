package newrelic

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal"
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

func testApp(replyfn func(*internal.ConnectReply), cfgfn func(*Config), t testing.TB) expectApp {
	cfg := NewConfig("my app", "0123456789012345678901234567890123456789")

	if nil != cfgfn {
		cfgfn(&cfg)
	}

	app, err := newTestApp(replyfn, cfg)
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
	cfgfn := func(cfg *Config) { cfg.HighSecurity = true }
	app := testApp(nil, cfgfn, t)
	err := app.RecordCustomEvent("myType", validParams)
	if err != errHighSecurityEnabled {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantCustomEvent{})
}

func TestRecordCustomEventEventsDisabled(t *testing.T) {
	cfgfn := func(cfg *Config) { cfg.CustomInsightsEvents.Enabled = false }
	app := testApp(nil, cfgfn, t)
	err := app.RecordCustomEvent("myType", validParams)
	if err != errCustomEventsDisabled {
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
	if err != errCustomEventsRemoteDisabled {
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
		Klass:   "newrelic.myError",
		Caller:  "go-agent.TestNoticeErrorBackground",
		URL:     "",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
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
		Klass:   "newrelic.myError",
		Caller:  "go-agent.TestNoticeErrorWeb",
		URL:     "/hello",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "WebTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
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
	if err != errAlreadyEnded {
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
	cfgFn := func(cfg *Config) { cfg.HighSecurity = true }
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("myName", nil, nil)
	err := txn.NoticeError(myError{})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     highSecurityErrorMsg,
		Klass:   "newrelic.myError",
		Caller:  "go-agent.TestNoticeErrorHighSecurity",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     highSecurityErrorMsg,
		Klass:   "newrelic.myError",
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
	cfgFn := func(cfg *Config) { cfg.ErrorCollector.Enabled = false }
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("myName", nil, nil)
	err := txn.NoticeError(myError{})
	if errorsLocallyDisabled != err {
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

func TestNoticeErrorRemotelyDisabled(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) { reply.CollectErrors = false }
	app := testApp(replyfn, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	err := txn.NoticeError(myError{})
	if errorsRemotelyDisabled != err {
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

func TestNoticeErrorNil(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	err := txn.NoticeError(nil)
	if errNilError != err {
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
	cfgFn := func(cfg *Config) { cfg.ErrorCollector.CaptureEvents = false }
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
		Klass:   "newrelic.myError",
		Caller:  "go-agent.TestNoticeErrorEventsLocallyDisabled",
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
		Klass:   "newrelic.myError",
		Caller:  "go-agent.TestNoticeErrorEventsRemotelyDisabled",
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
	cfgFn := func(cfg *Config) { cfg.TransactionEvents.Enabled = false }
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
		Caller:  "go-agent.myErrorHandler",
		URL:     "/hello",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
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
		Caller:  "go-agent.myErrorHandler",
		URL:     "/hello",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
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
	if err := txn.SetName("three"); err != errAlreadyEnded {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/two", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
	})
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
	txn := app.StartTransaction("myName", nil, nil)

	e := myError{}
	r := deferEndPanic(txn, e)
	if r != e {
		t.Error("panic not propagated", r)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   internal.PanicErrorKlass,
		Caller:  "go-agent.(*txn).End",
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
		t.Error("panic not propagated", r)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my string",
		Klass:   internal.PanicErrorKlass,
		Caller:  "go-agent.(*txn).End",
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
		t.Error("panic not propagated", r)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "22",
		Klass:   internal.PanicErrorKlass,
		Caller:  "go-agent.(*txn).End",
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
		Caller:  "go-agent.(*txn).WriteHeader",
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
	w := newCompatibleResponseRecorder()
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
	cfgFn := func(cfg *Config) {
		cfg.ErrorCollector.IgnoreStatusCodes =
			append(cfg.ErrorCollector.IgnoreStatusCodes,
				http.StatusNotFound)
	}
	app := testApp(nil, cfgFn, t)
	w := newCompatibleResponseRecorder()
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
	w := newCompatibleResponseRecorder()
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
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/hello", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/hello", "", false, nil},
	})
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
		Klass:   "newrelic.myError",
		Caller:  "go-agent.TestQueueTime",
		URL:     "/hello",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "WebTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
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

func TestIgnore(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	txn.NoticeError(myError{})
	err := txn.Ignore()
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{})
	app.ExpectMetrics(t, []internal.WantMetric{})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{})
}

func TestIgnoreAlreadyEnded(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	txn.NoticeError(myError{})
	txn.End()
	err := txn.Ignore()
	if err != errAlreadyEnded {
		t.Error(err)
	}
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
		Caller:  "go-agent.TestIgnoreAlreadyEnded",
		URL:     "",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{"OtherTransaction/Go/myName", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
		{"Errors/OtherTransaction/Go/myName", "", true, []float64{1, 0, 0, 0, 0, 0, 0}},
	})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name: "OtherTransaction/Go/myName",
		Zone: "",
	}})
}

func TestResponseCodeIsError(t *testing.T) {
	cfg := NewConfig("my app", "0123456789012345678901234567890123456789")

	if is := responseCodeIsError(&cfg, 200); is {
		t.Error(is)
	}
	if is := responseCodeIsError(&cfg, 400); !is {
		t.Error(is)
	}
	if is := responseCodeIsError(&cfg, 404); is {
		t.Error(is)
	}
	if is := responseCodeIsError(&cfg, 503); !is {
		t.Error(is)
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
	host := internal.HostFromURL(externalSegmentURL(ExternalSegment{}))
	if "" != host {
		t.Error(host)
	}
	// segment only containing url
	host = internal.HostFromURL(externalSegmentURL(ExternalSegment{URL: rawURL}))
	if "url.com" != host {
		t.Error(host)
	}
	// segment only containing request
	host = internal.HostFromURL(externalSegmentURL(ExternalSegment{Request: req}))
	if "request.com" != host {
		t.Error(host)
	}
	// segment only containing response
	host = internal.HostFromURL(externalSegmentURL(ExternalSegment{Response: response}))
	if "response.com" != host {
		t.Error(host)
	}
	// segment containing request and response
	host = internal.HostFromURL(externalSegmentURL(ExternalSegment{
		Request:  req,
		Response: response,
	}))
	if "response.com" != host {
		t.Error(host)
	}
	// segment containing url, request, and response
	host = internal.HostFromURL(externalSegmentURL(ExternalSegment{
		URL:      rawURL,
		Request:  req,
		Response: response,
	}))
	if "url.com" != host {
		t.Error(host)
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

func TestTraceSegment(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	func() {
		defer StartSegment(txn, "segment").End()
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
		defer StartSegment(txn, "segment").End()
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
	s := StartSegment(txn, "segment")
	txn.End()
	s.End()

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

func TestTraceSegmentNilTxn(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s := Segment{Name: "hello"}
	s.End()
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
		s := DatastoreSegment{}
		s.StartTime = txn.StartSegmentNow()
		s.Product = DatastoreMySQL
		s.Collection = "my_table"
		s.Operation = "SELECT"
		defer s.End()
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
		{"Datastore/instance/MySQL/unknown/unknown", "", false, nil},
	})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:            "WebTransaction/Go/myName",
		Msg:                "my msg",
		Klass:              "newrelic.myError",
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
		defer DatastoreSegment{
			StartTime:  txn.StartSegmentNow(),
			Product:    DatastoreMySQL,
			Collection: "my_table",
			Operation:  "SELECT",
		}.End()
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
		{"Datastore/instance/MySQL/unknown/unknown", "", false, nil},
	})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:            "OtherTransaction/Go/myName",
		Msg:                "my msg",
		Klass:              "newrelic.myError",
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
		defer DatastoreSegment{
			StartTime: txn.StartSegmentNow(),
		}.End()
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
		{"Datastore/instance/Unknown/unknown/unknown", "", false, nil},
	})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:            "WebTransaction/Go/myName",
		Msg:                "my msg",
		Klass:              "newrelic.myError",
		DatastoreCallCount: 1,
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:               "WebTransaction/Go/myName",
		Zone:               "F",
		DatastoreCallCount: 1,
	}})
}

func TestTraceDatastoreNilTxn(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	var s DatastoreSegment
	s.Product = DatastoreMySQL
	s.Collection = "my_table"
	s.Operation = "SELECT"
	s.End()
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
		Klass:   "newrelic.myError",
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
	s := DatastoreSegment{
		StartTime:  txn.StartSegmentNow(),
		Product:    DatastoreMySQL,
		Collection: "my_table",
		Operation:  "SELECT",
	}
	txn.End()
	s.End()

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
		Klass:   "newrelic.myError",
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
		defer ExternalSegment{
			StartTime: txn.StartSegmentNow(),
			URL:       "http://example.com/",
		}.End()
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
		Klass:             "newrelic.myError",
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
		defer ExternalSegment{
			StartTime: txn.StartSegmentNow(),
			URL:       "http://example.com/",
		}.End()
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
		Klass:             "newrelic.myError",
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
		defer ExternalSegment{
			StartTime: txn.StartSegmentNow(),
		}.End()
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
		Klass:             "newrelic.myError",
		ExternalCallCount: 1,
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:              "WebTransaction/Go/myName",
		Zone:              "F",
		ExternalCallCount: 1,
	}})
}

func TestTraceExternalNilTxn(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	txn.NoticeError(myError{})
	var s ExternalSegment
	s.End()
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
		Klass:   "newrelic.myError",
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
	s := ExternalSegment{
		StartTime: txn.StartSegmentNow(),
		URL:       "http://example.com/",
	}
	txn.End()
	s.End()

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
		Klass:   "newrelic.myError",
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name: "WebTransaction/Go/myName",
		Zone: "F",
	}})
}

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
	client.Transport = NewRoundTripper(txn, inner)
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
		Klass:             "newrelic.myError",
		ExternalCallCount: 1,
	}})
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:              "OtherTransaction/Go/myName",
		Zone:              "",
		ExternalCallCount: 1,
	}})
}

func TestTraceBelowThreshold(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{})
}

func TestTraceBelowThresholdBackground(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
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
	txn := app.StartTransaction("myName", nil, helloRequest)
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName:  "WebTransaction/Go/myName",
		CleanURL:    "/hello",
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
	txn := app.StartTransaction("myName", nil, helloRequest)
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
	txn := app.StartTransaction("myName", nil, helloRequest)
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
	txn := app.StartTransaction("myName", nil, helloRequest)
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
		MetricName:  "WebTransaction/Go/myName",
		CleanURL:    "/hello",
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
	txn := app.StartTransaction("myName", nil, helloRequest)
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
		MetricName:  "WebTransaction/Go/myName",
		CleanURL:    "/hello",
		NumSegments: 0,
	}})
}
