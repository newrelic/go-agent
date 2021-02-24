// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrmicro

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/micro/go-micro"
	"github.com/micro/go-micro/broker"
	bmemory "github.com/micro/go-micro/broker/memory"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/client/selector"
	microerrors "github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/metadata"
	rmemory "github.com/micro/go-micro/registry/memory"
	"github.com/micro/go-micro/server"

	proto "github.com/newrelic/go-agent/v4/integrations/nrmicro/example/proto"

	"github.com/newrelic/go-agent/v4/internal"
	"github.com/newrelic/go-agent/v4/internal/integrationsupport"
	"github.com/newrelic/go-agent/v4/newrelic"
)

const (
	missingHeaders  = "HEADERS NOT FOUND"
	missingMetadata = "METADATA NOT FOUND"
	serverName      = "testing"
	topic           = "topic"
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
	defer newrelic.FromContext(ctx).StartSegment("Method").End()
	return nil
}

func (t *TestHandler) StreamingMethod(ctx context.Context, stream server.Stream) error {
	if err := stream.Recv(new(string)); nil != err {
		return err
	}
	if err := stream.Send(getDTRequestHeaderVal(ctx)); nil != err {
		return err
	}
	return nil
}

type TestHandlerWithError struct{}

func (t *TestHandlerWithError) Method(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	rsp.RequestHeaders = getDTRequestHeaderVal(ctx)
	return microerrors.Unauthorized("id", "format")
}

type TestHandlerWithNonMicroError struct{}

func (t *TestHandlerWithNonMicroError) Method(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	rsp.RequestHeaders = getDTRequestHeaderVal(ctx)
	return errors.New("Non-Micro Error")
}

func getDTRequestHeaderVal(ctx context.Context) string {
	if md, ok := metadata.FromContext(ctx); ok {
		if dtHeader, ok := md[newrelic.DistributedTraceW3CTraceParentHeader]; ok {
			return dtHeader
		}
		return missingHeaders
	}
	return missingMetadata
}

func createTestApp() integrationsupport.ExpectApp {
	return integrationsupport.NewTestApp(cfgFn, integrationsupport.ConfigFullTraces)
}

var cfgFn = func(cfg *newrelic.Config) {
	cfg.Attributes.Include = append(cfg.Attributes.Include,
		newrelic.AttributeMessageRoutingKey,
		newrelic.AttributeMessageQueueName,
		newrelic.AttributeMessageExchangeType,
		newrelic.AttributeMessageReplyTo,
		newrelic.AttributeMessageCorrelationID,
	)
}

func newTestWrappedClientAndServer(app *newrelic.Application, wrapperOption client.Option, t *testing.T) (client.Client, server.Server) {
	registry := rmemory.NewRegistry()
	sel := selector.NewSelector(selector.Registry(registry))
	c := client.NewClient(
		client.Selector(sel),
		wrapperOption,
	)
	s := server.NewServer(
		server.Name(serverName),
		server.Registry(registry),
		server.WrapHandler(HandlerWrapper(app)),
	)
	s.Handle(s.NewHandler(new(TestHandler)))
	s.Handle(s.NewHandler(new(TestHandlerWithError)))
	s.Handle(s.NewHandler(new(TestHandlerWithNonMicroError)))

	if err := s.Start(); nil != err {
		t.Fatal(err)
	}
	return c, s
}

func TestClientCallWithNoTransaction(t *testing.T) {
	c, s := newTestWrappedClientAndServer(createTestApp().Application, client.Wrap(ClientWrapper()), t)
	defer s.Stop()
	testClientCallWithNoTransaction(c, t)
}

func TestClientCallWrapperWithNoTransaction(t *testing.T) {
	c, s := newTestWrappedClientAndServer(createTestApp().Application, client.WrapCall(CallWrapper()), t)
	defer s.Stop()
	testClientCallWithNoTransaction(c, t)
}

func testClientCallWithNoTransaction(c client.Client, t *testing.T) {

	ctx := context.Background()
	req := c.NewRequest(serverName, "TestHandler.Method", &TestRequest{}, client.WithContentType("application/json"))
	rsp := TestResponse{}
	if err := c.Call(ctx, req, &rsp); nil != err {
		t.Fatal("Error calling test client:", err)
	}
	if rsp.RequestHeaders != missingHeaders {
		t.Error("Header should not be here", rsp.RequestHeaders)
	}
}

