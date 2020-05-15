// +build go1.9
// This build tag is necessary because Infinite Tracing is only supported for Go version 1.9 and up

package newrelic

import (
	"fmt"
	"io"
	"math"
	"net"
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

// anySupportabilityCount indicates that we don't know/care what the value of the metric will be;
// it is math.Pi because that will never be an actual value of a support metric count
const anySupportabilityCount float64 = math.Pi

func expectSupportabilityMetrics(t *testing.T, to traceObserver, expected map[string]float64) {
	t.Helper()
	actual := to.dumpSupportabilityMetrics()
	if len(expected) != len(actual) {
		t.Errorf("Supportability metrics sizes do not match.\nExpected: %#v\nActual: %#v\n", expected, actual)
		return
	}
	for expectKey, expectVal := range expected {
		if actualVal, ok := actual[expectKey]; ok {
			if !supportMetricsMatch(expectVal, actualVal) {
				t.Errorf("Supportability metrics values do not match.\n"+
					"Key: %s\nExpected: %f\nActual: %f", expectKey, expectVal, actualVal)
			}
		} else {
			t.Errorf("Supportability metrics key not found in actual metrics: %s", expectKey)
		}
	}
}

func supportMetricsMatch(expectVal float64, actualVal float64) bool {
	return expectVal == anySupportabilityCount || expectVal == actualVal
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
	to, err := newTraceObserver(runToken, cfg)
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
	actualMetadataLen := len(s.metadata)
	s.Unlock()

	extraMetadata := map[string]string{
		":authority":   internal.MatchAnyString,
		"content-type": internal.MatchAnyString,
		"user-agent":   internal.MatchAnyString,
	}

	want = mergeMetadata(want, extraMetadata)

	if len(want) != actualMetadataLen {
		t.Error("length of metadata is incorrect: expected/actual", len(want), actualMetadataLen)
		return
	}

	s.Lock()
	actual := s.metadata
	s.Unlock()
	for key, expectedVal := range want {
		found, ok := actual[key]
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
	for key, val := range actual {
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

	bufDialer := func(string, time.Duration) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.Dial("bufnet",
		grpc.WithDialer(bufDialer),
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

func (s *expectServer) WaitForSpans(t *testing.T, expected int, secTimeout time.Duration) bool {
	t.Helper()
	var rcvd int
	timeout := time.NewTicker(secTimeout)
	defer timeout.Stop()
	for {
		select {
		case <-s.spansReceivedChan:
			rcvd++
			if rcvd >= expected {
				return true
			}
		case <-timeout.C:
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
	waitForTrObs(t, app.app.TraceObserver)
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
