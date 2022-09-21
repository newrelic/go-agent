// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrgrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/newrelic/go-agent/v3/integrations/nrgrpc/testapp"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"google.golang.org/grpc/metadata"
)

func TestGetURL(t *testing.T) {
	testcases := []struct {
		method   string
		target   string
		expected string
	}{
		{
			method:   "/TestApplication/DoUnaryUnary",
			target:   "",
			expected: "grpc:///TestApplication/DoUnaryUnary",
		},
		{
			method:   "TestApplication/DoUnaryUnary",
			target:   "",
			expected: "grpc://TestApplication/DoUnaryUnary",
		},
		{
			method:   "/TestApplication/DoUnaryUnary",
			target:   ":8080",
			expected: "grpc://:8080/TestApplication/DoUnaryUnary",
		},
		{
			method:   "/TestApplication/DoUnaryUnary",
			target:   "localhost:8080",
			expected: "grpc://localhost:8080/TestApplication/DoUnaryUnary",
		},
		{
			method:   "TestApplication/DoUnaryUnary",
			target:   "localhost:8080",
			expected: "grpc://localhost:8080/TestApplication/DoUnaryUnary",
		},
		{
			method:   "/TestApplication/DoUnaryUnary",
			target:   "dns:///localhost:8080",
			expected: "grpc://localhost:8080/TestApplication/DoUnaryUnary",
		},
		{
			method:   "/TestApplication/DoUnaryUnary",
			target:   "unix:/path/to/socket",
			expected: "grpc://localhost/TestApplication/DoUnaryUnary",
		},
		{
			method:   "/TestApplication/DoUnaryUnary",
			target:   "unix:///path/to/socket",
			expected: "grpc://localhost/TestApplication/DoUnaryUnary",
		},
	}

	for _, test := range testcases {
		actual := getURL(test.method, test.target)
		if actual.String() != test.expected {
			t.Errorf("incorrect URL:\n\tmethod=%s,\n\ttarget=%s,\n\texpected=%s,\n\tactual=%s",
				test.method, test.target, test.expected, actual.String())
		}
	}
}

func testApp() integrationsupport.ExpectApp {
	return integrationsupport.NewTestApp(replyFn, integrationsupport.ConfigFullTraces)
}

var replyFn = func(reply *internal.ConnectReply) {
	reply.SetSampleEverything()
	reply.AccountID = "123"
	reply.TrustedAccountKey = "123"
	reply.PrimaryAppID = "456"
}

func TestUnaryClientInterceptor(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction("UnaryUnary")
	ctx := newrelic.NewContext(context.Background(), txn)

	s, conn := newTestServerAndConn(t, nil)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	resp, err := client.DoUnaryUnary(ctx, &testapp.Message{})
	if err != nil {
		t.Fatal("client call to DoUnaryUnary failed", err)
	}
	var hdrs map[string][]string
	err = json.Unmarshal([]byte(resp.Text), &hdrs)
	if err != nil {
		t.Fatal("cannot unmarshall client response", err)
	}
	if hdr, ok := hdrs["newrelic"]; !ok || len(hdr) != 1 || hdr[0] == "" {
		t.Error("distributed trace header not sent", hdrs)
	}
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/UnaryUnary", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/UnaryUnary", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/bufnet/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/bufnet/gRPC/TestApplication/DoUnaryUnary", Scope: "OtherTransaction/Go/UnaryUnary", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category":  "http",
				"component": "gRPC",
				"name":      "External/bufnet/gRPC/TestApplication/DoUnaryUnary",
				"parentId":  internal.MatchAnything,
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"category":         "generic",
				"name":             "OtherTransaction/Go/UnaryUnary",
				"transaction.name": "OtherTransaction/Go/UnaryUnary",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/UnaryUnary",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/UnaryUnary",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "External/bufnet/gRPC/TestApplication/DoUnaryUnary",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}

