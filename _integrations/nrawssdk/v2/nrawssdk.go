// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrawssdk instruments https://github.com/aws/aws-sdk-go-v2 requests.
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
// To add instrumentation to a Config and see segments created for each
// invocation that uses that Config, call InstrumentHandlers with the config's
// Handlers and add the current Transaction to the `http.Request`'s Context:
//
//    cfg, _ := external.LoadDefaultAWSConfig()
//    cfg.Region = "us-west-2"
//    // Add instrumentation to handlers
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
//    // Add txn to http.Request's context
//    ctx := newrelic.NewContext(req.Context(), txn)
//    resp, err := req.Send(ctx)
//
// To add instrumentation to a Request and see a segment created just for the
// individual request, call InstrumentHandlers with the `aws.Request`'s
// Handlers and add the current Transaction to the `http.Request`'s Context:
//
//    req := lambdaClient.InvokeRequest(&lambda.InvokeInput{
//        ClientContext:  aws.String("MyApp"),
//        FunctionName:   aws.String("Function"),
//        InvocationType: lambda.InvocationTypeEvent,
//        LogType:        lambda.LogTypeTail,
//        Payload:        []byte("{}"),
//    }
//    // Add instrumentation to handlers
//    nrawssdk.InstrumentHandlers(&req.Handlers)
//    // Add txn to http.Request's context
//    ctx := newrelic.NewContext(req.Context(), txn)
//    resp, err := req.Send(ctx)
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
