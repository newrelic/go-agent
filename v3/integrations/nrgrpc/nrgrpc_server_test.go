// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrgrpc

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/newrelic/go-agent/v3/integrations/nrgrpc/testapp"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// newTestServerAndConn creates a new *grpc.Server and *grpc.ClientConn for use
// in testing. It adds instrumentation to both. If app is nil, then
// instrumentation is not applied to the server. Be sure to Stop() the server
// and Close() the connection when done with them.
func newTestServerAndConn(t *testing.T, app *newrelic.Application) (*grpc.Server, *grpc.ClientConn) {
	s := grpc.NewServer(
		grpc.UnaryInterceptor(UnaryServerInterceptor(app)),
		grpc.StreamInterceptor(StreamServerInterceptor(app)),
	)
	testapp.RegisterTestApplicationServer(s, &testapp.Server{})
	lis := bufconn.Listen(1024 * 1024)

	go func() {
		s.Serve(lis)
	}()

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(), // create the connection synchronously
		grpc.WithUnaryInterceptor(UnaryClientInterceptor),
		grpc.WithStreamInterceptor(StreamClientInterceptor),
	)
	if err != nil {
		t.Fatal("failure to create ClientConn", err)
	}

	return s, conn
}

func TestUnaryServerInterceptor(t *testing.T) {
	app := testApp()

	s, conn := newTestServerAndConn(t, app.Application)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	txn := app.StartTransaction("client")
	ctx := newrelic.NewContext(context.Background(), txn)
	_, err := client.DoUnaryUnary(ctx, &testapp.Message{})
	if err != nil {
		t.Fatal("unable to call client DoUnaryUnary", err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/TestApplication/DoUnaryUnary", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoUnaryUnary", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoUnaryUnary", Scope: "WebTransaction/Go/TestApplication/DoUnaryUnary", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/TestApplication/DoUnaryUnary", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestApplication/DoUnaryUnary", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"guid":                     internal.MatchAnything,
			"name":                     "WebTransaction/Go/TestApplication/DoUnaryUnary",
			"nr.apdexPerfZone":         internal.MatchAnything,
			"parent.account":           123,
			"parent.app":               456,
			"parent.transportDuration": internal.MatchAnything,
			"parent.transportType":     "HTTP",
			"parent.type":              "App",
			"parentId":                 internal.MatchAnything,
			"parentSpanId":             internal.MatchAnything,
			"priority":                 internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"traceId":                  internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":            0,
			"http.statusCode":             0,
			"request.headers.contentType": "application/grpc",
			"request.method":              "TestApplication/DoUnaryUnary",
			"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryUnary",
		},
	}})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category": "generic",
				"name":     "Custom/DoUnaryUnary",
				"parentId": internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"category":         "generic",
				"name":             "WebTransaction/Go/TestApplication/DoUnaryUnary",
				"transaction.name": "WebTransaction/Go/TestApplication/DoUnaryUnary",
				"nr.entryPoint":    true,
				"parentId":         internal.MatchAnything,
				"trustedParentId":  internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"httpResponseCode":            0,
				"http.statusCode":             0,
				"parent.account":              "123",
				"parent.app":                  "456",
				"parent.transportDuration":    internal.MatchAnything,
				"parent.transportType":        "HTTP",
				"parent.type":                 "App",
				"request.headers.contentType": "application/grpc",
				"request.method":              "TestApplication/DoUnaryUnary",
				"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryUnary",
			},
		},
	})
}