func TestUnaryStreamClientInterceptor(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction("UnaryStream")
	ctx := newrelic.NewContext(context.Background(), txn)

	s, conn := newTestServerAndConn(t, nil)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	stream, err := client.DoUnaryStream(ctx, &testapp.Message{})
	if err != nil {
		t.Fatal("client call to DoUnaryStream failed", err)
	}
	var recved int
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal("error receiving message", err)
		}
		var hdrs map[string][]string
		err = json.Unmarshal([]byte(msg.Text), &hdrs)
		if err != nil {
			t.Fatal("cannot unmarshall client response", err)
		}
		if hdr, ok := hdrs["newrelic"]; !ok || len(hdr) != 1 || hdr[0] == "" {
			t.Error("distributed trace header not sent", hdrs)
		}
		recved++
	}
	if recved != 3 {
		t.Fatal("received incorrect number of messages from server", recved)
	}
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/UnaryStream", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/UnaryStream", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/bufnet/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/bufnet/gRPC/TestApplication/DoUnaryStream", Scope: "OtherTransaction/Go/UnaryStream", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category":  "http",
				"component": "gRPC",
				"name":      "External/bufnet/gRPC/TestApplication/DoUnaryStream",
				"parentId":  internal.MatchAnything,
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"category":         "generic",
				"name":             "OtherTransaction/Go/UnaryStream",
				"transaction.name": "OtherTransaction/Go/UnaryStream",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/UnaryStream",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/UnaryStream",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "External/bufnet/gRPC/TestApplication/DoUnaryStream",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}

func TestStreamUnaryClientInterceptor(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction("StreamUnary")
	ctx := newrelic.NewContext(context.Background(), txn)

	s, conn := newTestServerAndConn(t, nil)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
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
	msg, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatal("failure to CloseAndRecv", err)
	}
	var hdrs map[string][]string
	err = json.Unmarshal([]byte(msg.Text), &hdrs)
	if err != nil {
		t.Fatal("cannot unmarshall client response", err)
	}
	if hdr, ok := hdrs["newrelic"]; !ok || len(hdr) != 1 || hdr[0] == "" {
		t.Error("distributed trace header not sent", hdrs)
	}
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/StreamUnary", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/StreamUnary", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/bufnet/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/bufnet/gRPC/TestApplication/DoStreamUnary", Scope: "OtherTransaction/Go/StreamUnary", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category":  "http",
				"component": "gRPC",
				"name":      "External/bufnet/gRPC/TestApplication/DoStreamUnary",
				"parentId":  internal.MatchAnything,
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"category":         "generic",
				"name":             "OtherTransaction/Go/StreamUnary",
				"transaction.name": "OtherTransaction/Go/StreamUnary",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/StreamUnary",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/StreamUnary",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "External/bufnet/gRPC/TestApplication/DoStreamUnary",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}

func TestStreamStreamClientInterceptor(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction("StreamStream")
	ctx := newrelic.NewContext(context.Background(), txn)

	s, conn := newTestServerAndConn(t, nil)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	stream, err := client.DoStreamStream(ctx)
	if err != nil {
		t.Fatal("client call to DoStreamStream failed", err)
	}

	errC := make(chan error)
	go func(errC chan error) {
		var recved int
		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				errC <- fmt.Errorf("failure to Recv: %v", err)
				return
			}
			var hdrs map[string][]string
			err = json.Unmarshal([]byte(msg.Text), &hdrs)
			if err != nil {
				errC <- fmt.Errorf("cannot unmarshall client response: %v", err)
				return
			}
			if hdr, ok := hdrs["newrelic"]; !ok || len(hdr) != 1 || hdr[0] == "" {
				errC <- fmt.Errorf("distributed trace header not sent: %v", hdrs)
				return
			}
			recved++
		}
		if recved != 3 {
			errC <- fmt.Errorf("received incorrect number of messages from server: %v", recved)
			return
		}
		errC <- nil
	}(errC)
	for i := 0; i < 3; i++ {
		if err := stream.Send(&testapp.Message{Text: "Hello DoStreamStream"}); err != nil {
			t.Fatal("failure to Send", err)
		}
	}
	stream.CloseSend()

	err = <-errC
	close(errC)
	if err != nil {
		t.Fatal(err)
	}

	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/StreamStream", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/StreamStream", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/bufnet/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/bufnet/gRPC/TestApplication/DoStreamStream", Scope: "OtherTransaction/Go/StreamStream", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category":  "http",
				"component": "gRPC",
				"name":      "External/bufnet/gRPC/TestApplication/DoStreamStream",
				"parentId":  internal.MatchAnything,
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"category":         "generic",
				"name":             "OtherTransaction/Go/StreamStream",
				"transaction.name": "OtherTransaction/Go/StreamStream",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/StreamStream",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/StreamStream",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "External/bufnet/gRPC/TestApplication/DoStreamStream",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}

