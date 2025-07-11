// Copyright 2024 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package testapp

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"connectrpc.com/connect"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// Server is a Connect RPC server.
type Server struct{}

// DoUnaryUnary is a unary request, unary response method.
func (s *Server) DoUnaryUnary(ctx context.Context, req *connect.Request[Message]) (*connect.Response[Message], error) {
	defer newrelic.FromContext(ctx).StartSegment("DoUnaryUnary").End()
	headers := req.Header()
	js, _ := json.Marshal(headers)
	response := connect.NewResponse(&Message{Text: string(js)})
	return response, nil
}

// DoUnaryStream is a unary request, stream response method.
func (s *Server) DoUnaryStream(ctx context.Context, req *connect.Request[Message], stream *connect.ServerStream[Message]) error {
	defer newrelic.FromContext(ctx).StartSegment("DoUnaryStream").End()
	headers := req.Header()
	js, _ := json.Marshal(headers)
	for i := 0; i < 3; i++ {
		if err := stream.Send(&Message{Text: string(js)}); err != nil {
			return err
		}
	}
	return nil
}

// DoStreamUnary is a stream request, unary response method.
func (s *Server) DoStreamUnary(ctx context.Context, stream *connect.ClientStream[Message]) (*connect.Response[Message], error) {
	defer newrelic.FromContext(ctx).StartSegment("DoStreamUnary").End()
	headers := stream.RequestHeader()
	js, _ := json.Marshal(headers)
	for stream.Receive() {
	}
	err := stream.Err()
	if errors.Is(err, io.EOF) {
		response := connect.NewResponse(&Message{Text: string(js)})
		return response, nil
	}
	return nil, err
}

// DoStreamStream is a stream request, stream response method.
func (s *Server) DoStreamStream(ctx context.Context, stream *connect.BidiStream[Message, Message]) error {
	defer newrelic.FromContext(ctx).StartSegment("DoStreamStream").End()
	headers := stream.RequestHeader()
	js, _ := json.Marshal(headers)
	for {
		_, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}
		if err := stream.Send(&Message{Text: string(js)}); err != nil {
			return err
		}
	}
}

// DoUnaryUnaryError is a unary request, unary response method that returns an
// error.
func (s *Server) DoUnaryUnaryError(ctx context.Context, req *connect.Request[Message]) (*connect.Response[Message], error) {
	return nil, connect.NewError(connect.CodeDataLoss, nil)
}

// DoUnaryStreamError is a unary request, stream response method that returns an
// error.
func (s *Server) DoUnaryStreamError(ctx context.Context, req *connect.Request[Message], stream *connect.ServerStream[Message]) error {
	return connect.NewError(connect.CodeDataLoss, nil)
}
