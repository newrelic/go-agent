// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

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
	}
	txn := app.StartTransaction(method)
	txn.SetWebRequest(webReq)

	return txn
}

// UnaryServerInterceptor instruments server unary RPCs.
//
// Use this function with grpc.UnaryInterceptor and a newrelic.Application to
// sreate a grpc.ServerOption to pass to grpc.NewServer.  This interceptor
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
// The nrgrpc integration has a built-in set of handlers for each gRPC status
// code encountered. Serious errors are reported as error traces ala the
// `newrelic.NoticeError()` function, while the others are reported but not
// counted as errors.
//
// If you wish to change this behavior, you may do so at a global level for
// all intercepted functions by calling the `Configure()` function, passing
// any number of `WithStatusHandler(code, handler)` functions as parameters.
// In each of these, `code` represents a gRPC code such as `codes.Unknown`,
// and `handler` is a function with the calling signature
//   func myHandler(c context.Context, t *newrelic.Transaction, s *status.Status)
// If the given gRPC code is produced by the intercepted function, then `myHandler()`
// will be called to report that out in the current transaction, in whatever way
// is appropriate. To assist with this, `myHandler()` is provided with the corresponding
// context and transaction along with the actual gRPC `Status` value captured.
//
// We provide a set of standard handlers which should suffice in most cases to report
// non-error, info-level, warning-level, and error-level statuses:
//   ErrorInterceptorStatusHandler     // report as error
//   IgnoreInterceptorStatusHandler    // no report AT ALL
//   InfoInterceptorStatusHandler      // report as informational message
//   OKInterceptorStatusHandler        // report as successful
//   WarningInterceptorStatusHandler   // report as warning
//
// Thus, to specify that all codes with status `OutOfRange` should be logged as warnings
// and all `Unimplemented` ones should be informational, then make this call:
//   Config(
//     WithStatusHandler(codes.OutOfRange, WarningInterceptorStatusHandler),
//     WithStatusHandler(codes.Unimplemented, InfoInterceptorStatusHandler))
//
// This will affect the behavior of calls to `UnaryInterceptor()` and `StreamInterceptor()`
// which occur after this. You may call `Config()` again to change the handling of errors
// from that point forward (but note that once an interceptor is created, it will use whatever
// handlers were defined at that point, even if the intercepted service call happens later.
//
// You can also specify a custom set of handlers with each interceptor creation by adding
// `WithStatusHandler()` calls at the end of the `<type>StreamInterceptor()` call's parameter list,
// like so:
//   grpc.UnaryInterceptor(nrgrpc.UnaryServerInterceptor(app,
//     nrgrpc.WithStatusHandler(codes.OutOfRange, nrgrpc.WarningInterceptorStatusHandler),
//     nrgrpc.WithStatusHandler(codes.Unimplemented, nrgrpc.InfoInterceptorStatusHandler)))
// In this case, those two handlers are used (along with the current defaults for the other status
// codes) only for that interceptor.
//
//
// These interceptors add the transaction to the call context so it may be
// accessed in your method handlers using newrelic.FromContext.
//
// Full example:
// https://github.com/newrelic/go-agent/blob/master/v3/integrations/nrgrpc/example/server/server.go
//

type ErrorHandler func(context.Context, *newrelic.Transaction, *status.Status)
type StatusHandlerMap map[codes.Code]ErrorHandler

var interceptorStatusHandlerRegistry = StatusHandlerMap{
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

type handlerOption func(StatusHandlerMap)

func WithStatusHandler(c codes.Code, h ErrorHandler) handlerOption {
	return func(m StatusHandlerMap) {
		m[c] = h
	}
}

func Configure(options ...handlerOption) {
	for _, option := range options {
		option(interceptorStatusHandlerRegistry)
	}
}

func IgnoreInterceptorStatusHandler(_ context.Context, _ *newrelic.Transaction, _ *status.Status) {}

func OKInterceptorStatusHandler(ctx context.Context, txn *newrelic.Transaction, s *status.Status) {
	txn.SetWebResponse(nil).WriteHeader(int(codes.OK))
}

func ErrorInterceptorStatusHandler(ctx context.Context, txn *newrelic.Transaction, s *status.Status) {
	txn.SetWebResponse(nil).WriteHeader(int(s.Code()))
	//
	// TODO: Figure out specifically how we want to set up the custom attributes for this
	//       (it was txn.NoticeError(s.Err()))
	txn.NoticeError(&newrelic.Error{
		Message: s.Err().Error(),
		Class:   "...",
		Attributes: map[string]interface{}{
			"...": "...",
		},
	})
}

func WarningInterceptorStatusHandler(ctx context.Context, txn *newrelic.Transaction, s *status.Status) {
	txn.SetWebResponse(nil).WriteHeader(int(s.Code()))
	// TODO: just add some attributes about this (treat as WARN, not NoticeError())
}

func InfoInterceptorStatusHandler(ctx context.Context, txn *newrelic.Transaction, s *status.Status) {
	txn.SetWebResponse(nil).WriteHeader(int(s.Code()))
	// TODO: just add some attributes about this (treat as INFO, not NoticeError())
}

var DefaultInterceptorStatusHandler = InfoInterceptorStatusHandler

func reportInterceptorStatus(ctx context.Context, txn *newrelic.Transaction, handlers StatusHandlerMap, err error) {
	grpcStatus := status.Convert(err)
	handler, ok := handlers[grpcStatus.Code()]
	if !ok {
		handler = DefaultInterceptorStatusHandler
	}
	handler(ctx, txn, grpcStatus)
}

func UnaryServerInterceptor(app *newrelic.Application, options ...handlerOption) grpc.UnaryServerInterceptor {
	if app == nil {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}

	localHandlerMap := make(StatusHandlerMap)
	for code, handler := range interceptorStatusHandlerRegistry {
		localHandlerMap[code] = handler
	}
	for _, option := range options {
		option(localHandlerMap)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		txn := startTransaction(ctx, app, info.FullMethod)
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
// Full example:
// https://github.com/newrelic/go-agent/blob/master/v3/integrations/nrgrpc/example/server/server.go
//
func StreamServerInterceptor(app *newrelic.Application, options ...handlerOption) grpc.StreamServerInterceptor {
	if app == nil {
		return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}

	localHandlerMap := make(StatusHandlerMap)
	for code, handler := range interceptorStatusHandlerRegistry {
		localHandlerMap[code] = handler
	}
	for _, option := range options {
		option(localHandlerMap)
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		txn := startTransaction(ss.Context(), app, info.FullMethod)
		defer txn.End()

		err := handler(srv, newWrappedServerStream(ss, txn))
		reportInterceptorStatus(ss.Context(), txn, localHandlerMap, err)
		return err
	}
}
