// Copyright 2024 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//
// This integration instruments Connect RPC service calls via interceptor functions.
//
// The results of these calls are reported as errors or as informational
// messages based on the Connect status code they return.
//
// In the simplest case, simply add an interceptor when creating your handler or client:
//
//	app, _ := newrelic.NewApplication(
//		newrelic.ConfigAppName("Connect Server"),
//		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
//		newrelic.ConfigDebugLogger(os.Stdout),
//	)
//
//	// For server handlers:
//	mux := http.NewServeMux()
//	path, handler := greetv1connect.NewGreetServiceHandler(
//		&greetServer{},
//		connect.WithInterceptors(nrconnect.Interceptor(app)),
//	)
//	mux.Handle(path, handler)
//
//	// For clients:
//	client := greetv1connect.NewGreetServiceClient(
//		http.DefaultClient,
//		"https://api.acme.com",
//		connect.WithInterceptors(nrconnect.Interceptor(app)),
//	)
//

package nrconnect

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"connectrpc.com/connect"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func getURL(method, target string) *url.URL {
	return &url.URL{
		Scheme: "connect",
		Host:   target,
		Path:   method,
	}
}

// startTransaction starts a New Relic transaction for server-side requests
func startTransaction(ctx context.Context, app *newrelic.Application, method string, hdr http.Header) *newrelic.Transaction {
	method = strings.TrimPrefix(method, "/")
	target := hdr.Get("Host")
	url := getURL(method, target)
	transport := newrelic.TransportHTTP
	webReq := newrelic.WebRequest{
		Header:    hdr,
		URL:       url,
		Method:    method,
		Transport: transport,
	}

	txn := app.StartTransaction(method)
	txn.SetWebRequest(webReq)
	return txn
}

// startClientSegment starts an ExternalSegment for client-side requests and adds Distributed Trace headers
func startClientSegment(ctx context.Context, method, target string, hdr http.Header) *newrelic.ExternalSegment {
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return nil
	}

	method = strings.TrimPrefix(method, "/")
	seg := newrelic.StartExternalSegment(txn, getDummyRequest(method, target, hdr))
	seg.Host = getURL(method, target).Host
	seg.Library = "Connect"
	seg.Procedure = method

	if hdr != nil {
		setTracingHeaders(ctx, hdr)
	}

	return seg
}

func getDummyRequest(method, target string, hdr http.Header) *http.Request {
	return &http.Request{
		Method: "POST",
		URL:    getURL(method, target),
		Header: hdr,
	}
}

// reportStatus reports the Connect error status to New Relic
func reportStatus(ctx context.Context, txn *newrelic.Transaction, err error) {
	if err == nil {
		txn.SetWebResponse(nil).WriteHeader(200)
		return
	}

	// Handle Connect errors
	if connectErr := new(connect.Error); errors.As(err, &connectErr) {
		code := connectErr.Code()
		message := connectErr.Message()

		// Set HTTP status based on Connect code
		// https://connectrpc.com/docs/protocol#error-codes
		httpStatus := 200
		switch code {
		case connect.CodeInvalidArgument, connect.CodeOutOfRange:
			httpStatus = 400
		case connect.CodeUnauthenticated:
			httpStatus = 401
		case connect.CodePermissionDenied:
			httpStatus = 403
		case connect.CodeNotFound:
			httpStatus = 404
		case connect.CodeResourceExhausted:
			httpStatus = 429
		case connect.CodeCanceled:
			httpStatus = 499
		case connect.CodeInternal, connect.CodeDataLoss:
			httpStatus = 500
		case connect.CodeUnimplemented:
			httpStatus = 501
		case connect.CodeUnavailable:
			httpStatus = 503
		case connect.CodeDeadlineExceeded:
			httpStatus = 504
		}

		txn.SetWebResponse(nil).WriteHeader(httpStatus)

		// Report as error for serious status codes
		if code == connect.CodeInternal || code == connect.CodeDataLoss || code == connect.CodeUnknown {
			txn.NoticeError(&newrelic.Error{
				Message: message,
				Class:   "Connect Error: " + code.String(),
			})
		}
		txn.AddAttribute("connectStatusCode", code.String())
		txn.AddAttribute("connectStatusMessage", message)
		return
	}

	// Non-Connect error
	txn.SetWebResponse(nil).WriteHeader(500)
	txn.NoticeError(err)
}

