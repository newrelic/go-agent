// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//
// This integration instruments grpc service calls via
// UnaryServerInterceptor and StreamServerInterceptor functions.
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
// gRPC status codes, is determined by a built-in set of defaults. These
// may be overridden on a case-by-case basis using WithStatusHandler
// options to each UnaryServerInterceptor or StreamServerInterceptor
// call, or globally via the Configure function.
//
// Full example:
// https://github.com/newrelic/go-agent/blob/master/v3/integrations/nrgrpc/example/server/server.go
//

package nrgrpc

import (
	"context"
	"net/http"
	"strings"

	"github.com/newrelic/go-agent/v3/newrelic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func startTransaction(ctx context.Context, app *newrelic.Application, fullMethod string) *newrelic.Transaction {
	method := strings.TrimPrefix(fullMethod, "/")

	var hdrs http.Header
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		hdrs = make(http.Header, len(md))
		for k, vs := range md {
			for _, v := range vs {
				hdrs.Add(k, v)
			}
		}
	}

	target := hdrs.Get(":authority")
	url := getURL(method, target)

	webReq := newrelic.WebRequest{
		Header:    hdrs,
		URL:       url,
		Method:    method,
		Transport: newrelic.TransportHTTP,
		Type:      "gRPC",
	}
	txn := app.StartTransaction(method)
	txn.SetWebRequest(webReq)

	return txn
}

//
// ErrorHandler is the type of a gRPC status handler function.
// Normally the supplied set of ErrorHandler functions will suffice, but
// a custom handler may be crafted by the user and installed as a handler
// if needed.
//
type ErrorHandler func(context.Context, *newrelic.Transaction, *status.Status)

//
// Internal registry of handlers associated with various
// status codes.
//
type statusHandlerMap map[codes.Code]ErrorHandler

//
// interceptorStatusHandlerRegistry is the current default set of handlers
// used by each interceptor.
//
var interceptorStatusHandlerRegistry = statusHandlerMap{
	codes.OK:                 OKInterceptorStatusHandler,
	codes.Canceled:           InfoInterceptorStatusHandler,
	codes.Unknown:            ErrorInterceptorStatusHandler,
	codes.InvalidArgument:    InfoInterceptorStatusHandler,
	codes.DeadlineExceeded:   WarningInterceptorStatusHandler,
	codes.NotFound:           InfoInterceptorStatusHandler,
	codes.AlreadyExists:      InfoInterceptorStatusHandler,
	codes.PermissionDenied:   WarningInterceptorStatusHandler,
	codes.ResourceExhausted:  WarningInterceptorStatusHandler,
	codes.FailedPrecondition: WarningInterceptorStatusHandler,
	codes.Aborted:            WarningInterceptorStatusHandler,
	codes.OutOfRange:         WarningInterceptorStatusHandler,
	codes.Unimplemented:      ErrorInterceptorStatusHandler,
	codes.Internal:           ErrorInterceptorStatusHandler,
	codes.Unavailable:        WarningInterceptorStatusHandler,
	codes.DataLoss:           ErrorInterceptorStatusHandler,
	codes.Unauthenticated:    InfoInterceptorStatusHandler,
}

//
// HandlerOption is the type for options passed to the interceptor
// functions to specify gRPC status handlers.
//
type HandlerOption func(statusHandlerMap)

//
// WithStatusHandler indicates a handler function to be used to
// report the indicated gRPC status. Zero or more of these may be
// given to the Configure, StreamServiceInterceptor, or
// UnaryServiceInterceptor functions.
//
// The ErrorHandler parameter is generally one of the provided standard
// reporting functions:
//  OKInterceptorStatusHandler      // report the operation as successful
//  ErrorInterceptorStatusHandler   // report the operation as an error
//  WarningInterceptorStatusHandler // report the operation as a warning
//  InfoInterceptorStatusHandler    // report the operation as an informational message
//
// The following reporting function should only be used if you know for sure
// you want this. It does not report the error in any way at all, but completely
// ignores it.
//  IgnoreInterceptorStatusHandler  // report the operation as successful
//
// Finally, if you have a custom reporting need that isn't covered by the standard
// handler functions, you can create your own handler function as
//   func myHandler(ctx context.Context, txn *newrelic.Transaction, s *status.Status) {
//      ...
//   }
// Within the function, do whatever you need to do with the txn parameter to report the
// gRPC status passed as s. If needed, the context is also passed to your function.
//
// If you wish to use your custom handler for a code such as codes.NotFound, you would
// include the parameter
//   WithStatusHandler(codes.NotFound, myHandler)
// to your Configure, StreamServiceInterceptor, or UnaryServiceInterceptor function.
//
func WithStatusHandler(c codes.Code, h ErrorHandler) HandlerOption {
	return func(m statusHandlerMap) {
		m[c] = h
	}
}

