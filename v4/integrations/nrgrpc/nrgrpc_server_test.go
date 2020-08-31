// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrgrpc

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	"github.com/newrelic/go-agent/v4/integrations/nrgrpc/testapp"
	"github.com/newrelic/go-agent/v4/internal"
	"github.com/newrelic/go-agent/v4/newrelic"
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
		grpc.WithInsecure(),
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
	if nil != err {
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
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:          "DoUnaryUnary",
			ParentID:      internal.MatchAnyParent,
			SkipAttrsTest: true,
			Attributes: map[string]interface{}{
				"guid":                        internal.MatchAnything,
				"nr.apdexPerfZone":            internal.MatchAnything,
				"parent.account":              123,
				"parent.app":                  456,
				"parent.transportDuration":    internal.MatchAnything,
				"parent.transportType":        "HTTP",
				"parent.type":                 "App",
				"parentId":                    internal.MatchAnything,
				"parentSpanId":                internal.MatchAnything,
				"priority":                    internal.MatchAnything,
				"sampled":                     internal.MatchAnything,
				"traceId":                     internal.MatchAnything,
				"httpResponseCode":            0,
				"http.statusCode":             0,
				"request.headers.contentType": "application/grpc",
				"request.method":              "TestApplication/DoUnaryUnary",
				"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryUnary",
			},
		},
		{
			Name:          "TestApplication/DoUnaryUnary",
			ParentID:      internal.MatchAnyParent,
			SkipAttrsTest: true,
			Attributes: map[string]interface{}{
				"category": "generic",
				"parentId": internal.MatchAnything,
			},
		},
		{
			Name:     "grpc TestApplication/DoUnaryUnary bufnet",
			ParentID: internal.MatchAnyParent,
			Attributes: map[string]interface{}{
				"http.component":   "grpc",
				"http.method":      "TestApplication/DoUnaryUnary",
				"http.status_code": int64(0),
				"http.url":         "unknown",
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
	if nil == err {
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
	app.ExpectSpanEvents(t, []internal.WantSpan{{
		Name:          "TestApplication/DoUnaryUnaryError",
		ParentID:      internal.MatchNoParent,
		SkipAttrsTest: true,
		Attributes: map[string]interface{}{
			"guid":                        internal.MatchAnything,
			"nr.apdexPerfZone":            internal.MatchAnything,
			"priority":                    internal.MatchAnything,
			"sampled":                     internal.MatchAnything,
			"traceId":                     internal.MatchAnything,
			"httpResponseCode":            15,
			"http.statusCode":             15,
			"request.headers.contentType": "application/grpc",
			"request.method":              "TestApplication/DoUnaryUnaryError",
			"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryUnaryError",
		},
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "15",
			"error.message":   "response code 15",
			"guid":            internal.MatchAnything,
			"priority":        internal.MatchAnything,
			"sampled":         internal.MatchAnything,
			"spanId":          internal.MatchAnything,
			"traceId":         internal.MatchAnything,
			"transactionName": "WebTransaction/Go/TestApplication/DoUnaryUnaryError",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":            15,
			"http.statusCode":             15,
			"request.headers.User-Agent":  internal.MatchAnything,
			"request.headers.userAgent":   internal.MatchAnything,
			"request.headers.contentType": "application/grpc",
			"request.method":              "TestApplication/DoUnaryUnaryError",
			"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryUnaryError",
		},
		UserAttributes: map[string]interface{}{},
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
	if nil != err {
		t.Fatal("client call to DoUnaryStream failed", err)
	}
	var recved int
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if nil != err {
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
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:          "DoUnaryStream",
			ParentID:      internal.MatchAnyParent,
			SkipAttrsTest: true,
			Attributes: map[string]interface{}{
				"guid":                        internal.MatchAnything,
				"nr.apdexPerfZone":            internal.MatchAnything,
				"parent.account":              123,
				"parent.app":                  456,
				"parent.transportDuration":    internal.MatchAnything,
				"parent.transportType":        "HTTP",
				"parent.type":                 "App",
				"parentId":                    internal.MatchAnything,
				"parentSpanId":                internal.MatchAnything,
				"priority":                    internal.MatchAnything,
				"sampled":                     internal.MatchAnything,
				"traceId":                     internal.MatchAnything,
				"httpResponseCode":            0,
				"http.statusCode":             0,
				"request.headers.contentType": "application/grpc",
				"request.method":              "TestApplication/DoUnaryStream",
				"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryStream",
			},
		},
		{
			Name:          "TestApplication/DoUnaryStream",
			ParentID:      internal.MatchAnyParent,
			SkipAttrsTest: true,
			Attributes: map[string]interface{}{
				"category": "generic",
				"parentId": internal.MatchAnything,
			},
		},
		{
			Name:     "grpc TestApplication/DoUnaryStream bufnet",
			ParentID: internal.MatchAnyParent,
			Attributes: map[string]interface{}{
				"http.component":   "grpc",
				"http.method":      "TestApplication/DoUnaryStream",
				"http.status_code": int64(0),
				"http.url":         "unknown",
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
	if nil != err {
		t.Fatal("client call to DoStreamUnary failed", err)
	}
	for i := 0; i < 3; i++ {
		if err := stream.Send(&testapp.Message{Text: "Hello DoStreamUnary"}); nil != err {
			if err == io.EOF {
				break
			}
			t.Fatal("failure to Send", err)
		}
	}
	_, err = stream.CloseAndRecv()
	if nil != err {
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
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:          "DoStreamUnary",
			ParentID:      internal.MatchAnyParent,
			SkipAttrsTest: true,
			Attributes: map[string]interface{}{
				"guid":                        internal.MatchAnything,
				"nr.apdexPerfZone":            internal.MatchAnything,
				"parent.account":              123,
				"parent.app":                  456,
				"parent.transportDuration":    internal.MatchAnything,
				"parent.transportType":        "HTTP",
				"parent.type":                 "App",
				"parentId":                    internal.MatchAnything,
				"parentSpanId":                internal.MatchAnything,
				"priority":                    internal.MatchAnything,
				"sampled":                     internal.MatchAnything,
				"traceId":                     internal.MatchAnything,
				"httpResponseCode":            0,
				"http.statusCode":             0,
				"request.headers.contentType": "application/grpc",
				"request.method":              "TestApplication/DoStreamUnary",
				"request.uri":                 "grpc://bufnet/TestApplication/DoStreamUnary",
			},
		},
		{
			Name:          "TestApplication/DoStreamUnary",
			ParentID:      internal.MatchAnyParent,
			SkipAttrsTest: true,
			Attributes: map[string]interface{}{
				"category": "generic",
				"parentId": internal.MatchAnything,
			},
		},
		{
			Name:     "grpc TestApplication/DoStreamUnary bufnet",
			ParentID: internal.MatchAnyParent,
			Attributes: map[string]interface{}{
				"http.component":   "grpc",
				"http.method":      "TestApplication/DoStreamUnary",
				"http.status_code": int64(0),
				"http.url":         "unknown",
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
	if nil != err {
		t.Fatal("client call to DoStreamStream failed", err)
	}
	waitc := make(chan struct{})
	go func() {
		defer close(waitc)
		var recved int
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal("failure to Recv", err)
			}
			recved++
		}
		if recved != 3 {
			t.Fatal("received incorrect number of messages from server", recved)
		}
	}()
	for i := 0; i < 3; i++ {
		if err := stream.Send(&testapp.Message{Text: "Hello DoStreamStream"}); err != nil {
			t.Fatal("failure to Send", err)
		}
	}
	stream.CloseSend()
	<-waitc

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
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:          "DoStreamStream",
			ParentID:      internal.MatchAnyParent,
			SkipAttrsTest: true,
			Attributes: map[string]interface{}{
				"guid":                        internal.MatchAnything,
				"nr.apdexPerfZone":            internal.MatchAnything,
				"parent.account":              123,
				"parent.app":                  456,
				"parent.transportDuration":    internal.MatchAnything,
				"parent.transportType":        "HTTP",
				"parent.type":                 "App",
				"parentId":                    internal.MatchAnything,
				"parentSpanId":                internal.MatchAnything,
				"priority":                    internal.MatchAnything,
				"sampled":                     internal.MatchAnything,
				"traceId":                     internal.MatchAnything,
				"httpResponseCode":            0,
				"http.statusCode":             0,
				"request.headers.contentType": "application/grpc",
				"request.method":              "TestApplication/DoStreamStream",
				"request.uri":                 "grpc://bufnet/TestApplication/DoStreamStream",
			},
		},
		{
			Name:          "TestApplication/DoStreamStream",
			ParentID:      internal.MatchAnyParent,
			SkipAttrsTest: true,
			Attributes: map[string]interface{}{
				"category": "generic",
				"parentId": internal.MatchAnything,
			},
		},
		{
			Name:     "grpc TestApplication/DoStreamStream bufnet",
			ParentID: internal.MatchAnyParent,
			Attributes: map[string]interface{}{
				"http.component":   "grpc",
				"http.method":      "TestApplication/DoStreamStream",
				"http.status_code": int64(0),
				"http.url":         "unknown",
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
	if nil != err {
		t.Fatal("client call to DoUnaryStream failed", err)
	}
	_, err = stream.Recv()
	if nil == err {
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
	app.ExpectSpanEvents(t, []internal.WantSpan{{
		Name:          "TestApplication/DoUnaryStreamError",
		ParentID:      internal.MatchNoParent,
		SkipAttrsTest: true,
		Attributes: map[string]interface{}{
			"guid":                        internal.MatchAnything,
			"nr.apdexPerfZone":            internal.MatchAnything,
			"priority":                    internal.MatchAnything,
			"sampled":                     internal.MatchAnything,
			"traceId":                     internal.MatchAnything,
			"httpResponseCode":            15,
			"http.statusCode":             15,
			"request.headers.contentType": "application/grpc",
			"request.method":              "TestApplication/DoUnaryStreamError",
			"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryStreamError",
		},
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "15",
			"error.message":   "response code 15",
			"guid":            internal.MatchAnything,
			"priority":        internal.MatchAnything,
			"sampled":         internal.MatchAnything,
			"spanId":          internal.MatchAnything,
			"traceId":         internal.MatchAnything,
			"transactionName": "WebTransaction/Go/TestApplication/DoUnaryStreamError",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":            15,
			"http.statusCode":             15,
			"request.headers.User-Agent":  internal.MatchAnything,
			"request.headers.userAgent":   internal.MatchAnything,
			"request.headers.contentType": "application/grpc",
			"request.method":              "TestApplication/DoUnaryStreamError",
			"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryStreamError",
		},
		UserAttributes: map[string]interface{}{},
	}})
}

func TestUnaryServerInterceptorNilApp(t *testing.T) {
	s, conn := newTestServerAndConn(t, nil)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	msg, err := client.DoUnaryUnary(context.Background(), &testapp.Message{})
	if nil != err {
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
	if nil != err {
		t.Fatal("client call to DoStreamUnary failed", err)
	}
	for i := 0; i < 3; i++ {
		if err := stream.Send(&testapp.Message{Text: "Hello DoStreamUnary"}); nil != err {
			if err == io.EOF {
				break
			}
			t.Fatal("failure to Send", err)
		}
	}
	msg, err := stream.CloseAndRecv()
	if nil != err {
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
