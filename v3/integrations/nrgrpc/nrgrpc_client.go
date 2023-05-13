// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrgrpc

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/newrelic/go-agent/v3/newrelic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func getURL(method, target string) *url.URL {
	var host string
	// target can be anything from
	// https://github.com/grpc/grpc/blob/master/doc/naming.md
	// see https://godoc.org/google.golang.org/grpc#DialContext
	if strings.HasPrefix(target, "unix:") {
		host = "localhost"
	} else {
		host = strings.TrimPrefix(target, "dns:///")
	}
	return &url.URL{
		Scheme: "grpc",
		Host:   host,
		Path:   method,
	}
}

func getDummyRequest(method, target string) (request *http.Request) {
	request = &http.Request{}
	request.URL = getURL(method, target)
	request.Header = http.Header{}
	return request
}

// startClientSegment starts an ExternalSegment and adds Distributed Trace
// headers to the outgoing grpc metadata in the context.
func startClientSegment(ctx context.Context, method, target string) (*newrelic.ExternalSegment, context.Context) {
	var seg *newrelic.ExternalSegment
	var req *http.Request

	if txn := newrelic.FromContext(ctx); txn != nil {
		if newrelic.IsSecurityAgentPresent() {
			req = getDummyRequest(method, target)
		}
		seg = newrelic.StartExternalSegment(txn, req)

		method = strings.TrimPrefix(method, "/")
		seg.Host = getURL(method, target).Host
		seg.Library = "gRPC"
		seg.Procedure = method

		hdrs := http.Header{}
		txn.InsertDistributedTraceHeaders(hdrs)
		if len(hdrs) > 0 {
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				md = metadata.New(nil)
			}
			for k := range hdrs {
				if v := hdrs.Get(k); v != "" {
					md.Set(k, v)
				}
			}
			if newrelic.IsSecurityAgentPresent() {
				for k := range req.Header {
					if v := req.Header.Get(k); v != "" {
						md.Set(k, v)
					}
				}
			}
			ctx = metadata.NewOutgoingContext(ctx, md)
		}
	}

	return seg, ctx
}

// UnaryClientInterceptor instruments client unary RPCs.  This interceptor
// records each unary call with an external segment.  Using it requires two steps:
//
// 1. Use this function with grpc.WithChainUnaryInterceptor or
// grpc.WithUnaryInterceptor when creating a grpc.ClientConn.  Example:
//
//	conn, err := grpc.Dial(
//		"localhost:8080",
//		grpc.WithUnaryInterceptor(nrgrpc.UnaryClientInterceptor),
//		grpc.WithStreamInterceptor(nrgrpc.StreamClientInterceptor),
//	)
//
// 2. Ensure that calls made with this grpc.ClientConn are done with a context
// which contains a newrelic.Transaction.
//
// Full example:
// https://github.com/newrelic/go-agent/blob/master/v3/integrations/nrgrpc/example/client/client.go
//
// This interceptor only instruments unary calls.  You must use both
// UnaryClientInterceptor and StreamClientInterceptor to instrument unary and
// streaming calls.  These interceptors add headers to the call metadata if
// distributed tracing is enabled.
func UnaryClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	seg, ctx := startClientSegment(ctx, method, cc.Target())
	defer seg.End()
	return invoker(ctx, method, req, reply, cc, opts...)
}

type wrappedClientStream struct {
	grpc.ClientStream
	segment       *newrelic.ExternalSegment
	isUnaryServer bool
}

func (s wrappedClientStream) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if err == io.EOF || s.isUnaryServer {
		s.segment.End()
	}
	return err
}

// StreamClientInterceptor instruments client streaming RPCs.  This interceptor
// records streaming each call with an external segment.  Using it requires two steps:
//
// 1. Use this function with grpc.WithChainStreamInterceptor or
// grpc.WithStreamInterceptor when creating a grpc.ClientConn.  Example:
//
//	conn, err := grpc.Dial(
//		"localhost:8080",
//		grpc.WithUnaryInterceptor(nrgrpc.UnaryClientInterceptor),
//		grpc.WithStreamInterceptor(nrgrpc.StreamClientInterceptor),
//	)
//
// 2. Ensure that calls made with this grpc.ClientConn are done with a context
// which contains a newrelic.Transaction.
//
// Full example:
// https://github.com/newrelic/go-agent/blob/master/v3/integrations/nrgrpc/example/client/client.go
//
// This interceptor only instruments streaming calls.  You must use both
// UnaryClientInterceptor and StreamClientInterceptor to instrument unary and
// streaming calls.  These interceptors add headers to the call metadata if
// distributed tracing is enabled.
func StreamClientInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	seg, ctx := startClientSegment(ctx, method, cc.Target())
	s, err := streamer(ctx, desc, cc, method, opts...)
	if err != nil {
		return s, err
	}
	return wrappedClientStream{
		segment:       seg,
		ClientStream:  s,
		isUnaryServer: !desc.ServerStreams,
	}, nil
}
