// Copyright 2024 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrconnect instruments https://github.com/connectrpc/connect-go.
//
// This package can be used to instrument Connect RPC servers and Connect RPC clients.
//
// # Server
//
// To instrument a Connect RPC server, use the Interceptor function with your
// newrelic.Application to create an interceptor to pass to connect.WithInterceptors
// when creating your handler.
//
// The results of these calls are reported as errors or as informational
// messages based on the Connect status code they return.
//
// In the simplest case, simply add an interceptor when creating your handler:
//
//	app, _ := newrelic.NewApplication(
//		newrelic.ConfigAppName("Connect Server"),
//		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
//		newrelic.ConfigDebugLogger(os.Stdout),
//	)
//
//	mux := http.NewServeMux()
//	path, handler := greetv1connect.NewGreetServiceHandler(
//		&greetServer{},
//		connect.WithInterceptors(nrconnect.Interceptor(app)),
//	)
//	mux.Handle(path, handler)
//
// The disposition of each Connect status code is handled as follows:
//
//	OK                   200 HTTP status, no error reported
//	InvalidArgument      400 HTTP status
//	OutOfRange           400 HTTP status
//	Unauthenticated      401 HTTP status
//	PermissionDenied     403 HTTP status
//	NotFound             404 HTTP status
//	ResourceExhausted    429 HTTP status
//	Canceled             499 HTTP status
//	Internal             500 HTTP status, reported as error
//	DataLoss             500 HTTP status, reported as error
//	Unknown              500 HTTP status, reported as error
//	Unimplemented        501 HTTP status
//	Unavailable          503 HTTP status
//	DeadlineExceeded     504 HTTP status
//
// These interceptors create transactions for inbound calls. The transaction is
// added to the call context and can be accessed in your method handlers
// using newrelic.FromContext.
//
//	// handler is your Connect RPC server handler. Access the currently running
//	// transaction using newrelic.FromContext.
//	func (s *Server) Greet(ctx context.Context, req *connect.Request[greetv1.GreetRequest]) (*connect.Response[greetv1.GreetResponse], error) {
//		if err := processRequest(req); err != nil {
//			txn := newrelic.FromContext(ctx)
//			txn.NoticeError(err)
//			return nil, err
//		}
//		return connect.NewResponse(&greetv1.GreetResponse{Greeting: "Hello World!"}), nil
//	}
//
// Full server example:
// https://github.com/newrelic/go-agent/blob/master/v3/integrations/nrconnect/example/server/server.go
//
// # Client
//
// To instrument a Connect RPC client, use the Interceptor function when creating a
// Connect RPC client. Example:
//
//	app, _ := newrelic.NewApplication(
//		newrelic.ConfigAppName("Connect Client"),
//		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
//	)
//
//	client := greetv1connect.NewGreetServiceClient(
//		http.DefaultClient,
//		"https://api.acme.com",
//		connect.WithInterceptors(nrconnect.Interceptor(app)),
//	)
//
// Ensure that calls made with this Connect RPC client are done with a context
// which contains a newrelic.Transaction.
//
//	// Add the currently running transaction to the context before making a
//	// client call.
//	ctx := newrelic.NewContext(context.Background(), txn)
//	resp, err := client.Greet(ctx, connect.NewRequest(&greetv1.GreetRequest{Name: "World"}))
//
// Full client example:
// https://github.com/newrelic/go-agent/blob/master/v3/integrations/nrconnect/example/client/client.go
package nrconnect

import "github.com/newrelic/go-agent/v3/internal"

func init() { internal.TrackUsage("integration", "framework", "connect") }