// Interceptor creates a Connect interceptor that instruments RPC calls with New Relic monitoring.
//
// This interceptor automatically creates transactions for server-side handlers and external segments
// for client-side calls. It also adds distributed tracing headers for client requests.
//
// Example usage:
//
//	app, _ := newrelic.NewApplication(...)
//
//	// For server handlers:
//	mux := http.NewServeMux()
//	path, handler := greetv1connect.NewGreetServiceHandler(
//		&greetServer{},
//		connect.WithInterceptors(nrconnect.Interceptor(app)),
//	)
//	mux.Handle(path, handler)
//
//	// For clients:
//	client := greetv1connect.NewGreetServiceClient(
//		http.DefaultClient,
//		"https://api.acme.com",
//		connect.WithInterceptors(nrconnect.Interceptor(app)),
//	)
func Interceptor(app *newrelic.Application) connect.Interceptor {
	return &interceptor{app: app}
}

type interceptor struct {
	app *newrelic.Application
}

func (i *interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	if i.app == nil {
		return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
			return next(ctx, request)
		}
	}

	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		method := req.Spec().Procedure

		if req.Spec().IsClient {
			// Client-side: create external segment
			seg := startClientSegment(ctx, method, req.Peer().Addr, req.Header())
			if seg != nil {
				defer seg.End()
			}
			return next(ctx, req)
		} else {
			// Server-side: create transaction
			txn := startTransaction(ctx, i.app, method, req.Header())
			defer func() {
				txn.End()
			}()

			ctx = newrelic.NewContext(ctx, txn)
			resp, err := next(ctx, req)
			reportStatus(ctx, txn, err)
			return resp, err
		}
	}
}

func (i *interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	if i.app == nil {
		return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
			return next(ctx, spec)
		}
	}

	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		method := spec.Procedure
		target := conn.Peer().Addr
		seg := startClientSegment(ctx, method, target, conn.RequestHeader())
		return &wrappedStreamingClientConn{
			StreamingClientConn: conn,
			segment:             seg,
			ended:               false,
		}
	}
}

func (i *interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	if i.app == nil {
		return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
			return next(ctx, conn)
		}
	}

	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		method := conn.Spec().Procedure
		txn := startTransaction(ctx, i.app, method, conn.RequestHeader())
		defer txn.End()
		ctx = newrelic.NewContext(ctx, txn)
		err := next(ctx, conn)
		reportStatus(ctx, txn, err)
		return err
	}
}

type wrappedStreamingClientConn struct {
	connect.StreamingClientConn
	segment *newrelic.ExternalSegment
	ended   bool
}

func (c *wrappedStreamingClientConn) Receive(m any) error {
	err := c.StreamingClientConn.Receive(m)
	if err != nil && !c.ended {
		if c.segment != nil {
			c.segment.End()
			c.ended = true
		}
	}
	return err
}

func (c *wrappedStreamingClientConn) CloseRequest() error {
	err := c.StreamingClientConn.CloseRequest()
	if !c.ended {
		if c.segment != nil {
			c.segment.End()
			c.ended = true
		}
	}
	return err
}

func (c *wrappedStreamingClientConn) CloseResponse() error {
	err := c.StreamingClientConn.CloseResponse()
	if !c.ended {
		if c.segment != nil {
			c.segment.End()
			c.ended = true
		}
	}
	return err
}

func setTracingHeaders(ctx context.Context, target http.Header) {
	tracingHeaders := http.Header{}
	txn := newrelic.FromContext(ctx)
	if txn != nil {
		txn.InsertDistributedTraceHeaders(tracingHeaders)
		for k, values := range tracingHeaders {
			for _, v := range values {
				target.Set(k, v)
			}
		}
	}
}