func TestClientCallWithTransaction(t *testing.T) {
	c, s := newTestWrappedClientAndServer(createTestApp().Application, client.Wrap(ClientWrapper()), t)
	defer s.Stop()
	testClientCallWithTransaction(c, t)
}

func TestClientCallWrapperWithTransaction(t *testing.T) {
	c, s := newTestWrappedClientAndServer(createTestApp().Application, client.WrapCall(CallWrapper()), t)
	defer s.Stop()
	testClientCallWithTransaction(c, t)
}

func testClientCallWithTransaction(c client.Client, t *testing.T) {

	req := c.NewRequest(serverName, "TestHandler.Method", &TestRequest{}, client.WithContentType("application/json"))
	rsp := TestResponse{}
	app := createTestApp()
	txn := app.StartTransaction("name")
	ctx := newrelic.NewContext(context.Background(), txn)
	if err := c.Call(ctx, req, &rsp); nil != err {
		t.Fatal("Error calling test client:", err)
	}
	if !dtHeadersFound(rsp.RequestHeaders) {
		t.Error("Incorrect header:", rsp.RequestHeaders)
	}

	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/name", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/name", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/testing/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/testing/micro/TestHandler.Method", Scope: "OtherTransaction/Go/name", Forced: false, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:     "micro TestHandler.Method testing",
			ParentID: internal.MatchAnyParent,
			Attributes: map[string]interface{}{
				"http.component":   "micro",
				"http.method":      "TestHandler.Method",
				"http.status_code": int64(0),
				"http.url":         "unknown",
			},
		},
		{
			Name:       "name",
			ParentID:   internal.MatchNoParent,
			Attributes: map[string]interface{}{},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/name",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/name",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "External/testing/micro/TestHandler.Method",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}

func TestClientCallMetadata(t *testing.T) {
	c, s := newTestWrappedClientAndServer(createTestApp().Application, client.Wrap(ClientWrapper()), t)
	defer s.Stop()
	testClientCallMetadata(c, t)
}

func TestCallMetadata(t *testing.T) {
	c, s := newTestWrappedClientAndServer(createTestApp().Application, client.WrapCall(CallWrapper()), t)
	defer s.Stop()
	testClientCallMetadata(c, t)
}

