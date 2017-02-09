package newrelic

import (
	"runtime"
	"testing"

	"github.com/newrelic/go-agent/internal"
)

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
		{Name: "OtherTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/OtherTransaction/Go/myName", Scope: "", Forced: true, Data: singleCount},
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
		{Name: "WebTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/myName", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/WebTransaction/Go/myName", Scope: "", Forced: true, Data: singleCount},
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
		{Name: "OtherTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
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
		{Name: "OtherTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/OtherTransaction/Go/myName", Scope: "", Forced: true, Data: singleCount},
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
		{Name: "OtherTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
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
		{Name: "OtherTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
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
		{Name: "OtherTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
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
		{Name: "OtherTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/OtherTransaction/Go/myName", Scope: "", Forced: true, Data: singleCount},
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
		{Name: "OtherTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/OtherTransaction/Go/myName", Scope: "", Forced: true, Data: singleCount},
	})
}

type errorWithClass struct{ class string }

func (e errorWithClass) Error() string      { return "my msg" }
func (e errorWithClass) ErrorClass() string { return e.class }

func TestErrorWithClasser(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	err := txn.NoticeError(errorWithClass{class: "zap"})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "zap",
		Caller:  "go-agent.TestErrorWithClasser",
		URL:     "",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "zap",
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/OtherTransaction/Go/myName", Scope: "", Forced: true, Data: singleCount},
	})
}

func TestErrorWithClasserReturnsEmpty(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	err := txn.NoticeError(errorWithClass{class: ""})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "newrelic.errorWithClass",
		Caller:  "go-agent.TestErrorWithClasserReturnsEmpty",
		URL:     "",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "newrelic.errorWithClass",
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/OtherTransaction/Go/myName", Scope: "", Forced: true, Data: singleCount},
	})
}

type withStackTrace struct{ trace []uintptr }

func makeErrorWithStackTrace() error {
	callers := make([]uintptr, 20)
	written := runtime.Callers(1, callers)
	return withStackTrace{
		trace: callers[0:written],
	}
}

func (e withStackTrace) Error() string         { return "my msg" }
func (e withStackTrace) StackTrace() []uintptr { return e.trace }

func TestErrorWithStackTrace(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	e := makeErrorWithStackTrace()
	err := txn.NoticeError(e)
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "newrelic.withStackTrace",
		Caller:  "go-agent.makeErrorWithStackTrace",
		URL:     "",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "newrelic.withStackTrace",
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/OtherTransaction/Go/myName", Scope: "", Forced: true, Data: singleCount},
	})
}

func TestErrorWithStackTraceReturnsNil(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("myName", nil, nil)
	e := withStackTrace{trace: nil}
	err := txn.NoticeError(e)
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "newrelic.withStackTrace",
		Caller:  "go-agent.TestErrorWithStackTraceReturnsNil",
		URL:     "",
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName: "OtherTransaction/Go/myName",
		Msg:     "my msg",
		Klass:   "newrelic.withStackTrace",
	}})
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/OtherTransaction/Go/myName", Scope: "", Forced: true, Data: singleCount},
	})
}
