package nrawssdk

import (
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrawssdk/common"
)

func startSegment(req *request.Request) {
	req.HTTPRequest = common.StartSegment(req.HTTPRequest,
		req.ClientInfo.ServiceName, req.Operation.Name, req.Params)
}

func endSegment(req *request.Request) {
	ctx := req.HTTPRequest.Context()
	common.EndSegment(ctx)
}

func InstrumentHandlers(handlers *request.Handlers) {
	handlers.Validate.SetFrontNamed(request.NamedHandler{
		Name: "StartNewRelicSegment",
		Fn:   startSegment,
	})
	handlers.Complete.SetBackNamed(request.NamedHandler{
		Name: "EndNewRelicSegment",
		Fn:   endSegment,
	})
}

func InstrumentSession(s *session.Session) *session.Session {
	InstrumentHandlers(&s.Handlers)
	return s
}

func InstrumentRequest(req *request.Request, txn newrelic.Transaction) *request.Request {
	InstrumentHandlers(&req.Handlers)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)
	return req
}
