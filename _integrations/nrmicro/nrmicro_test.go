package nrmicro

import (
	"context"
	"testing"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/client/selector"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/registry/memory"
	"github.com/micro/go-micro/server"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

const (
	missingHeaders  = "HEADERS NOT FOUND"
	missingMetadata = "METADATA NOT FOUND"
)

type TestRequest struct{}

type TestResponse struct {
	RequestHeaders string
}

func (r TestResponse) dtHeadersFound() bool {
	return r.RequestHeaders != "" && r.RequestHeaders != missingMetadata && r.RequestHeaders != missingHeaders
}

type TestHandler struct{}

func (t *TestHandler) Method(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	if md, ok := metadata.FromContext(ctx); ok {
		if dtHeader, ok := md[newrelic.DistributedTracePayloadHeader]; ok {
			rsp.RequestHeaders = dtHeader
		} else {
			rsp.RequestHeaders = missingHeaders
		}
	} else {
		rsp.RequestHeaders = missingMetadata
	}
	return nil
}

func createTestApp(t *testing.T) newrelic.Application {
	cfg := newrelic.NewConfig("appname", "0123456789012345678901234567890123456789")
	cfg.Enabled = false
	cfg.DistributedTracer.Enabled = true
	cfg.TransactionTracer.SegmentThreshold = 0
	cfg.TransactionTracer.Threshold.IsApdexFailing = false
	cfg.TransactionTracer.Threshold.Duration = 0
	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		t.Fatal(err)
	}
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
		reply.AccountID = "123"
		reply.TrustedAccountKey = "123"
		reply.PrimaryAppID = "456"
	}
	internal.HarvestTesting(app, replyfn)
	return app
}

func newTestClientAndServer(t *testing.T) (client.Client, server.Server) {
	registry := memory.NewRegistry()
	sel := selector.NewSelector(selector.Registry(registry))
	c := client.NewClient(
		client.Selector(sel),
		client.Wrap(ClientWrapper),
	)
	s := server.NewServer(
		server.Name("testing"),
		server.Registry(registry),
	)
	s.Handle(s.NewHandler(new(TestHandler)))
	if err := s.Start(); nil != err {
		t.Fatal(err)
	}
	return c, s
}

func TestClientCallWithNoTransaction(t *testing.T) {
	c, s := newTestClientAndServer(t)
	defer s.Stop()

	ctx := context.Background()
	req := c.NewRequest("testing", "TestHandler.Method", &TestRequest{}, client.WithContentType("application/json"))
	rsp := TestResponse{}
	if err := c.Call(ctx, req, &rsp); nil != err {
		t.Fatal("Error calling test client:", err)
	}
	if rsp.RequestHeaders != missingHeaders {
		t.Error("Header should not be here", rsp.RequestHeaders)
	}
}

func TestClientCallWithTransaction(t *testing.T) {
	c, s := newTestClientAndServer(t)
	defer s.Stop()

	req := c.NewRequest("testing", "TestHandler.Method", &TestRequest{}, client.WithContentType("application/json"))
	rsp := TestResponse{}
	app := createTestApp(t)
	txn := app.StartTransaction("name", nil, nil)
	ctx := newrelic.NewContext(context.Background(), txn)
	if err := c.Call(ctx, req, &rsp); nil != err {
		t.Fatal("Error calling test client:", err)
	}
	if !rsp.dtHeadersFound() {
		t.Error("Incorrect header:", rsp.RequestHeaders)
	}

	txn.End()
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/name", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/name", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/testing/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/testing/Micro/TestHandler.Method", Scope: "OtherTransaction/Go/name", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
	})
	app.(internal.Expect).ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category":      "generic",
				"name":          "OtherTransaction/Go/name",
				"nr.entryPoint": true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"category":  "http",
				"component": "Micro",
				"name":      "External/testing/Micro/TestHandler.Method",
				"parentId":  internal.MatchAnything,
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.(internal.Expect).ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/name",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/name",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "External/testing/Micro/TestHandler.Method",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}

func TestClientCallMetadata(t *testing.T) {
	// test that context metadata is not changed by the newrelic wrapper
	c, s := newTestClientAndServer(t)
	defer s.Stop()

	req := c.NewRequest("testing", "TestHandler.Method", &TestRequest{}, client.WithContentType("application/json"))
	rsp := TestResponse{}
	app := createTestApp(t)
	txn := app.StartTransaction("name", nil, nil)
	ctx := newrelic.NewContext(context.Background(), txn)
	md := metadata.Metadata{
		"zip": "zap",
	}
	ctx = metadata.NewContext(ctx, md)
	if err := c.Call(ctx, req, &rsp); nil != err {
		t.Fatal("Error calling test client:", err)
	}
	if len(md) != 1 || md["zip"] != "zap" {
		t.Error("metadata changed:", md)
	}
}
