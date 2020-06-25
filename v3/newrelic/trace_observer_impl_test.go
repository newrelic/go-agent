// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// +build go1.9
// This build tag is necessary because Infinite Tracing is only supported for Go version 1.9 and up

package newrelic

import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"github.com/newrelic/go-agent/v3/internal"
	v1 "github.com/newrelic/go-agent/v3/internal/com_newrelic_trace_v1"
	"github.com/newrelic/go-agent/v3/internal/logger"
)

// This file contains helper functions for Trace Observer tests

func expectSupportabilityMetrics(t *testing.T, to traceObserver, expected map[string]float64) {
	t.Helper()
	actual := to.dumpSupportabilityMetrics()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Supportability metrics do not match.\nExpected: %#v\nActual: %#v\n", expected, actual)
	}
}

func createServerAndObserver(t *testing.T) (testObsServer, traceObserver) {
	s := newTestObsServer(t, simpleRecordSpan)
	cfg := observerConfig{
		log:         logger.ShimLogger{},
		license:     testLicenseKey,
		queueSize:   20,
		appShutdown: make(chan struct{}),
		dialer:      s.dialer,
	}
	to, err := newTraceObserver(runToken, nil, cfg)
	if nil != err {
		t.Fatal(err)
	}
	waitForTrObs(t, to)
	return s, to
}

type recordSpanFunc func(*expectServer, v1.IngestService_RecordSpanServer) error

type expectServer struct {
	metadata metadata.MD
	sync.Mutex

	spansReceivedChan chan struct{}
	recordSpanFunc    recordSpanFunc

	v1.UnimplementedIngestServiceServer
}

func (s *expectServer) RecordSpan(stream v1.IngestService_RecordSpanServer) error {
	return s.recordSpanFunc(s, stream)
}

func simpleRecordSpan(s *expectServer, stream v1.IngestService_RecordSpanServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if ok {
		s.Lock()
		s.metadata = md
		s.Unlock()
	}
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			return nil
		} else if nil != err {
			return err
		}
		s.spansReceivedChan <- struct{}{}
	}
}

func (s *expectServer) ExpectMetadata(t *testing.T, want map[string]string) {
	t.Helper()
	s.Lock()
	actualMetadata := s.metadata
	s.Unlock()

	extraMetadata := map[string]string{
		":authority":   internal.MatchAnyString,
		"content-type": internal.MatchAnyString,
		"user-agent":   internal.MatchAnyString,
	}

	want = mergeMetadata(want, extraMetadata)

	if len(want) != len(actualMetadata) {
		t.Error("length of metadata is incorrect: expected/actual", len(want), len(actualMetadata))
		return
	}

	for key, expectedVal := range want {
		found, ok := actualMetadata[key]
		actualVal := strings.Join(found, ",")
		if !ok {
			t.Error("expected metadata not found: ", key)
			continue
		}
		if expectedVal == internal.MatchAnyString {
			continue
		}
		if actualVal != expectedVal {
			t.Error("metadata value difference - expected/actual",
				fmt.Sprintf("key=%s", key), expectedVal, actualVal)
		}
	}
	for key, val := range actualMetadata {
		_, ok := want[key]
		if !ok {
			t.Error("unexpected metadata present", key, val)
			continue
		}
	}
}

// Add the `extraMetadata` to each of the maps in the `want` parameter.
// The data in `want` takes precedence over the `extraMetadata`. If `want` is
// nil, returns nil.
func mergeMetadata(want map[string]string, extraMetadata map[string]string) map[string]string {
	if nil == want {
		return nil
	}
	newMap := make(map[string]string)
	for k, v := range extraMetadata {
		newMap[k] = v
	}
	for k, v := range want {
		newMap[k] = v
	}
	return newMap
}

// testObsServer contains an in-memory grpc.Server and associated information
// needed to connect to it and verify the data it receives
type testObsServer struct {
	*expectServer
	server *grpc.Server
	conn   *grpc.ClientConn
	dialer internal.DialerFunc
}

func (ts *testObsServer) Close() {
	ts.conn.Close()
	ts.server.Stop()
}

// newTestObsServer creates a new testObsServer for use in testing. Be sure
// to Close() the server when done with it.
func newTestObsServer(t *testing.T, fn recordSpanFunc) testObsServer {
	grpcServer := grpc.NewServer()
	s := &expectServer{
		// Hard coding the buffer to 10 for now, but it could be variable if needed later.
		spansReceivedChan: make(chan struct{}, 10),
		recordSpanFunc:    fn,
	}
	v1.RegisterIngestServiceServer(grpcServer, s)
	lis := bufconn.Listen(1024 * 1024)

	go grpcServer.Serve(lis)

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithInsecure(),
		grpc.WithBlock(), // create the connection synchronously
	)
	if err != nil {
		t.Fatal("failure to create ClientConn", err)
	}
	return testObsServer{
		expectServer: s,
		server:       grpcServer,
		conn:         conn,
		dialer:       bufDialer,
	}
}

// DidSpansArrive blocks until at least the expected number of spans arrives, or the timeout is reached.
// It returns whether or not the expected number of spans did, in fact, arrive.
func (s *expectServer) DidSpansArrive(t *testing.T, expected int, timeout time.Duration) bool {
	t.Helper()
	var rcvd int
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()
	for {
		select {
		case <-s.spansReceivedChan:
			rcvd++
			if rcvd >= expected {
				return true
			}
		case <-ticker.C:
			t.Logf("INFO: Waited for %d spans but received %d\n", expected, rcvd)
			return false
		}
	}
}

// testAppBlockOnTrObs is to be used when creating a test application that needs to block
// until the trace observer (which should be configured in the cfgfn) has connected.
func testAppBlockOnTrObs(replyfn func(*internal.ConnectReply), cfgfn func(*Config), t testing.TB) *expectApp {
	app := testApp(replyfn, cfgfn, t)
	app.app.connectTraceObserver(app.app.placeholderRun.Reply)
	waitForTrObs(t, app.app.trObserver)
	return &app
}

func waitForTrObs(t testing.TB, to traceObserver) {
	deadline := time.Now().Add(3 * time.Second)
	pollPeriod := 10 * time.Millisecond
	for {
		if to.initialConnCompleted() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("Error connecting to trace observer")
		}
		time.Sleep(pollPeriod)
	}
}

func DTReplyFieldsWithTrObsDialer(d internal.DialerFunc, runToken string) func(*internal.ConnectReply) {
	return func(reply *internal.ConnectReply) {
		distributedTracingReplyFields(reply)
		reply.RunID = internal.AgentRunID(runToken)
		reply.TraceObsDialer = d
	}
}

func toCfgWithTrObserver(cfg *Config) {
	cfg.CrossApplicationTracer.Enabled = false
	cfg.DistributedTracer.Enabled = true
	cfg.InfiniteTracing.TraceObserver.Host = "localhost"
}