//
// Configure takes a list of WithStatusHandler options and sets them
// as the new default handlers for the specified gRPC status codes, in the same
// way as if WithStatusHandler were given to the StreamServiceInterceptor
// or UnaryServiceInterceptor functions (q.v.); however, in this case the new handlers
// become the default for any subsequent interceptors created by the above functions.
//
func Configure(options ...HandlerOption) {
	for _, option := range options {
		option(interceptorStatusHandlerRegistry)
	}
}

//
// IgnoreInterceptorStatusHandler is our standard handler for
// gRPC statuses which we want to ignore (in terms of any gRPC-specific
// reporting on the transaction).
//
func IgnoreInterceptorStatusHandler(_ context.Context, _ *newrelic.Transaction, _ *status.Status) {}

//
// OKInterceptorStatusHandler is our standard handler for
// gRPC statuses which we want to report as being successful, as with the
// status code OK.
//
// This adds no additional attributes on the transaction other than
// the fact that it was successful.
//
func OKInterceptorStatusHandler(ctx context.Context, txn *newrelic.Transaction, s *status.Status) {
	txn.SetWebResponse(nil).WriteHeader(int(codes.OK))
}

//
// ErrorInterceptorStatusHandler is our standard handler for
// gRPC statuses which we want to report as being errors,
// with the relevant error messages and
// contextual information gleaned from the error value received from the RPC call.
//
func ErrorInterceptorStatusHandler(ctx context.Context, txn *newrelic.Transaction, s *status.Status) {
	txn.SetWebResponse(nil).WriteHeader(int(codes.OK))
	txn.NoticeError(&newrelic.Error{
		Message: s.Message(),
		Class:   "gRPC Status: " + s.Code().String(),
	})
	txn.AddAttribute("grpcStatusLevel", "error")
	txn.AddAttribute("grpcStatusMessage", s.Message())
	txn.AddAttribute("grpcStatusCode", s.Code().String())
}

//
// WarningInterceptorStatusHandler is our standard handler for
// gRPC statuses which we want to report as warnings.
//
// Reports the transaction's status with attributes containing information gleaned
// from the error value returned, but does not count this as an error.
//
func WarningInterceptorStatusHandler(ctx context.Context, txn *newrelic.Transaction, s *status.Status) {
	txn.SetWebResponse(nil).WriteHeader(int(codes.OK))
	txn.AddAttribute("grpcStatusLevel", "warning")
	txn.AddAttribute("grpcStatusMessage", s.Message())
	txn.AddAttribute("grpcStatusCode", s.Code().String())
}

//
// InfoInterceptorStatusHandler is our standard handler for
// gRPC statuses which we want to report as informational messages only.
//
// Reports the transaction's status with attributes containing information gleaned
// from the error value returned, but does not count this as an error.
//
func InfoInterceptorStatusHandler(ctx context.Context, txn *newrelic.Transaction, s *status.Status) {
	txn.SetWebResponse(nil).WriteHeader(int(codes.OK))
	txn.AddAttribute("grpcStatusLevel", "info")
	txn.AddAttribute("grpcStatusMessage", s.Message())
	txn.AddAttribute("grpcStatusCode", s.Code().String())
}

//
// DefaultInterceptorStatusHandler indicates which of our standard handlers
// will be used for any status code which is not
// explicitly assigned a handler.
//
var DefaultInterceptorStatusHandler = InfoInterceptorStatusHandler

