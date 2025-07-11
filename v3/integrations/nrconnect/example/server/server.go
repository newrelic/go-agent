// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/newrelic/go-agent/v3/newrelic"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/newrelic/go-agent/v3/integrations/nrconnect"
	"github.com/newrelic/go-agent/v3/integrations/nrconnect/example/sampleapp"
	"github.com/newrelic/go-agent/v3/integrations/nrconnect/example/sampleapp/sampleappconnect"
)

type service struct{}

func processMessage(ctx context.Context, msg *sampleapp.Message) {
	defer newrelic.FromContext(ctx).StartSegment("processMessage").End()
	log.Printf("Message received: %s\n", msg.Text)
}

func (s *service) DoUnaryUnary(ctx context.Context, req *connect.Request[sampleapp.Message]) (*connect.Response[sampleapp.Message], error) {
	processMessage(ctx, req.Msg)
	return connect.NewResponse(&sampleapp.Message{Text: "Hello from DoUnaryUnary"}), nil
}

func (s *service) DoUnaryStream(ctx context.Context, req *connect.Request[sampleapp.Message], stream *connect.ServerStream[sampleapp.Message]) error {
	processMessage(ctx, req.Msg)
	for i := 0; i < 3; i++ {
		if err := stream.Send(&sampleapp.Message{Text: "Hello from DoUnaryStream"}); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) DoStreamUnary(ctx context.Context, stream *connect.ClientStream[sampleapp.Message]) (*connect.Response[sampleapp.Message], error) {
	for stream.Receive() {
		msg := stream.Msg()
		processMessage(ctx, msg)
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	return connect.NewResponse(&sampleapp.Message{Text: "Hello from DoStreamUnary"}), nil
}

func (s *service) DoStreamStream(ctx context.Context, stream *connect.BidiStream[sampleapp.Message, sampleapp.Message]) error {
	for {
		msg, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		processMessage(ctx, msg)
		if err := stream.Send(&sampleapp.Message{Text: "Hello from DoStreamStream"}); err != nil {
			return err
		}
	}
}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Connect service"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if err != nil {
		panic(err)
	}

	// Create the Connect handler with New Relic instrumentation
	mux := http.NewServeMux()
	mux.Handle(sampleappconnect.NewSampleApplicationHandler(
		&service{},
		connect.WithInterceptors(nrconnect.Interceptor(app)),
	))
	server := &http.Server{
		Addr:    ":8080",
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	server.ListenAndServe()
}