func TestUnaryServerInterceptorError(t *testing.T) {
	app := testApp()

	s, conn := newTestServerAndConn(t, app.Application)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	_, err := client.DoUnaryUnaryError(context.Background(), &testapp.Message{})
	if err == nil {
		t.Fatal("DoUnaryUnaryError should have returned an error")
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/TestApplication/DoUnaryUnaryError", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/WebTransaction/Go/TestApplication/DoUnaryUnaryError", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/TestApplication/DoUnaryUnaryError", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestApplication/DoUnaryUnaryError", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"guid":             internal.MatchAnything,
			"name":             "WebTransaction/Go/TestApplication/DoUnaryUnaryError",
			"nr.apdexPerfZone": internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"traceId":          internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{
			"grpcStatusMessage": "oooooops!",
			"grpcStatusCode":    "DataLoss",
			"grpcStatusLevel":   "error",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":            0,
			"http.statusCode":             0,
			"request.headers.contentType": "application/grpc",
			"request.method":              "TestApplication/DoUnaryUnaryError",
			"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryUnaryError",
		},
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "gRPC Status: DataLoss",
			"error.message":   "oooooops!",
			"guid":            internal.MatchAnything,
			"priority":        internal.MatchAnything,
			"sampled":         internal.MatchAnything,
			"spanId":          internal.MatchAnything,
			"traceId":         internal.MatchAnything,
			"transactionName": "WebTransaction/Go/TestApplication/DoUnaryUnaryError",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":            0,
			"http.statusCode":             0,
			"request.headers.User-Agent":  internal.MatchAnything,
			"request.headers.userAgent":   internal.MatchAnything,
			"request.headers.contentType": "application/grpc",
			"request.method":              "TestApplication/DoUnaryUnaryError",
			"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryUnaryError",
		},
		UserAttributes: map[string]interface{}{
			"grpcStatusMessage": "oooooops!",
			"grpcStatusCode":    "DataLoss",
			"grpcStatusLevel":   "error",
		},
	}})
}

func TestUnaryStreamServerInterceptor(t *testing.T) {
	app := testApp()

	s, conn := newTestServerAndConn(t, app.Application)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	txn := app.StartTransaction("client")
	ctx := newrelic.NewContext(context.Background(), txn)
	stream, err := client.DoUnaryStream(ctx, &testapp.Message{})
	if err != nil {
		t.Fatal("client call to DoUnaryStream failed", err)
	}
	var recved int
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal("error receiving message", err)
		}
		recved++
	}
	if recved != 3 {
		t.Fatal("received incorrect number of messages from server", recved)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/TestApplication/DoUnaryStream", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoUnaryStream", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoUnaryStream", Scope: "WebTransaction/Go/TestApplication/DoUnaryStream", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/TestApplication/DoUnaryStream", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestApplication/DoUnaryStream", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"guid":                     internal.MatchAnything,
			"name":                     "WebTransaction/Go/TestApplication/DoUnaryStream",
			"nr.apdexPerfZone":         internal.MatchAnything,
			"parent.account":           123,
			"parent.app":               456,
			"parent.transportDuration": internal.MatchAnything,
			"parent.transportType":     "HTTP",
			"parent.type":              "App",
			"parentId":                 internal.MatchAnything,
			"parentSpanId":             internal.MatchAnything,
			"priority":                 internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"traceId":                  internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":            0,
			"http.statusCode":             0,
			"request.headers.contentType": "application/grpc",
			"request.method":              "TestApplication/DoUnaryStream",
			"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryStream",
		},
	}})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category": "generic",
				"name":     "Custom/DoUnaryStream",
				"parentId": internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"category":         "generic",
				"name":             "WebTransaction/Go/TestApplication/DoUnaryStream",
				"transaction.name": "WebTransaction/Go/TestApplication/DoUnaryStream",
				"nr.entryPoint":    true,
				"parentId":         internal.MatchAnything,
				"trustedParentId":  internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"httpResponseCode":            0,
				"http.statusCode":             0,
				"parent.account":              "123",
				"parent.app":                  "456",
				"parent.transportDuration":    internal.MatchAnything,
				"parent.transportType":        "HTTP",
				"parent.type":                 "App",
				"request.headers.contentType": "application/grpc",
				"request.method":              "TestApplication/DoUnaryStream",
				"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryStream",
			},
		},
	})
}

