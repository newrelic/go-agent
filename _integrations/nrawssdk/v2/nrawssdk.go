package nrawssdk

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	newrelic "github.com/newrelic/go-agent"
	internal "github.com/newrelic/go-agent/_integrations/nrawssdk/internal"
)

func startSegment(req *aws.Request) {
	req.HTTPRequest = internal.StartSegment(req.HTTPRequest,
		req.Metadata.ServiceName, req.Operation.Name, req.Params)
}

func endSegment(req *aws.Request) {
	ctx := req.HTTPRequest.Context()
	internal.EndSegment(ctx)
}

// InstrumentHandlers will add instrumentation to the given *request.Handlers.
// A segment will be created for each out going request. For DynamoDB calls,
// these will be Datastore segments and for all others they will be External
// segments.
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

// InstrumentConfig will insert instrumentation to add transaction segments to
// all requests using the given Config. These segments will only appear if the
// Transaction is also added to every request context.
//
//    cfg, _ := external.LoadDefaultAWSConfig()
//    cfg.Region = endpoints.UsWest2RegionID
//    cfg = nrawssdk.InstrumentConfig(cfg)
//    lambdaClient   = lambda.New(cfg)
//
//    req := lambdaClient.InvokeRequest(&lambda.InvokeInput{
//        ClientContext:  aws.String("MyApp"),
//        FunctionName:   aws.String("Function"),
//        InvocationType: lambda.InvocationTypeEvent,
//        LogType:        lambda.LogTypeTail,
//        Payload:        []byte("{}"),
//    }
//    req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)
//    resp, err := req.Send()
func InstrumentConfig(cfg aws.Config) aws.Config {
	InstrumentHandlers(&cfg.Handlers)
	return cfg
}

// InstrumentRequest will add transaction segments to the given request and add
// the Transaction to the request's context.
//
//    req := lambdaClient.InvokeRequest(&lambda.InvokeInput{
//        ClientContext:  aws.String("MyApp"),
//        FunctionName:   aws.String("Function"),
//        InvocationType: lambda.InvocationTypeEvent,
//        LogType:        lambda.LogTypeTail,
//        Payload:        []byte("{}"),
//    }
//    req = nrawssdk.InstrumentRequest(req, txn)
//    resp, err := req.Send()
func InstrumentRequest(req *aws.Request, txn newrelic.Transaction) *aws.Request {
	InstrumentHandlers(&req.Handlers)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)
	return req
}
