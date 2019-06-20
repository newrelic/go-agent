package nrgrpc

import (
	"context"
	"net"
	"testing"
	"time"

	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrgrpc/testapp"
	"github.com/newrelic/go-agent/internal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/test/bufconn"
)

func TestTranslateCode(t *testing.T) {
	testcases := []struct {
		grpcCode codes.Code
		httpCode int
	}{
		{grpcCode: 0, httpCode: 200},
		{grpcCode: 1, httpCode: 499},
		{grpcCode: 2, httpCode: 500},
		{grpcCode: 3, httpCode: 400},
		{grpcCode: 4, httpCode: 504},
		{grpcCode: 5, httpCode: 404},
		{grpcCode: 6, httpCode: 409},
		{grpcCode: 7, httpCode: 403},
		{grpcCode: 8, httpCode: 429},
		{grpcCode: 9, httpCode: 400},
		{grpcCode: 10, httpCode: 409},
		{grpcCode: 11, httpCode: 400},
		{grpcCode: 12, httpCode: 501},
		{grpcCode: 13, httpCode: 500},
		{grpcCode: 14, httpCode: 503},
		{grpcCode: 15, httpCode: 500},
		{grpcCode: 16, httpCode: 401},
		{grpcCode: 100, httpCode: 0},
	}

	for _, test := range testcases {
		actual := translateCode(test.grpcCode)
		if actual != test.httpCode {
			t.Errorf("incorrect response code: grpcCode=%d httpCode=%d actual=%d",
				test.grpcCode, test.httpCode, actual)
		}
	}
}

func newTestServerAndConn(t *testing.T, app newrelic.Application) (*grpc.Server, *grpc.ClientConn) {
	s := grpc.NewServer(
		grpc.UnaryInterceptor(UnaryServerInterceptor(app)),
	)
	testapp.RegisterTestApplicationServer(s, &testapp.Server{})
	lis := bufconn.Listen(1024 * 1024)

	go func() {
		s.Serve(lis)
	}()

	var err error
	bufDialer := func(string, time.Duration) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err = grpc.Dial("bufnet",
		grpc.WithDialer(bufDialer),
		grpc.WithInsecure(),
		grpc.WithBlock(), // create the connection synchronously
		grpc.WithUnaryInterceptor(UnaryClientInterceptor),
		grpc.WithStreamInterceptor(StreamClientInterceptor),
	)
	if err != nil {
		t.Fatal("failure to create Dial", err)
	}

	return s, conn
}

func TestUnaryServerInterceptor(t *testing.T) {
	app := testApp(t)

	s, conn := newTestServerAndConn(t, app)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	txn := app.StartTransaction("client", nil, nil)
	ctx := newrelic.NewContext(context.Background(), txn)
	_, err := client.DoUnaryUnary(ctx, &testapp.Message{})
	if nil != err {
		t.Fatal("unable to call client DoUnaryUnary", err)
	}

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/TestApplication/DoUnaryUnary", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoUnaryUnary", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoUnaryUnary", Scope: "WebTransaction/Go/TestApplication/DoUnaryUnary", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/TestApplication/DoUnaryUnary", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestApplication/DoUnaryUnary", Scope: "", Forced: false, Data: nil},
	})
	app.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
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
			"httpResponseCode":            200,
			"request.headers.contentType": "application/grpc",
			"request.method":              "/TestApplication/DoUnaryUnary",
			"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryUnary",
		},
	}})
	app.(internal.Expect).ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category":      "generic",
				"name":          "WebTransaction/Go/TestApplication/DoUnaryUnary",
				"nr.entryPoint": true,
				"parentId":      internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"category": "generic",
				"name":     "Custom/DoUnaryUnary",
				"parentId": internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestUnaryServerInterceptorError(t *testing.T) {
	app := testApp(t)

	s, conn := newTestServerAndConn(t, app)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	_, err := client.DoUnaryUnaryError(context.Background(), &testapp.Message{})
	if nil == err {
		t.Fatal("DoUnaryUnaryError should have returned an error")
	}

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/TestApplication/DoUnaryUnaryError", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/WebTransaction/Go/TestApplication/DoUnaryUnaryError", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/TestApplication/DoUnaryUnaryError", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestApplication/DoUnaryUnaryError", Scope: "", Forced: false, Data: nil},
	})
	app.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"guid":             internal.MatchAnything,
			"name":             "WebTransaction/Go/TestApplication/DoUnaryUnaryError",
			"nr.apdexPerfZone": internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"traceId":          internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":            500,
			"request.headers.contentType": "application/grpc",
			"request.method":              "/TestApplication/DoUnaryUnaryError",
			"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryUnaryError",
		},
	}})
	app.(internal.Expect).ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "500",
			"error.message":   "Internal Server Error",
			"guid":            internal.MatchAnything,
			"priority":        internal.MatchAnything,
			"sampled":         internal.MatchAnything,
			"traceId":         internal.MatchAnything,
			"transactionName": "WebTransaction/Go/TestApplication/DoUnaryUnaryError",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":            500,
			"request.headers.User-Agent":  internal.MatchAnything,
			"request.headers.contentType": "application/grpc",
			"request.method":              "/TestApplication/DoUnaryUnaryError",
			"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryUnaryError",
		},
		UserAttributes: map[string]interface{}{},
	}})
}

func TestUnaryServerInterceptorNilApp(t *testing.T) {
	s, conn := newTestServerAndConn(t, nil)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	_, err := client.DoUnaryUnary(context.Background(), &testapp.Message{})
	if nil != err {
		t.Fatal("unable to call client DoUnaryUnary", err)
	}
}
