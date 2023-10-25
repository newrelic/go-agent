package nrfasthttp

import (
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

type fasthttpWrapperResponse struct {
	ctx *fasthttp.RequestCtx
}

func (rw fasthttpWrapperResponse) Header() http.Header {
	hdrs := http.Header{}
	rw.ctx.Request.Header.VisitAll(func(key, value []byte) {
		hdrs.Add(string(key), string(value))
	})
	return hdrs
}

func (rw fasthttpWrapperResponse) Write(b []byte) (int, error) {
	return rw.ctx.Write(b)
}

func (rw fasthttpWrapperResponse) WriteHeader(code int) {
	rw.ctx.SetStatusCode(code)
}

func WrapHandleFunc(app *newrelic.Application, pattern string, handler func(*fasthttp.RequestCtx), options ...newrelic.TraceOption) (string, func(*fasthttp.RequestCtx)) {
	// add the wrapped function to the trace options as the source code reference point
	// (to the beginning of the option list, so that the user can override this)

	p, h := WrapHandle(app, pattern, fasthttp.RequestHandler(handler), options...)
	return p, func(ctx *fasthttp.RequestCtx) { h(ctx) }
}

func WrapHandle(app *newrelic.Application, pattern string, handler fasthttp.RequestHandler, options ...newrelic.TraceOption) (string, fasthttp.RequestHandler) {
	if app == nil {
		return pattern, handler
	}

	// add the wrapped function to the trace options as the source code reference point
	// (but only if we know we're collecting CLM for this transaction and the user didn't already
	// specify a different code location explicitly).
	return pattern, func(ctx *fasthttp.RequestCtx) {
		cache := newrelic.NewCachedCodeLocation()
		txnOptionList := newrelic.AddCodeLevelMetricsTraceOptions(app, options, cache, handler)
		method := string(ctx.Method())
		path := string(ctx.Path())
		txn := app.StartTransaction(method+" "+path, txnOptionList...)
		ctx.SetUserValue("transaction", txn)
		defer txn.End()
		r := &http.Request{}
		fasthttpadaptor.ConvertRequest(ctx, r, true)
		resp := fasthttpWrapperResponse{ctx: ctx}

		txn.SetWebResponse(resp)
		txn.SetWebRequestHTTP(r)

		//		r = newrelic.RequestWithTransactionContext(r, txn)

		handler(ctx)
	}
}
