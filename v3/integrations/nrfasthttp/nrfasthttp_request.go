package nrfasthttp

import (
	"net/http"

	internal "github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

func init() { internal.TrackUsage("integration", "framework", "fasthttp") }

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

func GetTransaction(ctx *fasthttp.RequestCtx) *newrelic.Transaction {
	txn := ctx.UserValue("transaction")

	if txn == nil {
		return nil
	}

	return txn.(*newrelic.Transaction)
}

func NRHandler(app *newrelic.Application, original fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// Ignore reporting transaction for browser requesting .ico files
		if string(ctx.Path()) == "/favicon.ico" {
			original(ctx)
			return
		}

		txn := app.StartTransaction(string(ctx.Path()))
		defer txn.End()
		ctx.SetUserValue("transaction", txn)

		segRequest := txn.StartSegment("fasthttp-set-request")
		// Set transaction attributes
		txn.AddAttribute("method", string(ctx.Method()))
		txn.AddAttribute("path", string(ctx.Path()))
		// convert fasthttp request to http request
		r := &http.Request{}
		fasthttpadaptor.ConvertRequest(ctx, r, true)

		txn.SetWebRequestHTTP(r)
		txn.InsertDistributedTraceHeaders(r.Header)
		segRequest.End()

		original(ctx)
		// Set Web Response
		seg := txn.StartSegment("fasthttp-set-response")
		resp := fasthttpWrapperResponse{ctx: ctx}
		txn.SetWebResponse(resp)
		seg.End()
	}
}