func testClientCallMetadata(c client.Client, t *testing.T) {
	// test that context metadata is not changed by the newrelic wrapper
	req := c.NewRequest(serverName, "TestHandler.Method", &TestRequest{}, client.WithContentType("application/json"))
	rsp := TestResponse{}
	app := createTestApp()
	txn := app.StartTransaction("name")
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

func TestClientPublishWithNoTransaction(t *testing.T) {
	c, _, b := newTestClientServerAndBroker(createTestApp().Application, t)

	var wg sync.WaitGroup
	if err := b.Connect(); nil != err {
		t.Fatal("broker connect error:", err)
	}
	defer b.Disconnect()
	if _, err := b.Subscribe(topic, func(e broker.Event) error {
		defer wg.Done()
		h := e.Message().Header
		if _, ok := h[newrelic.DistributedTraceW3CTraceParentHeader]; ok {
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
	c, _, b := newTestClientServerAndBroker(createTestApp().Application, t)

	var wg sync.WaitGroup
	if err := b.Connect(); nil != err {
		t.Fatal("broker connect error:", err)
	}
	defer b.Disconnect()
	if _, err := b.Subscribe(topic, func(e broker.Event) error {
		defer wg.Done()
		h := e.Message().Header
		if _, ok := h[newrelic.DistributedTraceW3CTraceParentHeader]; !ok {
			t.Error("Distributed tracing headers not found", h)
		}
		return nil
	}); nil != err {
		t.Fatal("Failure to subscribe to broker:", err)
	}

	app := createTestApp()
	txn := app.StartTransaction("name")
	ctx := newrelic.NewContext(context.Background(), txn)
	msg := c.NewMessage(topic, "hello world")
	wg.Add(1)
	if err := c.Publish(ctx, msg); nil != err {
		t.Fatal("Error calling test client:", err)
	}
	waitOrTimeout(t, &wg)

	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "MessageBroker/micro/Topic/Produce/Named/topic", Scope: "", Forced: false, Data: nil},
		{Name: "MessageBroker/micro/Topic/Produce/Named/topic", Scope: "OtherTransaction/Go/name", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/name", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/name", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:     "topic send",
			ParentID: internal.MatchAnyParent,
			Attributes: map[string]interface{}{
				"messaging.destination":      "topic",
				"messaging.destination_kind": "topic",
				"messaging.system":           "micro",
			},
		},
		{
			Name:       "name",
			ParentID:   internal.MatchNoParent,
			Attributes: map[string]interface{}{},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/name",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/name",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "MessageBroker/micro/Topic/Produce/Named/topic",
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
	c, s := newTestWrappedClientAndServer(createTestApp().Application, client.Wrap(ClientWrapper()), t)
	defer s.Stop()

	ctx := context.Background()
	req := c.NewRequest(
		serverName,
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
	if err := stream.Send(&resp); nil != err {
		t.Fatal(err)
	}

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
	c, s := newTestWrappedClientAndServer(createTestApp().Application, client.Wrap(ClientWrapper()), t)
	defer s.Stop()

	app := createTestApp()
	txn := app.StartTransaction("name")
	ctx := newrelic.NewContext(context.Background(), txn)
	req := c.NewRequest(
		serverName,
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
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/name", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/name", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/testing/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/testing/micro/TestHandler.StreamingMethod", Scope: "OtherTransaction/Go/name", Forced: false, Data: []float64{1}},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:     "micro TestHandler.StreamingMethod testing",
			ParentID: internal.MatchAnyParent,
			Attributes: map[string]interface{}{
				"http.component":   "micro",
				"http.method":      "TestHandler.StreamingMethod",
				"http.status_code": int64(0),
				"http.url":         "unknown",
			},
		},
		{
			Name:       "name",
			ParentID:   internal.MatchNoParent,
			Attributes: map[string]interface{}{},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/name",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/name",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "External/testing/micro/TestHandler.StreamingMethod",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}

func TestServerWrapperWithNoApp(t *testing.T) {
	c, s := newTestWrappedClientAndServer(nil, client.Wrap(ClientWrapper()), t)
	defer s.Stop()
	ctx := context.Background()
	req := c.NewRequest(serverName, "TestHandler.Method", &TestRequest{}, client.WithContentType("application/json"))
	rsp := TestResponse{}
	if err := c.Call(ctx, req, &rsp); nil != err {
		t.Fatal("Error calling test client:", err)
	}
	if rsp.RequestHeaders != missingHeaders {
		t.Error("Header should not be here", rsp.RequestHeaders)
	}
}

func TestServerWrapperWithApp(t *testing.T) {
	app := createTestApp()
	c, s := newTestWrappedClientAndServer(app.Application, client.Wrap(ClientWrapper()), t)
	defer s.Stop()
	ctx := context.Background()
	txn := app.StartTransaction("txn")
	defer txn.End()
	ctx = newrelic.NewContext(ctx, txn)
	req := c.NewRequest(serverName, "TestHandler.Method", &TestRequest{}, client.WithContentType("application/json"))
	rsp := TestResponse{}
	if err := c.Call(ctx, req, &rsp); nil != err {
		t.Fatal("Error calling test client:", err)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "DurationByCaller/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/TestHandler.Method", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/TestHandler.Method", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestHandler.Method", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Method", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Method", Scope: "WebTransaction/Go/TestHandler.Method", Forced: false, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:       "Method",
			ParentID:   internal.MatchAnyParent,
			Attributes: map[string]interface{}{},
		},
		{
			Name:     "TestHandler.Method",
			ParentID: internal.MatchAnyParent,
			Attributes: map[string]interface{}{
				"http.method":                 "TestHandler.Method",
				"http.request_content_length": int64(3),
				"http.status_code":            int64(200),
				"http.url":                    "micro://testing/TestHandler.Method",
				"http.user_agent":             internal.MatchAnything,
			},
		},
		{
			Name:     "micro TestHandler.Method testing",
			ParentID: internal.MatchAnyParent,
			Attributes: map[string]interface{}{
				"http.component":   "micro",
				"http.method":      "TestHandler.Method",
				"http.status_code": int64(0),
				"http.url":         "unknown",
			},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "WebTransaction/Go/TestHandler.Method",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "WebTransaction/Go/TestHandler.Method",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "Custom/Method",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}

func TestServerWrapperWithAppReturnsError(t *testing.T) {
	app := createTestApp()
	c, s := newTestWrappedClientAndServer(app.Application, client.Wrap(ClientWrapper()), t)
	defer s.Stop()
	ctx := context.Background()
	req := c.NewRequest(serverName, "TestHandlerWithError.Method", &TestRequest{}, client.WithContentType("application/json"))
	rsp := TestResponse{}
	if err := c.Call(ctx, req, &rsp); nil == err {
		t.Fatal("Expected an error but did not get one")
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex/Go/TestHandlerWithError.Method", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/WebTransaction/Go/TestHandlerWithError.Method", Scope: "", Forced: true, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransaction/Go/TestHandlerWithError.Method", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestHandlerWithError.Method", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:       "TestHandlerWithError.Method",
			ParentID:   internal.MatchNoParent,
			StatusCode: 1,
			Attributes: map[string]interface{}{
				"http.method":                 "TestHandlerWithError.Method",
				"http.request_content_length": int64(3),
				"http.status_code":            int64(401),
				"http.url":                    "micro://testing/TestHandlerWithError.Method",
				"http.user_agent":             internal.MatchAnything,
			},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "WebTransaction/Go/TestHandlerWithError.Method",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "WebTransaction/Go/TestHandlerWithError.Method",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children:    []internal.WantTraceSegment{},
			}},
		},
	}})
	app.ExpectSpanEvents(t, []internal.WantSpan{{
		Name:       "TestHandlerWithError.Method",
		ParentID:   internal.MatchNoParent,
		StatusCode: 1,
		Attributes: map[string]interface{}{
			"http.method":                 "TestHandlerWithError.Method",
			"http.request_content_length": int64(3),
			"http.status_code":            int64(401),
			"http.url":                    "micro://testing/TestHandlerWithError.Method",
			"http.user_agent":             internal.MatchAnything,
		},
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/TestHandlerWithError.Method",
		Msg:     "Unauthorized",
		Klass:   "401",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.message":   "Unauthorized",
			"error.class":     "401",
			"transactionName": "WebTransaction/Go/TestHandlerWithError.Method",
			"traceId":         internal.MatchAnything,
			"priority":        internal.MatchAnything,
			"guid":            internal.MatchAnything,
			"spanId":          internal.MatchAnything,
			"sampled":         "true",
		},
	}})
}

func TestServerWrapperWithAppReturnsNonMicroError(t *testing.T) {
	app := createTestApp()
	c, s := newTestWrappedClientAndServer(app.Application, client.Wrap(ClientWrapper()), t)
	defer s.Stop()
	ctx := context.Background()
	req := c.NewRequest("testing", "TestHandlerWithNonMicroError.Method", &TestRequest{}, client.WithContentType("application/json"))
	rsp := TestResponse{}
	if err := c.Call(ctx, req, &rsp); nil == err {
		t.Fatal("Expected an error but did not get one")
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex/Go/TestHandlerWithNonMicroError.Method", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/WebTransaction/Go/TestHandlerWithNonMicroError.Method", Scope: "", Forced: true, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransaction/Go/TestHandlerWithNonMicroError.Method", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/TestHandlerWithNonMicroError.Method", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{{
		Name:       "TestHandlerWithNonMicroError.Method",
		ParentID:   internal.MatchNoParent,
		StatusCode: 1,
		Attributes: map[string]interface{}{
			"http.method":                 "TestHandlerWithNonMicroError.Method",
			"http.request_content_length": int64(3),
			"http.status_code":            int64(500),
			"http.url":                    "micro://testing/TestHandlerWithNonMicroError.Method",
			"http.user_agent":             internal.MatchAnything,
		},
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/TestHandlerWithNonMicroError.Method",
		Msg:     "Internal Server Error",
		Klass:   "500",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.message":   "Internal Server Error",
			"error.class":     "500",
			"transactionName": "WebTransaction/Go/TestHandlerWithNonMicroError.Method",
			"traceId":         internal.MatchAnything,
			"priority":        internal.MatchAnything,
			"guid":            internal.MatchAnything,
			"spanId":          internal.MatchAnything,
			"sampled":         "true",
		},
	}})
}

