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

	switch request.(type) {
	// preserve prior functionality
	case *fasthttp.RequestCtx:
		ctx := request.(*fasthttp.RequestCtx)

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

			for k, values := range request.Header {
				for _, value := range values {
					ctx.Request.Header.Set(k, value)
				}
			}
		}

		return s

	case *fasthttp.Request:
		req := request.(*fasthttp.Request)
		request := &http.Request{}

		// it is ok to copy req here because we are not using its methods, just copying it into an http object
		// for data collection
		fasthttpadaptor.ConvertRequest(&fasthttp.RequestCtx{Request: *req}, request, true)
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

			for k, values := range request.Header {
				for _, value := range values {
					req.Header.Set(k, value)
				}
			}
		}

		return s
	}

	return nil
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
