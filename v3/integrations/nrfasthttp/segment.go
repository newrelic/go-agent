package nrfasthttp

import (
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

// StartExternalSegment automatically creates and fills out a New Relic external segment for a given
// fasthttp request object. This function will accept either a fasthttp.Request or a fasthttp.RequestContext
// object as the request argument.
func StartExternalSegment(txn *newrelic.Transaction, request any) *newrelic.ExternalSegment {
	var secureAgentEvent any
	var ctx *fasthttp.RequestCtx

	switch reqObject := request.(type) {

	case *fasthttp.RequestCtx:
		ctx = reqObject

	case *fasthttp.Request:
		ctx = &fasthttp.RequestCtx{}
		reqObject.CopyTo(&ctx.Request)

	default:
		return nil
	}

	if nil == txn {
		txn = transactionFromRequestContext(ctx)
	}
	req := &http.Request{}

	fasthttpadaptor.ConvertRequest(ctx, req, true)
	s := &newrelic.ExternalSegment{
		StartTime: txn.StartSegmentNow(),
		Request:   req,
	}

	if newrelic.IsSecurityAgentPresent() {
		secureAgentEvent = newrelic.GetSecurityAgentInterface().SendEvent("OUTBOUND", request)
		s.SetSecureAgentEvent(secureAgentEvent)
	}

	if request != nil && req.Header != nil {
		for key, values := range s.GetOutboundHeaders() {
			for _, value := range values {
				req.Header.Set(key, value)
			}
		}

		if newrelic.IsSecurityAgentPresent() {
			newrelic.GetSecurityAgentInterface().DistributedTraceHeaders(req, secureAgentEvent)
		}

		for k, values := range req.Header {
			for _, value := range values {
				ctx.Request.Header.Set(k, value)
			}
		}
	}

	return s
}

// FromContext extracts a transaction pointer from a fasthttp.RequestContext object
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
