// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//
// Package nrgrpc instruments https://github.com/grpc/grpc-go.
//
// This package can be used to instrument gRPC servers and gRPC clients.
//
// Server
//
// To instrument a gRPC server, use UnaryServerInterceptor and
// StreamServerInterceptor with your newrelic.Application to create server
// interceptors to pass to grpc.NewServer.
//
// The results of these calls are reported as errors or as informational
// messages (of levels OK, Info, Warning, or Error) based on the gRPC status
// code they return.
//
// In the simplest case, simply add interceptors as in the following example:
//
//  app, _ := newrelic.NewApplication(
//     newrelic.ConfigAppName("gRPC Server"),
//     newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
//     newrelic.ConfigDebugLogger(os.Stdout),
//  )
//  server := grpc.NewServer(
//     grpc.UnaryInterceptor(nrgrpc.UnaryServerInterceptor(app)),
//     grpc.StreamInterceptor(nrgrpc.StreamServerInterceptor(app)),
//  )
//
// The disposition of each, in terms of how to report each of the various
// gRPC status codes, is determined by a built-in set of defaults:
//   OK       OK
//   Info     AlreadyExists, Canceled, InvalidArgument, NotFound,
//            Unauthenticated
//   Warning  Aborted, DeadlineExceeded, FailedPrecondition, OutOfRange,
//            PermissionDenied, ResourceExhausted, Unavailable
//   Error    DataLoss, Internal, Unknown, Unimplemented
//
// These
// may be overridden on a case-by-case basis using `WithStatusHandler()`
// options to each `UnaryServerInterceptor()` or `StreamServerInterceptor()`
// call, or globally via the `Configure()` function.
//
// For example, to report DeadlineExceeded as an error and NotFound
// as a warning, for the UnaryInterceptor only:
//   server := grpc.NewServer(
//      grpc.UnaryInterceptor(nrgrpc.UnaryServerInterceptor(app,
//       nrgrpc.WithStatusHandler(codes.DeadlineExceeded, nrgrpc.ErrorInterceptorStatusHandler),
//       nrgrpc.WithStatusHandler(codes.NotFound, nrgrpc.WarningInterceptorStatusHandler)),
//      grpc.StreamInterceptor(nrgrpc.StreamServerInterceptor(app)),
//   )
//
// If you wanted to make those two changes to the overall default behavior, so they
// apply to all subsequently declared interceptors:
//   nrgrpc.Configure(
//     nrgrpc.WithStatusHandler(codes.DeadlineExceeded, nrgrpc.ErrorInterceptorStatusHandler),
//     nrgrpc.WithStatusHandler(codes.NotFound, nrgrpc.WarningInterceptorStatusHandler),
//   )
//   server := grpc.NewServer(
//      grpc.UnaryInterceptor(nrgrpc.UnaryServerInterceptor(app)),
//      grpc.StreamInterceptor(nrgrpc.StreamServerInterceptor(app)),
//   )
// In this case the new behavior for those two status codes applies to both interceptors.
//
// These interceptors create transactions for inbound calls.  The transaction is
// added to the call context and can be accessed in your method handlers
// using newrelic.FromContext.
//
//	// handler is your gRPC server handler. Access the currently running
//	// transaction using newrelic.FromContext.
//	func (s *Server) handler(ctx context.Context, msg *pb.Message) (*pb.Message, error) {
//		if err := processMsg(msg); err != nil {
//			txn := newrelic.FromContext(ctx)
//			txn.NoticeError(err)
//			return nil, err
//		}
// 		return &pb.Message{Text: "Hello World!"}, nil
// 	}
//
// Full server example:
// https://github.com/newrelic/go-agent/blob/master/v3/integrations/nrgrpc/example/server/server.go
//
// Client
//
// To instrument a gRPC client, follow these two steps:
//
// 1. Use UnaryClientInterceptor and StreamClientInterceptor when creating a
// grpc.ClientConn.  Example:
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
//	// Add the currently running transaction to the context before making a
//	// client call.
//	ctx := newrelic.NewContext(context.Background(), txn)
//	msg, err := client.handler(ctx, &pb.Message{"Hello World"})
//
// Full client example:
// https://github.com/newrelic/go-agent/blob/master/v3/integrations/nrgrpc/example/client/client.go
package nrgrpc

import "github.com/newrelic/go-agent/v3/internal"

func init() { internal.TrackUsage("integration", "framework", "grpc") }
