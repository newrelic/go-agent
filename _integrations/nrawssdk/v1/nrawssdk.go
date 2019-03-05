package nrawssdk

import (
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	newrelic "github.com/newrelic/go-agent"
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
// A segment will be created for each out going request. For DynamoDB calls,
// these will be Datastore segments and for all others they will be External
// segments.
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

// InstrumentSession will insert instrumentation to add transaction segments to
// all requests using the given Session. These segments will only appear if the
// Transaction is also added to every request context.
//
//    ses := session.New()
//    ses = nrawssdk.InstrumentSession(ses)
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
func InstrumentSession(s *session.Session) *session.Session {
	InstrumentHandlers(&s.Handlers)
	return s
}

// InstrumentRequest will add transaction segments to the given request and add
// the Transaction to the request's context.
//
//    req, out := lambdaClient.InvokeRequest(&lambda.InvokeInput{
//        ClientContext:  aws.String("MyApp"),
//        FunctionName:   aws.String("Function"),
//        InvocationType: aws.String("Event"),
//        LogType:        aws.String("Tail"),
//        Payload:        []byte("{}"),
//    }
//    req = nrawssdk.InstrumentRequest(req, txn)
//    err := req.Send()
func InstrumentRequest(req *request.Request, txn newrelic.Transaction) *request.Request {
	InstrumentHandlers(&req.Handlers)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)
	return req
}
