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
	"google.golang.org/grpc/test/bufconn"
)

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
	_, err := client.DoUnaryUnary(context.Background(), &testapp.Message{})
	if nil != err {
		t.Fatal("unable to call client DoUnaryUnary", err)
	}

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/TestApplication/DoUnaryUnary", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoUnaryUnary", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/DoUnaryUnary", Scope: "WebTransaction/Go/TestApplication/DoUnaryUnary", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/TestApplication/DoUnaryUnary", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestApplication/DoUnaryUnary", Scope: "", Forced: false, Data: nil},
	})
	app.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/TestApplication/DoUnaryUnary",
			"guid":             internal.MatchAnything,
			"nr.apdexPerfZone": internal.MatchAnything,
			"priority":         internal.MatchAnything,
			"sampled":          internal.MatchAnything,
			"traceId":          internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"request.method":              "/TestApplication/DoUnaryUnary",
			"request.headers.contentType": "application/grpc",
			"request.uri":                 "grpc://bufnet/TestApplication/DoUnaryUnary",
		},
	}})
	app.(internal.Expect).ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category":      "generic",
				"name":          "WebTransaction/Go/TestApplication/DoUnaryUnary",
				"nr.entryPoint": true,
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
