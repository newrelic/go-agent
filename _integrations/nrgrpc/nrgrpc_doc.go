// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrgrpc instruments https://github.com/grpc/grpc-go.
//
// This package can be used to instrument gRPC servers and gRPC clients.
//
// To instrument a gRPC server, use UnaryServerInterceptor and
// StreamServerInterceptor with your newrelic.Application to create server
// interceptors to pass to grpc.NewServer.  Example:
//
//
//	cfg := newrelic.NewConfig("gRPC Server", os.Getenv("NEW_RELIC_LICENSE_KEY"))
//	app, _ := newrelic.NewApplication(cfg)
//	server := grpc.NewServer(
//		grpc.UnaryInterceptor(nrgrpc.UnaryServerInterceptor(app)),
//		grpc.StreamInterceptor(nrgrpc.StreamServerInterceptor(app)),
//	)
//
// These interceptors create transactions for inbound calls.  The transaction is
// added to the call context and can be accessed in your method handlers
// using newrelic.FromContext.
//
// Full server example:
// https://github.com/newrelic/go-agent/blob/master/_integrations/nrgrpc/example/server/server.go
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
// Full client example:
// https://github.com/newrelic/go-agent/blob/master/_integrations/nrgrpc/example/client/client.go
package nrgrpc

import "github.com/newrelic/go-agent/internal"

func init() { internal.TrackUsage("integration", "framework", "grpc") }