func TestStreamUnaryServerInterceptor(t *testing.T) {
	app := testApp()

	s, conn := newTestServerAndConn(t, app.Application)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	txn := app.StartTransaction("client")
	ctx := newrelic.NewContext(context.Background(), txn)
	stream, err := client.DoStreamUnary(ctx)
	if err != nil {
		t.Fatal("client call to DoStreamUnary failed", err)
	}
	for i := 0; i < 3; i++ {
		if err := stream.Send(&testapp.Message{Text: "Hello DoStreamUnary"}); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal("failure to Send", err)
		}
	}
	_, err = stream.CloseAndRecv()
	if err != nil {
		t.Fatal("failure to CloseAndRecv", err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/TestApplication/DoStreamUnary", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoStreamUnary", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoStreamUnary", Scope: "WebTransaction/Go/TestApplication/DoStreamUnary", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/TestApplication/DoStreamUnary", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestApplication/DoStreamUnary", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"guid":                     internal.MatchAnything,
			"name":                     "WebTransaction/Go/TestApplication/DoStreamUnary",
			"nr.apdexPerfZone":         internal.MatchAnything,
			"parent.account":           123,
			"parent.app":               456,
			"parent.transportDuration": internal.MatchAnything,
			"parent.transportType":     "HTTP",
			"parent.type":              "App",
			"parentId":                 internal.MatchAnything,
			"parentSpanId":             internal.MatchAnything,
			"priority":                 internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"traceId":                  internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":            0,
			"http.statusCode":             0,
			"request.headers.contentType": "application/grpc",
			"request.method":              "TestApplication/DoStreamUnary",
			"request.uri":                 "grpc://bufnet/TestApplication/DoStreamUnary",
		},
	}})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category": "generic",
				"name":     "Custom/DoStreamUnary",
				"parentId": internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"category":         "generic",
				"name":             "WebTransaction/Go/TestApplication/DoStreamUnary",
				"transaction.name": "WebTransaction/Go/TestApplication/DoStreamUnary",
				"nr.entryPoint":    true,
				"parentId":         internal.MatchAnything,
				"trustedParentId":  internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"httpResponseCode":            0,
				"http.statusCode":             0,
				"parent.account":              "123",
				"parent.app":                  "456",
				"parent.transportDuration":    internal.MatchAnything,
				"parent.transportType":        "HTTP",
				"parent.type":                 "App",
				"request.headers.contentType": "application/grpc",
				"request.method":              "TestApplication/DoStreamUnary",
				"request.uri":                 "grpc://bufnet/TestApplication/DoStreamUnary",
			},
		},
	})
}

func TestStreamStreamServerInterceptor(t *testing.T) {
	app := testApp()

	s, conn := newTestServerAndConn(t, app.Application)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	txn := app.StartTransaction("client")
	ctx := newrelic.NewContext(context.Background(), txn)
	stream, err := client.DoStreamStream(ctx)
	if err != nil {
		t.Fatal("client call to DoStreamStream failed", err)
	}

	errc := make(chan error)
	go func(errc chan error) {
		var recved int
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				errc <- fmt.Errorf("failure to Recv: %v", err)
				return
			}
			recved++
		}
		if recved != 3 {
			errc <- fmt.Errorf("received incorrect number of messages from server: %v", recved)
			return
		}
		errc <- nil
	}(errc)
	for i := 0; i < 3; i++ {
		if err := stream.Send(&testapp.Message{Text: "Hello DoStreamStream"}); err != nil {
			t.Fatal("failure to Send", err)
		}
	}
	stream.CloseSend()

	err = <-errc
	close(errc)
	if err != nil {
		t.Fatal(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/TestApplication/DoStreamStream", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoStreamStream", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoStreamStream", Scope: "WebTransaction/Go/TestApplication/DoStreamStream", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/TestApplication/DoStreamStream", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestApplication/DoStreamStream", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"guid":                     internal.MatchAnything,
			"name":                     "WebTransaction/Go/TestApplication/DoStreamStream",
			"nr.apdexPerfZone":         internal.MatchAnything,
			"parent.account":           123,
			"parent.app":               456,
			"parent.transportDuration": internal.MatchAnything,
			"parent.transportType":     "HTTP",
			"parent.type":              "App",
			"parentId":                 internal.MatchAnything,
			"parentSpanId":             internal.MatchAnything,
			"priority":                 internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"traceId":                  internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":            0,
			"http.statusCode":             0,
			"request.headers.contentType": "application/grpc",
			"request.method":              "TestApplication/DoStreamStream",
			"request.uri":                 "grpc://bufnet/TestApplication/DoStreamStream",
		},
	}})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category": "generic",
				"name":     "Custom/DoStreamStream",
				"parentId": internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"category":         "generic",
				"name":             "WebTransaction/Go/TestApplication/DoStreamStream",
				"transaction.name": "WebTransaction/Go/TestApplication/DoStreamStream",
				"nr.entryPoint":    true,
				"parentId":         internal.MatchAnything,
				"trustedParentId":  internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"httpResponseCode":            0,
				"http.statusCode":             0,
				"parent.account":              "123",
				"parent.app":                  "456",
				"parent.transportDuration":    internal.MatchAnything,
				"parent.transportType":        "HTTP",
				"parent.type":                 "App",
				"request.headers.contentType": "application/grpc",
				"request.method":              "TestApplication/DoStreamStream",
				"request.uri":                 "grpc://bufnet/TestApplication/DoStreamStream",
			},
		},
	})
}

