package newrelic

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	ats "github.com/newrelic/go-agent/attributes"
	"github.com/newrelic/go-agent/datastore"
	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/utilization"
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
	app, err := newAppInternal(cfg)
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
	if errorsRemotelyDisabled != err {
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

func TestHostFromRequestResponse(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	host := hostFromRequestResponse(req, &http.Response{Request: req})
	if host != "example.com" {
		t.Error("normal usage", host)
	}
	host = hostFromRequestResponse(nil, &http.Response{Request: req})
	if host != "example.com" {
		t.Error("missing request", host)
	}
	host = hostFromRequestResponse(req, nil)
	if host != "example.com" {
		t.Error("missing response", host)
	}
	host = hostFromRequestResponse(nil, nil)
	if host != "" {
		t.Error("missing request and response", host)
	}
	req.URL = nil
	host = hostFromRequestResponse(req, nil)
	if host != "" {
		t.Error("missing URL", host)
	}
}

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
	var token Token
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
		Klass:              "newrelic.myError",
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
		Klass:             "newrelic.myError",
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

func BenchmarkTraceSegmentWithDefer(b *testing.B) {
	cfg := NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	app, err := newAppInternal(cfg)
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
	cfg := NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	app, err := newAppInternal(cfg)
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
	cfg := NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	app, err := newAppInternal(cfg)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn", nil, nil)
	fn := func(txn Transaction) {
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
	cfg := NewConfig("my app", sampleLicense)
	cfg.Enabled = false
	app, err := newAppInternal(cfg)
	if nil != err {
		b.Fatal(err)
	}
	txn := app.StartTransaction("my txn", nil, nil)
	fn := func(txn Transaction) {
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
	if err := txn.AddAttribute("already_ended", "zap"); err != errAlreadyEnded {
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
		Caller:          "go-agent.TestUserAttributeBasics",
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
	cfgfn := func(cfg *Config) {
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
		Caller:          "go-agent.TestUserAttributeConfiguration",
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
	cfgfn := func(cfg *Config) {
		cfg.HostDisplayName = `my\host\display\name`
	}

	app := testApp(nil, cfgfn, t)
	w := newCompatibleResponseRecorder()
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
		Caller:          "go-agent.TestAgentAttributes",
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
	cfgfn := func(cfg *Config) {
		cfg.Attributes.Enabled = false
		cfg.HostDisplayName = `my\host\display\name`
	}

	app := testApp(nil, cfgfn, t)
	w := newCompatibleResponseRecorder()
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
		Caller:          "go-agent.TestAttributesDisabled",
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
	w := newCompatibleResponseRecorder()
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
		Caller:          "go-agent.TestDefaultResponseCode",
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
	cfgfn := func(cfg *Config) {
		cfg.TransactionEvents.Attributes.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	w := newCompatibleResponseRecorder()
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
		Caller:          "go-agent.TestTxnEventAttributesDisabled",
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
	cfgfn := func(cfg *Config) {
		cfg.ErrorCollector.Attributes.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	w := newCompatibleResponseRecorder()
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
		Caller:          "go-agent.TestErrorAttributesDisabled",
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
	cfgfn := func(cfg *Config) {
		cfg.HostDisplayName = `my\host\display\name`
		cfg.Attributes.Exclude = allAgentAttributeNames
	}

	app := testApp(nil, cfgfn, t)
	w := newCompatibleResponseRecorder()
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
		Caller:          "go-agent.TestAgentAttributesExcluded",
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
	cfgfn := func(cfg *Config) {
		cfg.HostDisplayName = `my\host\display\name`
		cfg.ErrorCollector.Attributes.Exclude = allAgentAttributeNames
	}

	app := testApp(nil, cfgfn, t)
	w := newCompatibleResponseRecorder()
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
		Caller:          "go-agent.TestAgentAttributesExcludedFromErrors",
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
	cfgfn := func(cfg *Config) {
		cfg.HostDisplayName = `my\host\display\name`
		cfg.TransactionEvents.Attributes.Exclude = allAgentAttributeNames
	}

	app := testApp(nil, cfgfn, t)
	w := newCompatibleResponseRecorder()
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
		Caller:          "go-agent.TestAgentAttributesExcludedFromTxnEvents",
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

func TestCopyConfigReferenceFieldsPresent(t *testing.T) {
	cfg := NewConfig("my appname", "0123456789012345678901234567890123456789")
	cfg.Labels["zip"] = "zap"
	cfg.ErrorCollector.IgnoreStatusCodes = append(cfg.ErrorCollector.IgnoreStatusCodes, 405)
	cfg.Attributes.Include = append(cfg.Attributes.Include, "1")
	cfg.Attributes.Exclude = append(cfg.Attributes.Exclude, "2")
	cfg.TransactionEvents.Attributes.Include = append(cfg.TransactionEvents.Attributes.Include, "3")
	cfg.TransactionEvents.Attributes.Exclude = append(cfg.TransactionEvents.Attributes.Exclude, "4")
	cfg.ErrorCollector.Attributes.Include = append(cfg.ErrorCollector.Attributes.Include, "5")
	cfg.ErrorCollector.Attributes.Exclude = append(cfg.ErrorCollector.Attributes.Exclude, "6")
	cfg.Transport = &http.Transport{}
	cfg.Logger = NewLogger(os.Stdout)

	cp := copyConfigReferenceFields(cfg)

	cfg.Labels["zop"] = "zup"
	cfg.ErrorCollector.IgnoreStatusCodes[0] = 201
	cfg.Attributes.Include[0] = "zap"
	cfg.Attributes.Exclude[0] = "zap"
	cfg.TransactionEvents.Attributes.Include[0] = "zap"
	cfg.TransactionEvents.Attributes.Exclude[0] = "zap"
	cfg.ErrorCollector.Attributes.Include[0] = "zap"
	cfg.ErrorCollector.Attributes.Exclude[0] = "zap"

	expect := internal.CompactJSONString(`[
	{
		"pid":123,
		"language":"go",
		"agent_version":"0.2.2",
		"host":"my-hostname",
		"settings":{
			"AppName":"my appname",
			"Attributes":{"Enabled":true,"Exclude":["2"],"Include":["1"]},
			"BetaToken":"",
			"CustomInsightsEvents":{"Enabled":true},
			"Enabled":true,
			"ErrorCollector":{
				"Attributes":{"Enabled":true,"Exclude":["6"],"Include":["5"]},
				"CaptureEvents":true,
				"Enabled":true,
				"IgnoreStatusCodes":[404,405]
			},
			"HighSecurity":false,
			"HostDisplayName":"",
			"Labels":{"zip":"zap"},
			"Logger":"*logger.logFile",
			"RuntimeSampler":{"Enabled":true},
			"TransactionEvents":{
				"Attributes":{"Enabled":true,"Exclude":["4"],"Include":["3"]},
				"Enabled":true
			},
			"Transport":"*http.Transport",
			"UseTLS":true,
			"Utilization":{"DetectAWS":true,"DetectDocker":true}
		},
		"app_name":["my appname"],
		"high_security":false,
		"labels":[{"label_type":"zip","label_value":"zap"}],
		"environment":[["Compiler","comp"],["GOARCH","arch"],["GOOS","goos"],["Version","vers"]],
		"identifier":"my appname",
		"utilization":{
			"metadata_version":1,
			"logical_processors":16,
			"total_ram_mib":1024,
			"hostname":"my-hostname"
		}
	}]`)

	js, err := configConnectJSONInternal(cp, 123, &utilization.SampleData, internal.SampleEnvironment, "0.2.2")
	if nil != err {
		t.Fatal(err)
	}
	if string(js) != expect {
		t.Error(string(js))
	}
}

func TestCopyConfigReferenceFieldsAbsent(t *testing.T) {
	cfg := NewConfig("my appname", "0123456789012345678901234567890123456789")
	cfg.Labels = nil
	cfg.ErrorCollector.IgnoreStatusCodes = nil

	cp := copyConfigReferenceFields(cfg)

	expect := internal.CompactJSONString(`[
	{
		"pid":123,
		"language":"go",
		"agent_version":"0.2.2",
		"host":"my-hostname",
		"settings":{
			"AppName":"my appname",
			"Attributes":{"Enabled":true,"Exclude":null,"Include":null},
			"BetaToken":"",
			"CustomInsightsEvents":{"Enabled":true},
			"Enabled":true,
			"ErrorCollector":{
				"Attributes":{"Enabled":true,"Exclude":null,"Include":null},
				"CaptureEvents":true,
				"Enabled":true,
				"IgnoreStatusCodes":null
			},
			"HighSecurity":false,
			"HostDisplayName":"",
			"Labels":null,
			"Logger":null,
			"RuntimeSampler":{"Enabled":true},
			"TransactionEvents":{
				"Attributes":{"Enabled":true,"Exclude":null,"Include":null},
				"Enabled":true
			},
			"Transport":null,
			"UseTLS":true,
			"Utilization":{"DetectAWS":true,"DetectDocker":true}
		},
		"app_name":["my appname"],
		"high_security":false,
		"environment":[["Compiler","comp"],["GOARCH","arch"],["GOOS","goos"],["Version","vers"]],
		"identifier":"my appname",
		"utilization":{
			"metadata_version":1,
			"logical_processors":16,
			"total_ram_mib":1024,
			"hostname":"my-hostname"
		}
	}]`)

	js, err := configConnectJSONInternal(cp, 123, &utilization.SampleData, internal.SampleEnvironment, "0.2.2")
	if nil != err {
		t.Fatal(err)
	}
	if string(js) != expect {
		t.Error(string(js))
	}
}

func TestValidate(t *testing.T) {
	c := Config{
		License: "0123456789012345678901234567890123456789",
		AppName: "my app",
		Enabled: true,
	}
	if err := c.Validate(); nil != err {
		t.Error(err)
	}
	c = Config{
		License: "",
		AppName: "my app",
		Enabled: true,
	}
	if err := c.Validate(); err != errLicenseLen {
		t.Error(err)
	}
	c = Config{
		License: "",
		AppName: "my app",
		Enabled: false,
	}
	if err := c.Validate(); nil != err {
		t.Error(err)
	}
	c = Config{
		License: "wronglength",
		AppName: "my app",
		Enabled: true,
	}
	if err := c.Validate(); err != errLicenseLen {
		t.Error(err)
	}
	c = Config{
		License: "0123456789012345678901234567890123456789",
		AppName: "too;many;app;names",
		Enabled: true,
	}
	if err := c.Validate(); err != errAppNameLimit {
		t.Error(err)
	}
	c = Config{
		License: "0123456789012345678901234567890123456789",
		AppName: "",
		Enabled: true,
	}
	if err := c.Validate(); err != errAppNameMissing {
		t.Error(err)
	}
	c = Config{
		License:      "0123456789012345678901234567890123456789",
		AppName:      "my app",
		Enabled:      true,
		HighSecurity: true,
	}
	if err := c.Validate(); err != errHighSecurityTLS {
		t.Error(err)
	}
	c = Config{
		License:      "0123456789012345678901234567890123456789",
		AppName:      "my app",
		Enabled:      true,
		UseTLS:       true,
		HighSecurity: true,
	}
	if err := c.Validate(); err != nil {
		t.Error(err)
	}
}
