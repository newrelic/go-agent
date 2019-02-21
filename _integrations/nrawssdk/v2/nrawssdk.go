package nrawssdk

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrawssdk/common"
)

func startSegment(req *aws.Request) {
	req.HTTPRequest = common.StartSegment(req.HTTPRequest,
		req.Metadata.ServiceName, req.Operation.Name, req.Params)
}

func endSegment(req *aws.Request) {
	ctx := req.HTTPRequest.Context()
	common.EndSegment(ctx)
}

func InstrumentHandlers(handlers *aws.Handlers) {
	handlers.Validate.SetFrontNamed(aws.NamedHandler{
		Name: "StartNewRelicSegment",
		Fn:   startSegment,
	})
	handlers.Complete.SetBackNamed(aws.NamedHandler{
		Name: "EndNewRelicSegment",
		Fn:   endSegment,
	})
}

func InstrumentConfig(cfg aws.Config) aws.Config {
	InstrumentHandlers(&cfg.Handlers)
	return cfg
}

func InstrumentRequest(req *aws.Request, txn newrelic.Transaction) *aws.Request {
	InstrumentHandlers(&req.Handlers)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)
	return req
}
