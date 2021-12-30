// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrawssdk instruments https://github.com/aws/aws-sdk-go requests.
package nrawssdk

import (
	"net/http"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/awssupport"
)

func init() { internal.TrackUsage("integration", "library", "aws-sdk-go") }

func startSegment(req *request.Request) {
	input := awssupport.StartSegmentInputs{
		HTTPRequest: req.HTTPRequest,
		ServiceName: req.ClientInfo.ServiceName,
		Operation:   req.Operation.Name,
		Region:      req.ClientInfo.SigningRegion,
		Params:      req.Params,
	}
	req.HTTPRequest = awssupport.StartSegment(input)
}

func endSegment(req *request.Request) {
	ctx := req.HTTPRequest.Context()

	hdr := http.Header{}
	if req.HTTPRequest != nil {
		hdr = req.HTTPRequest.Header
	}

	awssupport.EndSegment(ctx, hdr)
}

// InstrumentHandlers will add instrumentation to the given *request.Handlers.
//
// A Segment will be created for each out going request. The Transaction must
// be added to the `http.Request`'s Context in order for the segment to be
// recorded.  For DynamoDB calls, these segments will be
// `newrelic.DatastoreSegment` type and for all others they will be
// `newrelic.ExternalSegment` type.
//
// Additional attributes will be added to Transaction Trace Segments and Span
// Events: aws.region, aws.requestId, and aws.operation.
//
// To add instrumentation to the Session and see segments created for each
// invocation that uses the Session, call InstrumentHandlers with the session's
// Handlers and add the current Transaction to the `http.Request`'s Context:
//
//    ses := session.New()
//    // Add instrumentation to handlers
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
//    // Add txn to http.Request's context
//    req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)
//    err := req.Send()
//
// To add instrumentation to a Request and see a segment created just for the
// individual request, call InstrumentHandlers with the `request.Request`'s
// Handlers and add the current Transaction to the `http.Request`'s Context:
//
//    req, out := lambdaClient.InvokeRequest(&lambda.InvokeInput{
//        ClientContext:  aws.String("MyApp"),
//        FunctionName:   aws.String("Function"),
//        InvocationType: aws.String("Event"),
//        LogType:        aws.String("Tail"),
//        Payload:        []byte("{}"),
//    }
//    // Add instrumentation to handlers
//    nrawssdk.InstrumentHandlers(&req.Handlers)
//    // Add txn to http.Request's context
//    req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)
//    err := req.Send()
func InstrumentHandlers(handlers *request.Handlers) {
	handlers.Sign.SetFrontNamed(request.NamedHandler{
		Name: "StartNewRelicSegment",
		Fn:   startSegment,
	})
	handlers.Send.SetBackNamed(request.NamedHandler{
		Name: "EndNewRelicSegment",
		Fn:   endSegment,
	})
}
