package nrmicro

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/micro/go-micro/broker"
	bmemory "github.com/micro/go-micro/broker/memory"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/client/selector"
	"github.com/micro/go-micro/metadata"
	rmemory "github.com/micro/go-micro/registry/memory"
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

func dtHeadersFound(hdr string) bool {
	return hdr != "" && hdr != missingMetadata && hdr != missingHeaders
}

type TestHandler struct{}

func (t *TestHandler) Method(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	rsp.RequestHeaders = getDTRequestHeaderVal(ctx)
	return nil
}

func (t *TestHandler) StreamingMethod(ctx context.Context, stream server.Stream) error {
	if err := stream.Send(getDTRequestHeaderVal(ctx)); nil != err {
		return err
	}
	return nil
}

func getDTRequestHeaderVal(ctx context.Context) string {
	if md, ok := metadata.FromContext(ctx); ok {
		if dtHeader, ok := md[newrelic.DistributedTracePayloadHeader]; ok {
			return dtHeader
		}
		return missingHeaders
	}
	return missingMetadata
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

func newTestClientAndServer(wrapperOption client.Option, t *testing.T) (client.Client, server.Server) {
	registry := rmemory.NewRegistry()
	sel := selector.NewSelector(selector.Registry(registry))
	c := client.NewClient(
		client.Selector(sel),
		wrapperOption,
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
	c, s := newTestClientAndServer(client.Wrap(ClientWrapper()), t)
	defer s.Stop()
	testClientCallWithNoTransaction(c, t)
}

func TestClientCallWrapperWithNoTransaction(t *testing.T) {
	c, s := newTestClientAndServer(client.WrapCall(CallWrapper()), t)
	defer s.Stop()
	testClientCallWithNoTransaction(c, t)
}

func testClientCallWithNoTransaction(c client.Client, t *testing.T) {

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
	c, s := newTestClientAndServer(client.Wrap(ClientWrapper()), t)
	defer s.Stop()
	testClientCallWithTransaction(c, t)
}

func TestClientCallWrapperWithTransaction(t *testing.T) {
	c, s := newTestClientAndServer(client.WrapCall(CallWrapper()), t)
	defer s.Stop()
	testClientCallWithTransaction(c, t)
}

func testClientCallWithTransaction(c client.Client, t *testing.T) {

	req := c.NewRequest("testing", "TestHandler.Method", &TestRequest{}, client.WithContentType("application/json"))
	rsp := TestResponse{}
	app := createTestApp(t)
	txn := app.StartTransaction("name", nil, nil)
	ctx := newrelic.NewContext(context.Background(), txn)
	if err := c.Call(ctx, req, &rsp); nil != err {
		t.Fatal("Error calling test client:", err)
	}
	if !dtHeadersFound(rsp.RequestHeaders) {
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
	c, s := newTestClientAndServer(client.Wrap(ClientWrapper()), t)
	defer s.Stop()
	testClientCallMetadata(c, t)
}

func TestCallMetadata(t *testing.T) {
	c, s := newTestClientAndServer(client.WrapCall(CallWrapper()), t)
	defer s.Stop()
	testClientCallMetadata(c, t)
}

func testClientCallMetadata(c client.Client, t *testing.T) {
	// test that context metadata is not changed by the newrelic wrapper
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

func newTestClientAndBroker() (client.Client, broker.Broker) {
	b := bmemory.NewBroker()
	c := client.NewClient(
		client.Broker(b),
		client.Wrap(ClientWrapper()),
	)
	return c, b
}

func waitOrTimeout(t *testing.T, wg *sync.WaitGroup) {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		wg.Wait()
	}()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

var topic = "topic"

func TestClientPublishWithNoTransaction(t *testing.T) {
	c, b := newTestClientAndBroker()

	var wg sync.WaitGroup
	if err := b.Connect(); nil != err {
		t.Fatal("broker connect error:", err)
	}
	defer b.Disconnect()
	if _, err := b.Subscribe(topic, func(e broker.Event) error {
		defer wg.Done()
		h := e.Message().Header
		if _, ok := h[newrelic.DistributedTracePayloadHeader]; ok {
			t.Error("Distributed tracing headers found", h)
		}
		return nil
	}); nil != err {
		t.Fatal("Failure to subscribe to broker:", err)
	}

	ctx := context.Background()
	msg := c.NewMessage(topic, "hello world")
	wg.Add(1)
	if err := c.Publish(ctx, msg); nil != err {
		t.Fatal("Error calling test client:", err)
	}
	waitOrTimeout(t, &wg)
}

func TestClientPublishWithTransaction(t *testing.T) {
	c, b := newTestClientAndBroker()

	var wg sync.WaitGroup
	if err := b.Connect(); nil != err {
		t.Fatal("broker connect error:", err)
	}
	defer b.Disconnect()
	if _, err := b.Subscribe(topic, func(e broker.Event) error {
		defer wg.Done()
		h := e.Message().Header
		if _, ok := h[newrelic.DistributedTracePayloadHeader]; !ok {
			t.Error("Distributed tracing headers not found", h)
		}
		return nil
	}); nil != err {
		t.Fatal("Failure to subscribe to broker:", err)
	}

	app := createTestApp(t)
	txn := app.StartTransaction("name", nil, nil)
	ctx := newrelic.NewContext(context.Background(), txn)
	msg := c.NewMessage(topic, "hello world")
	wg.Add(1)
	if err := c.Publish(ctx, msg); nil != err {
		t.Fatal("Error calling test client:", err)
	}
	waitOrTimeout(t, &wg)

	txn.End()
	addr := b.Address()
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/" + addr + "/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/" + addr + "/Micro/Publish", Scope: "OtherTransaction/Go/name", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/name", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/name", Scope: "", Forced: false, Data: nil},
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
				"name":      "External/" + addr + "/Micro/Publish",
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
						SegmentName: "External/" + addr + "/Micro/Publish",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}

func TestExtractHost(t *testing.T) {
	testcases := []struct {
		input  string
		expect string
	}{
		{
			input:  "192.168.0.10",
			expect: "192.168.0.10",
		},
		{
			input:  "192.168.0.10:1234",
			expect: "192.168.0.10:1234",
		},
		{
			input:  "unix:///path/to/file",
			expect: "localhost",
		},
		{
			input:  "nats://127.0.0.1:4222",
			expect: "127.0.0.1:4222",
		},
		{
			input:  "scheme://user:pass@host.com:5432/path?k=v#f",
			expect: "host.com:5432",
		},
	}

	for _, test := range testcases {
		if actual := extractHost(test.input); actual != test.expect {
			t.Errorf("incorrect host value extracted: actual=%s expected=%s", actual, test.expect)
		}
	}
}

func TestClientStreamWrapperWithNoTransaction(t *testing.T) {
	c, s := newTestClientAndServer(client.Wrap(ClientWrapper()), t)
	defer s.Stop()

	ctx := context.Background()
	req := c.NewRequest(
		"testing",
		"TestHandler.StreamingMethod",
		&TestRequest{},
		client.WithContentType("application/json"),
		client.StreamingRequest(),
	)
	stream, err := c.Stream(ctx, req)
	defer stream.Close()
	if nil != err {
		t.Fatal("Error calling test client:", err)
	}

	var resp string
	err = stream.Recv(&resp)
	if nil != err {
		t.Fatal(err)
	}
	if dtHeadersFound(resp) {
		t.Error("dt headers found:", resp)
	}

	err = stream.Recv(&resp)
	if nil == err {
		t.Fatal("should have received EOF error from server")
	}
}

func TestClientStreamWrapperWithTransaction(t *testing.T) {
	c, s := newTestClientAndServer(client.Wrap(ClientWrapper()), t)
	defer s.Stop()

	app := createTestApp(t)
	txn := app.StartTransaction("name", nil, nil)
	ctx := newrelic.NewContext(context.Background(), txn)
	req := c.NewRequest(
		"testing",
		"TestHandler.StreamingMethod",
		&TestRequest{},
		client.WithContentType("application/json"),
		client.StreamingRequest(),
	)
	stream, err := c.Stream(ctx, req)
	defer stream.Close()
	if nil != err {
		t.Fatal("Error calling test client:", err)
	}

	var resp string
	// second outgoing request to server, ensures we only create a single
	// metric for the entire streaming cycle
	if err := stream.Send(&resp); nil != err {
		t.Fatal(err)
	}

	// receive the distributed trace headers from the server
	if err := stream.Recv(&resp); nil != err {
		t.Fatal(err)
	}
	if !dtHeadersFound(resp) {
		t.Error("dt headers not found:", resp)
	}

	// exhaust the stream
	if err := stream.Recv(&resp); nil == err {
		t.Fatal("should have received EOF error from server")
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
		{Name: "External/testing/Micro/TestHandler.StreamingMethod", Scope: "OtherTransaction/Go/name", Forced: false, Data: []float64{1}},
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
				"name":      "External/testing/Micro/TestHandler.StreamingMethod",
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
						SegmentName: "External/testing/Micro/TestHandler.StreamingMethod",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}
