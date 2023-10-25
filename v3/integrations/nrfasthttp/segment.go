package nrfasthttp

import (
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

func StartExternalSegment(txn *newrelic.Transaction, ctx *fasthttp.RequestCtx) *newrelic.ExternalSegment {
	var secureAgentEvent any

	if nil == txn {
		txn = transactionFromRequestContext(ctx)
	}
	request := &http.Request{}

	fasthttpadaptor.ConvertRequest(ctx, request, true)
	s := &newrelic.ExternalSegment{
		StartTime: txn.StartSegmentNow(),
		Request:   request,
	}

	if newrelic.IsSecurityAgentPresent() {
		secureAgentEvent = newrelic.GetSecurityAgentInterface().SendEvent("OUTBOUND", request)
		s.SetSecureAgentEvent(secureAgentEvent)
	}

	if request != nil && request.Header != nil {
		for key, values := range s.GetOutboundHeaders() {
			for _, value := range values {
				request.Header.Set(key, value)
			}
		}

		if newrelic.IsSecurityAgentPresent() {
			newrelic.GetSecurityAgentInterface().DistributedTraceHeaders(request, secureAgentEvent)
		}
	}

	return s
}

func FromContext(ctx *fasthttp.RequestCtx) *newrelic.Transaction {
	return transactionFromRequestContext(ctx)
}

func transactionFromRequestContext(ctx *fasthttp.RequestCtx) *newrelic.Transaction {
	if nil != ctx {
		txn := ctx.UserValue("transaction").(*newrelic.Transaction)
		return txn
	}

	return nil
}
