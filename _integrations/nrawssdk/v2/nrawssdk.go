package nrawssdk

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	internal "github.com/newrelic/go-agent/_integrations/nrawssdk/internal"
	agentinternal "github.com/newrelic/go-agent/internal"
)

func init() { agentinternal.TrackUsage("integration", "library", "aws-sdk-go-v2") }

func startSegment(req *aws.Request) {
	input := internal.StartSegmentInputs{
		HTTPRequest: req.HTTPRequest,
		ServiceName: req.Metadata.ServiceName,
		Operation:   req.Operation.Name,
		Region:      req.Metadata.SigningRegion,
		Params:      req.Params,
	}
	req.HTTPRequest = internal.StartSegment(input)
}

func endSegment(req *aws.Request) {
	ctx := req.HTTPRequest.Context()
	internal.EndSegment(ctx, req.HTTPResponse.Header)
}

// InstrumentHandlers will add instrumentation to the given *aws.Handlers.
// A segment will be created for each out going request. The Transaction must
// be added to the request's Context in order for the segment to be recorded.
// For DynamoDB calls, these segments will be Datastore type and for all
// others they will be External type. Additionally, three attributes will be
// added to Transaction Traces and Spans: aws.region, aws.requestId, and
// aws.operation.
//
// To add instrumentation to a Config:
//
//    cfg, _ := external.LoadDefaultAWSConfig()
//    cfg.Region = endpoints.UsWest2RegionID
//    nrawssdk.InstrumentHandlers(&cfg.Handlers)
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
//
// To add instrumentation to a Request:
//
//    req := lambdaClient.InvokeRequest(&lambda.InvokeInput{
//        ClientContext:  aws.String("MyApp"),
//        FunctionName:   aws.String("Function"),
//        InvocationType: lambda.InvocationTypeEvent,
//        LogType:        lambda.LogTypeTail,
//        Payload:        []byte("{}"),
//    }
//    nrawssdk.InstrumentHandlers(&req.Handlers)
//    req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)
//    resp, err := req.Send()
func InstrumentHandlers(handlers *aws.Handlers) {
	handlers.Send.SetFrontNamed(aws.NamedHandler{
		Name: "StartNewRelicSegment",
		Fn:   startSegment,
	})
	handlers.Send.SetBackNamed(aws.NamedHandler{
		Name: "EndNewRelicSegment",
		Fn:   endSegment,
	})
}
