// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrgrpc instruments https://github.com/grpc/grpc-go.
//
// This package can be used to instrument gRPC servers and gRPC clients.
//
// Server
//
// To instrument a gRPC server, use UnaryServerInterceptor and
// StreamServerInterceptor with your newrelic.Application to create server
// interceptors to pass to grpc.NewServer.  Example:
//
//
//	app, _ := newrelic.NewApplication(
//		newrelic.ConfigAppName("gRPC Server"),
//		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
//		newrelic.ConfigDebugLogger(os.Stdout),
//	)
//	server := grpc.NewServer(
//		grpc.UnaryInterceptor(nrgrpc.UnaryServerInterceptor(app)),
//		grpc.StreamInterceptor(nrgrpc.StreamServerInterceptor(app)),
//	)
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
