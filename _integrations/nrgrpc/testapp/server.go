// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package testapp

import (
	"context"
	"encoding/json"
	"io"

	newrelic "github.com/newrelic/go-agent"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"
)

// Server is a gRPC server.
type Server struct{}

// DoUnaryUnary is a unary request, unary response method.
func (s *Server) DoUnaryUnary(ctx context.Context, msg *Message) (*Message, error) {
	defer newrelic.StartSegment(newrelic.FromContext(ctx), "DoUnaryUnary").End()
	md, _ := metadata.FromIncomingContext(ctx)
	js, _ := json.Marshal(md)
	return &Message{Text: string(js)}, nil
}

// DoUnaryStream is a unary request, stream response method.
func (s *Server) DoUnaryStream(msg *Message, stream TestApplication_DoUnaryStreamServer) error {
	defer newrelic.StartSegment(newrelic.FromContext(stream.Context()), "DoUnaryStream").End()
	md, _ := metadata.FromIncomingContext(stream.Context())
	js, _ := json.Marshal(md)
	for i := 0; i < 3; i++ {
		if err := stream.Send(&Message{Text: string(js)}); nil != err {
			return err
		}
	}
	return nil
}

// DoStreamUnary is a stream request, unary response method.
func (s *Server) DoStreamUnary(stream TestApplication_DoStreamUnaryServer) error {
	defer newrelic.StartSegment(newrelic.FromContext(stream.Context()), "DoStreamUnary").End()
	md, _ := metadata.FromIncomingContext(stream.Context())
	js, _ := json.Marshal(md)
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&Message{Text: string(js)})
		} else if nil != err {
			return err
		}
	}
}

// DoStreamStream is a stream request, stream response method.
func (s *Server) DoStreamStream(stream TestApplication_DoStreamStreamServer) error {
	defer newrelic.StartSegment(newrelic.FromContext(stream.Context()), "DoStreamStream").End()
	md, _ := metadata.FromIncomingContext(stream.Context())
	js, _ := json.Marshal(md)
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			return nil
		} else if nil != err {
			return err
		}
		if err := stream.Send(&Message{Text: string(js)}); nil != err {
			return err
		}
	}
}

// DoUnaryUnaryError is a unary request, unary response method that returns an
// error.
func (s *Server) DoUnaryUnaryError(ctx context.Context, msg *Message) (*Message, error) {
	return &Message{}, status.New(codes.DataLoss, "oooooops!").Err()
}

// DoUnaryStreamError is a unary request, unary response method that returns an
// error.
func (s *Server) DoUnaryStreamError(msg *Message, stream TestApplication_DoUnaryStreamErrorServer) error {
	return status.New(codes.DataLoss, "oooooops!").Err()
}
