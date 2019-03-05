package nrawssdk

import (
	"github.com/aws/aws-sdk-go/aws/request"
	internal "github.com/newrelic/go-agent/_integrations/nrawssdk/internal"
	agentinternal "github.com/newrelic/go-agent/internal"
)

func init() { agentinternal.TrackUsage("integration", "library", "aws-sdk-go") }

func startSegment(req *request.Request) {
	input := internal.StartSegmentInputs{
		HTTPRequest: req.HTTPRequest,
		ServiceName: req.ClientInfo.ServiceName,
		Operation:   req.Operation.Name,
		Region:      req.ClientInfo.SigningRegion,
		Params:      req.Params,
	}
	req.HTTPRequest = internal.StartSegment(input)
}

func endSegment(req *request.Request) {
	ctx := req.HTTPRequest.Context()
	internal.EndSegment(ctx, req.HTTPResponse.Header)
}

// InstrumentHandlers will add instrumentation to the given *request.Handlers.
// A segment will be created for each out going request. The Transaction must
// be added to the request's Context in order for the segment to be recorded.
// For DynamoDB calls, these segments will be Datastore type and for all
// others they will be External type. Additionally, three attributes will be
// added to Transaction Traces and Spans: aws.region, aws.requestId, and
// aws.operation.
//
// To add instrumentation to the Session:
//
//    ses := session.New()
//    nrawssdk.InstrumentHandlers(&ses.Handlers)
//    lambdaClient   = lambda.New(ses, aws.NewConfig())
//
//    req, out := lambdaClient.InvokeRequest(&lambda.InvokeInput{
//        ClientContext:  aws.String("MyApp"),
//        FunctionName:   aws.String("Function"),
//        InvocationType: aws.String("Event"),
//        LogType:        aws.String("Tail"),
//        Payload:        []byte("{}"),
//    }
//    req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)
//    err := req.Send()
//
// To add instrumentation to a Request:
//
//    req, out := lambdaClient.InvokeRequest(&lambda.InvokeInput{
//        ClientContext:  aws.String("MyApp"),
//        FunctionName:   aws.String("Function"),
//        InvocationType: aws.String("Event"),
//        LogType:        aws.String("Tail"),
//        Payload:        []byte("{}"),
//    }
//    nrawssdk.InstrumentHandlers(&req.Handlers)
//    req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)
//    err := req.Send()
func InstrumentHandlers(handlers *request.Handlers) {
	handlers.Send.SetFrontNamed(request.NamedHandler{
		Name: "StartNewRelicSegment",
		Fn:   startSegment,
	})
	handlers.Send.SetBackNamed(request.NamedHandler{
		Name: "EndNewRelicSegment",
		Fn:   endSegment,
	})
}
