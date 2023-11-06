package nrfasthttp

import (
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/valyala/fasthttp"
)

func createTestApp(dt bool) integrationsupport.ExpectApp {
	return integrationsupport.NewTestApp(replyFn, integrationsupport.ConfigFullTraces, newrelic.ConfigDistributedTracerEnabled(dt))
}

var replyFn = func(reply *internal.ConnectReply) {
	reply.SetSampleEverything()
}

func TestExternalSegment(t *testing.T) {
	app := createTestApp(false)
	txn := app.StartTransaction("myTxn")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	ctx := &fasthttp.RequestCtx{Request: fasthttp.Request{}}
	ctx.Request.SetRequestURI("http://localhost:8080/hello")
	ctx.Request.Header.SetMethod("GET")

	seg := StartExternalSegment(txn, ctx)
	defer seg.End()

	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/myTxn", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/myTxn", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
	})
}

func TestExternalSegmentRequest(t *testing.T) {
	app := createTestApp(false)
	txn := app.StartTransaction("myTxn")

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI("http://localhost:8080/hello")
	req.Header.SetMethod("GET")

	seg := StartExternalSegment(txn, req)
	defer seg.End()

	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/myTxn", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/myTxn", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
	})
}