//
// reportInterceptorStatus is the common routine for reporting any kind of interceptor.
//
func reportInterceptorStatus(ctx context.Context, txn *newrelic.Transaction, handlers statusHandlerMap, err error) {
	grpcStatus := status.Convert(err)
	handler, ok := handlers[grpcStatus.Code()]
	if !ok {
		handler = DefaultInterceptorStatusHandler
	}
	handler(ctx, txn, grpcStatus)
}

// UnaryServerInterceptor instruments server unary RPCs.
//
// Use this function with grpc.UnaryInterceptor and a newrelic.Application to
// create a grpc.ServerOption to pass to grpc.NewServer.  This interceptor
// records each unary call with a transaction.  You must use both
// UnaryServerInterceptor and StreamServerInterceptor to instrument unary and
// streaming calls.
//
// Example:
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
// These interceptors add the transaction to the call context so it may be
// accessed in your method handlers using newrelic.FromContext.
//
// The nrgrpc integration has a built-in set of handlers for each gRPC status
// code encountered. Serious errors are reported as error traces à la the
// newrelic.NoticeError function, while the others are reported but not
// counted as errors.
//
// If you wish to change this behavior, you may do so at a global level for
// all intercepted functions by calling the Configure function, passing
// any number of WithStatusHandler(code, handler) functions as parameters.
//
// You can specify a custom set of handlers with each interceptor creation by adding
// WithStatusHandler calls at the end of the <type>StreamInterceptor call's parameter list,
// like so:
//   grpc.UnaryInterceptor(nrgrpc.UnaryServerInterceptor(app,
//     nrgrpc.WithStatusHandler(codes.OutOfRange, nrgrpc.WarningInterceptorStatusHandler),
//     nrgrpc.WithStatusHandler(codes.Unimplemented, nrgrpc.InfoInterceptorStatusHandler)))
// In this case, those two handlers are used (along with the current defaults for the other status
// codes) only for that interceptor.
//
func UnaryServerInterceptor(app *newrelic.Application, options ...HandlerOption) grpc.UnaryServerInterceptor {
	if app == nil {
		return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}
	}

	localHandlerMap := make(statusHandlerMap)
	for code, handler := range interceptorStatusHandlerRegistry {
		localHandlerMap[code] = handler
	}
	for _, option := range options {
		option(localHandlerMap)
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		txn := startTransaction(ctx, app, info.FullMethod)
		newrelic.GetSecurityAgentInterface().SendEvent("GRPC", req)
		defer txn.End()

		ctx = newrelic.NewContext(ctx, txn)
		resp, err = handler(ctx, req)
		reportInterceptorStatus(ctx, txn, localHandlerMap, err)
		return
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	txn *newrelic.Transaction
}

func (s wrappedServerStream) Context() context.Context {
	ctx := s.ServerStream.Context()
	return newrelic.NewContext(ctx, s.txn)
}

func (s wrappedServerStream) RecvMsg(msg any) error {
	newrelic.GetSecurityAgentInterface().SendEvent("GRPC", msg)
	return s.ServerStream.RecvMsg(msg)
}

func newWrappedServerStream(stream grpc.ServerStream, txn *newrelic.Transaction) grpc.ServerStream {
	return wrappedServerStream{
		ServerStream: stream,
		txn:          txn,
	}
}

// StreamServerInterceptor instruments server streaming RPCs.
//
// Use this function with grpc.StreamInterceptor and a newrelic.Application to
// create a grpc.ServerOption to pass to grpc.NewServer.  This interceptor
// records each streaming call with a transaction.  You must use both
// UnaryServerInterceptor and StreamServerInterceptor to instrument unary and
// streaming calls.
//
// See the notes and examples for the UnaryServerInterceptor function.
//
func StreamServerInterceptor(app *newrelic.Application, options ...HandlerOption) grpc.StreamServerInterceptor {
	if app == nil {
		return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}

	localHandlerMap := make(statusHandlerMap)
	for code, handler := range interceptorStatusHandlerRegistry {
		localHandlerMap[code] = handler
	}
	for _, option := range options {
		option(localHandlerMap)
	}

	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		txn := startTransaction(ss.Context(), app, info.FullMethod)
		defer txn.End()

		err := handler(srv, newWrappedServerStream(ss, txn))
		reportInterceptorStatus(ss.Context(), txn, localHandlerMap, err)
		return err
	}
}