func TestClientUnaryMetadata(t *testing.T) {
	// Test that metadata on the outgoing request are presevered
	app := testApp()
	txn := app.StartTransaction("metadata")
	ctx := newrelic.NewContext(context.Background(), txn)

	md := metadata.New(map[string]string{
		"testing":  "hello world",
		"newrelic": "payload",
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	s, conn := newTestServerAndConn(t, nil)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	resp, err := client.DoUnaryUnary(ctx, &testapp.Message{})
	if err != nil {
		t.Fatal("client call to DoUnaryUnary failed", err)
	}
	var hdrs map[string][]string
	err = json.Unmarshal([]byte(resp.Text), &hdrs)
	if err != nil {
		t.Fatal("cannot unmarshall client response", err)
	}
	if hdr, ok := hdrs["newrelic"]; !ok || len(hdr) != 1 || hdr[0] == "" || hdr[0] == "payload" {
		t.Error("distributed trace header not sent", hdrs)
	}
	if hdr, ok := hdrs["testing"]; !ok || len(hdr) != 1 || hdr[0] != "hello world" {
		t.Error("testing header not sent", hdrs)
	}
}

func TestNilTxnClientUnary(t *testing.T) {
	s, conn := newTestServerAndConn(t, nil)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)
	resp, err := client.DoUnaryUnary(context.Background(), &testapp.Message{})
	if err != nil {
		t.Fatal("client call to DoUnaryUnary failed", err)
	}
	var hdrs map[string][]string
	err = json.Unmarshal([]byte(resp.Text), &hdrs)
	if err != nil {
		t.Fatal("cannot unmarshall client response", err)
	}
	if _, ok := hdrs["newrelic"]; ok {
		t.Error("distributed trace header sent", hdrs)
	}
}

func TestNilTxnClientStreaming(t *testing.T) {
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
	var hdrs map[string][]string
	err = json.Unmarshal([]byte(msg.Text), &hdrs)
	if err != nil {
		t.Fatal("cannot unmarshall client response", err)
	}
	if _, ok := hdrs["newrelic"]; ok {
		t.Error("distributed trace header sent", hdrs)
	}
}

func TestClientStreamingError(t *testing.T) {
	// Test that when creating the stream returns an error, no external
	// segments are created
	app := testApp()
	txn := app.StartTransaction("UnaryStream")

	s, conn := newTestServerAndConn(t, nil)
	defer s.Stop()
	defer conn.Close()

	client := testapp.NewTestApplicationClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	ctx = newrelic.NewContext(ctx, txn)
	_, err := client.DoUnaryStream(ctx, &testapp.Message{})
	if err == nil {
		t.Fatal("client call to DoUnaryStream did not return error")
	}
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/UnaryStream", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/UnaryStream", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category":         "generic",
				"name":             "OtherTransaction/Go/UnaryStream",
				"transaction.name": "OtherTransaction/Go/UnaryStream",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/UnaryStream",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/UnaryStream",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children:    []internal.WantTraceSegment{},
			}},
		},
	}})
}