func TestServerSubscribeNoApp(t *testing.T) {
	c, s, b := newTestClientServerAndBroker(nil, t)
	defer s.Stop()

	var wg sync.WaitGroup
	if err := b.Connect(); nil != err {
		t.Fatal("broker connect error:", err)
	}
	defer b.Disconnect()
	err := micro.RegisterSubscriber(topic, s, func(ctx context.Context, msg *proto.HelloRequest) error {
		defer wg.Done()
		return nil
	})
	if err != nil {
		t.Fatal("error registering subscriber", err)
	}
	if err := s.Start(); nil != err {
		t.Fatal(err)
	}

	ctx := context.Background()
	msg := c.NewMessage(topic, &proto.HelloRequest{Name: "test"})
	wg.Add(1)
	if err := c.Publish(ctx, msg); nil != err {
		t.Fatal("Error calling publish:", err)
	}
	waitOrTimeout(t, &wg)
}

func TestServerSubscribe(t *testing.T) {
	app := createTestApp()
	c, s, _ := newTestClientServerAndBroker(app.Application, t)

	var wg sync.WaitGroup
	err := micro.RegisterSubscriber(topic, s, func(ctx context.Context, msg *proto.HelloRequest) error {
		txn := newrelic.FromContext(ctx)
		defer txn.StartSegment("segment").End()
		defer wg.Done()
		return nil
	})
	if err != nil {
		t.Fatal("error registering subscriber", err)
	}
	if err := s.Start(); nil != err {
		t.Fatal(err)
	}

	ctx := context.Background()
	msg := c.NewMessage(topic, &proto.HelloRequest{Name: "test"})
	wg.Add(1)
	txn := app.StartTransaction("pub")
	ctx = newrelic.NewContext(ctx, txn)
	if err := c.Publish(ctx, msg); nil != err {
		t.Fatal("Error calling publish:", err)
	}
	defer txn.End()
	waitOrTimeout(t, &wg)
	s.Stop()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/Message/micro/Topic/Named/topic", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/Message/micro/Topic/Named/topic", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/segment", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/segment", Scope: "OtherTransaction/Go/Message/micro/Topic/Named/topic", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:       "segment",
			ParentID:   internal.MatchAnyParent,
			Attributes: map[string]interface{}{},
		},
		{
			Name:     "topic receive",
			ParentID: internal.MatchAnyParent,
			Attributes: map[string]interface{}{
				"message.routingKey": "topic",
			},
		},
		{
			Name:     "topic send",
			ParentID: internal.MatchAnyParent,
			Attributes: map[string]interface{}{
				"messaging.destination":      "topic",
				"messaging.destination_kind": "topic",
				"messaging.system":           "micro",
			},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/Message/micro/Topic/Named/topic",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/Message/micro/Topic/Named/topic",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{{
					SegmentName: "Custom/segment",
					Attributes:  map[string]interface{}{},
					Children:    []internal.WantTraceSegment{}},
				},
			}},
		},
	}})
}

