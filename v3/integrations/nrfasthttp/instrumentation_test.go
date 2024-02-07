package nrfasthttp

import (
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/valyala/fasthttp"
)

type myError struct{}

func (e myError) Error() string { return "my msg" }

func myErrorHandlerFastHTTP(ctx *fasthttp.RequestCtx) {
	ctx.WriteString("noticing an error")
	txn := ctx.UserValue("transaction").(*newrelic.Transaction)
	txn.NoticeError(myError{})
}

func TestWrapHandleFastHTTPFunc(t *testing.T) {
	singleCount := []float64{1, 0, 0, 0, 0, 0, 0}
	app := createTestApp(true)

	_, wrappedHandler := WrapHandleFunc(app.Application, "/hello", myErrorHandlerFastHTTP)

	if wrappedHandler == nil {
		t.Error("Error when creating a wrapped handler")
	}
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/hello")
	wrappedHandler(ctx)
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/GET /hello",
		Msg:     "my msg",
		Klass:   "nrfasthttp.myError",
	}})

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/GET /hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/GET /hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/GET /hello", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/WebTransaction/Go/GET /hello", Scope: "", Forced: true, Data: singleCount},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: nil},
	})
}