func TestStreamServerInterceptorError(t *testing.T) {
	app := testApp()

	s, conn := newTestServerAndConn(t, app.Application)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	stream, err := client.DoUnaryStreamError(context.Background(), &testapp.Message{})
	if err != nil {
		t.Fatal("client call to DoUnaryStream failed", err)
	}
	_, err = stream.Recv()
	if err == nil {
		t.Fatal("DoUnaryStreamError should have returned an error")
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/TestApplication/DoUnaryStreamError", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/WebTransaction/Go/TestApplication/DoUnaryStreamError", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/TestApplication/DoUnaryStreamError", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestApplication/DoUnaryStreamError", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"guid":             internal.MatchAnything,
			"name":             "WebTransaction/Go/TestApplication/DoUnaryStreamError",
			"nr.apdexPerfZone": internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"traceId":          internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{
			"grpcStatusLevel":   "error",
			"grpcStatusMessage": "oooooops!",
			"grpcStatusCode":    "DataLoss",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":            0,
			"http.statusCode":             0,
			"request.headers.contentType": "application/grpc",
			"request.method":              "TestApplication/DoUnaryStreamError",
			"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryStreamError",
		},
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "gRPC Status: DataLoss",
			"error.message":   "oooooops!",
			"guid":            internal.MatchAnything,
			"priority":        internal.MatchAnything,
			"sampled":         internal.MatchAnything,
			"spanId":          internal.MatchAnything,
			"traceId":         internal.MatchAnything,
			"transactionName": "WebTransaction/Go/TestApplication/DoUnaryStreamError",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":            0,
			"http.statusCode":             0,
			"request.headers.User-Agent":  internal.MatchAnything,
			"request.headers.userAgent":   internal.MatchAnything,
			"request.headers.contentType": "application/grpc",
			"request.method":              "TestApplication/DoUnaryStreamError",
			"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryStreamError",
		},
		UserAttributes: map[string]interface{}{
			"grpcStatusLevel":   "error",
			"grpcStatusMessage": "oooooops!",
			"grpcStatusCode":    "DataLoss",
		},
	}})
}

func TestUnaryServerInterceptorNilApp(t *testing.T) {
	s, conn := newTestServerAndConn(t, nil)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	msg, err := client.DoUnaryUnary(context.Background(), &testapp.Message{})
	if err != nil {
		t.Fatal("unable to call client DoUnaryUnary", err)
	}
	if !strings.Contains(msg.Text, "content-type") {
		t.Error("incorrect message received")
	}
}

func TestStreamServerInterceptorNilApp(t *testing.T) {
	s, conn := newTestServerAndConn(t, nil)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	stream, err := client.DoStreamUnary(context.Background())
	if err != nil {
		t.Fatal("client call to DoStreamUnary failed", err)
	}
	for i := 0; i < 3; i++ {
		if err := stream.Send(&testapp.Message{Text: "Hello DoStreamUnary"}); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal("failure to Send", err)
		}
	}
	msg, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatal("failure to CloseAndRecv", err)
	}
	if !strings.Contains(msg.Text, "content-type") {
		t.Error("incorrect message received")
	}
}

func TestInterceptorsNilAppReturnNonNil(t *testing.T) {
	// When using the `grpc_middleware.WithUnaryServerChain` or
	// `grpc_middleware.WithStreamServerChain` options (see
	// https://godoc.org/github.com/grpc-ecosystem/go-grpc-middleware), calls
	// will panic if our intercepters return nil.
	uInt := UnaryServerInterceptor(nil)
	if uInt == nil {
		t.Error("UnaryServerInterceptor returned nil")
	}

	sInt := StreamServerInterceptor(nil)
	if sInt == nil {
		t.Error("StreamServerInterceptor returned nil")
	}
}