func TestServerSubscribeWithError(t *testing.T) {
	app := createTestApp()
	c, s, _ := newTestClientServerAndBroker(app.Application, t)

	var wg sync.WaitGroup
	err := micro.RegisterSubscriber(topic, s, func(ctx context.Context, msg *proto.HelloRequest) error {
		defer wg.Done()
		return errors.New("subscriber error")
	})
	if err != nil {
		t.Fatal("error registering subscriber", err)
	}
	if err := s.Start(); nil != err {
		t.Fatal(err)
	}

	ctx := context.Background()
	msg := c.NewMessage(topic, &proto.HelloRequest{Name: "test"})
	wg.Add(1)
	if err := c.Publish(ctx, msg); nil == err {
		t.Fatal("Expected error but didn't get one")
	}
	waitOrTimeout(t, &wg)
	s.Stop()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/Message/micro/Topic/Named/topic", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/Message/micro/Topic/Named/topic", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/OtherTransaction/Go/Message/micro/Topic/Named/topic", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:     "topic receive",
			ParentID: internal.MatchNoParent,
			Attributes: map[string]interface{}{
				"message.routingKey": "topic",
			},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/Message/micro/Topic/Named/topic",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/Message/micro/Topic/Named/topic",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children:    []internal.WantTraceSegment{},
			}},
		},
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/Message/micro/Topic/Named/topic",
		Msg:     "subscriber error",
		Klass:   "*errors.errorString",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.message":   "subscriber error",
			"error.class":     "*errors.errorString",
			"transactionName": "OtherTransaction/Go/Message/micro/Topic/Named/topic",
			"traceId":         internal.MatchAnything,
			"priority":        internal.MatchAnything,
			"guid":            internal.MatchAnything,
			"spanId":          internal.MatchAnything,
			"sampled":         "true",
		},
	}})
}

func newTestClientServerAndBroker(app *newrelic.Application, t *testing.T) (client.Client, server.Server, broker.Broker) {
	b := bmemory.NewBroker()
	c := client.NewClient(
		client.Broker(b),
		client.Wrap(ClientWrapper()),
	)
	s := server.NewServer(
		server.Name(serverName),
		server.Broker(b),
		server.WrapSubscriber(SubscriberWrapper(app)),
	)
	return c, s, b
}
