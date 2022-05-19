// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build go1.9
// +build go1.9

// This build tag is necessary because Infinite Tracing is only supported for Go version 1.9 and up

package newrelic

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/newrelic/go-agent/v3/internal"
	v1 "github.com/newrelic/go-agent/v3/internal/com_newrelic_trace_v1"
	"github.com/newrelic/go-agent/v3/internal/logger"
)

func TestValidateTraceObserverURL(t *testing.T) {
	testcases := []struct {
		inputHost string
		inputPort int
		expectErr bool
		expectURL *observerURL
	}{
		{
			inputHost: "",
			expectErr: false,
			expectURL: nil,
		},
		{
			inputHost: "testing.com",
			expectErr: false,
			expectURL: &observerURL{
				host:   "testing.com:443",
				secure: true,
			},
		},
		{
			inputHost: "1.2.3.4",
			expectErr: false,
			expectURL: &observerURL{
				host:   "1.2.3.4:443",
				secure: true,
			},
		},
		{
			inputHost: "testing.com",
			inputPort: 123,
			expectErr: false,
			expectURL: &observerURL{
				host:   "testing.com:123",
				secure: true,
			},
		},
		{
			inputHost: "localhost",
			inputPort: 8080,
			expectErr: false,
			expectURL: &observerURL{
				host:   "localhost:8080",
				secure: false,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.inputHost, func(t *testing.T) {
			c := defaultConfig()
			c.DistributedTracer.Enabled = true
			c.SpanEvents.Enabled = true
			c.InfiniteTracing.TraceObserver.Host = tc.inputHost
			if tc.inputPort != 0 {
				c.InfiniteTracing.TraceObserver.Port = tc.inputPort
			}
			url, err := c.validateTraceObserverConfig()

			if tc.expectErr && err == nil {
				t.Error("expected error, received nil")
			} else if !tc.expectErr && err != nil {
				t.Errorf("expected no error, but got one: %s", err)
			}

			if !reflect.DeepEqual(url, tc.expectURL) {
				t.Errorf("url is not as expected: actual=%#v expect=%#v", url, tc.expectURL)
			}
		})
	}
}

func Test8TConfig(t *testing.T) {
	testcases := []struct {
		host         string
		spansEnabled bool
		DTEnabled    bool
		validConfig  bool
	}{
		{
			host:         "localhost",
			spansEnabled: true,
			DTEnabled:    true,
			validConfig:  true,
		},
		{
			host:         "localhost",
			spansEnabled: false,
			DTEnabled:    true,
			validConfig:  false,
		},
		{
			host:         "localhost",
			spansEnabled: true,
			DTEnabled:    false,
			validConfig:  false,
		},
		{
			host:         "localhost",
			spansEnabled: false,
			DTEnabled:    false,
			validConfig:  false,
		},
		{
			host:         "",
			spansEnabled: false,
			DTEnabled:    false,
			validConfig:  true,
		},
	}

	for _, test := range testcases {
		cfg := Config{}
		cfg.License = testLicenseKey
		cfg.AppName = "app"
		cfg.InfiniteTracing.TraceObserver.Host = test.host
		cfg.SpanEvents.Enabled = test.spansEnabled
		cfg.DistributedTracer.Enabled = test.DTEnabled

		_, err := newInternalConfig(cfg, func(s string) string { return "" }, []string{})
		if (err == nil) != test.validConfig {
			t.Errorf("Infite Tracing config validation failed: %v", test)
		}
	}
}

func TestTraceObserverErrToCodeString(t *testing.T) {
	// if the grpc code names change upstream, this test will alert us to that
	testcases := []struct {
		code   codes.Code
		expect string
	}{
		{code: 0, expect: "OK"},
		{code: 1, expect: "CANCELLED"},
		{code: 2, expect: "UNKNOWN"},
		{code: 3, expect: "INVALID_ARGUMENT"},
		{code: 4, expect: "DEADLINE_EXCEEDED"},
		{code: 5, expect: "NOT_FOUND"},
		{code: 6, expect: "ALREADY_EXISTS"},
		{code: 7, expect: "PERMISSION_DENIED"},
		{code: 8, expect: "RESOURCE_EXHAUSTED"},
		{code: 9, expect: "FAILED_PRECONDITION"},
		{code: 10, expect: "ABORTED"},
		{code: 11, expect: "OUT_OF_RANGE"},
		{code: 12, expect: "UNIMPLEMENTED"},
		{code: 13, expect: "INTERNAL"},
		{code: 14, expect: "UNAVAILABLE"},
		{code: 15, expect: "DATA_LOSS"},
		{code: 16, expect: "UNAUTHENTICATED"},
		// we should always test one more than the number of codes supported by
		// grpc so we can detect when a new code is added
		{code: 17, expect: "CODE(17)"},
	}
	for _, test := range testcases {
		t.Run(test.expect, func(t *testing.T) {
			err := status.Error(test.code, "oops")
			actual := errToCodeString(err)
			if actual != test.expect {
				t.Errorf("incorrect error string returned: actual=%s expected=%s",
					actual, test.expect)
			}
		})
	}
}

type mockClient struct {
	sendResponse error
	v1.IngestService_RecordSpanClient
}

func (c mockClient) Send(*v1.Span) error {
	return c.sendResponse
}

func TestSendSpanMetrics(t *testing.T) {
	appShutdown := make(chan struct{})
	to := &gRPCtraceObserver{
		supportability: newObserverSupport(),
		observerConfig: observerConfig{
			log:         logger.ShimLogger{},
			appShutdown: appShutdown,
		},
	}
	go to.handleSupportability()
	defer close(appShutdown)
	clientWithError := mockClient{
		sendResponse: errPermissionDenied,
	}
	clientWithoutError := mockClient{
		sendResponse: nil,
	}

	// The Seen count will be 0 for each example in this test because Seen is
	// incremented during consumeSpan which is never called here.
	expectSupportabilityMetrics(t, to, map[string]float64{
		"Supportability/InfiniteTracing/Span/Seen": 0,
		"Supportability/InfiniteTracing/Span/Sent": 0,
	})

	if err := to.sendSpan(clientWithError, &spanEvent{}); err == nil {
		t.Error("spendSpan should have returned an error when Send returns an error")
	}
	expectSupportabilityMetrics(t, to, map[string]float64{
		"Supportability/InfiniteTracing/Span/Response/Error":         1,
		"Supportability/InfiniteTracing/Span/Seen":                   0,
		"Supportability/InfiniteTracing/Span/Sent":                   1,
		"Supportability/InfiniteTracing/Span/gRPC/PERMISSION_DENIED": 1,
	})

	if err := to.sendSpan(clientWithoutError, &spanEvent{}); err != nil {
		t.Error("spendSpan should not have returned an error when Send returns a nil error")
	}
	expectSupportabilityMetrics(t, to, map[string]float64{
		"Supportability/InfiniteTracing/Span/Seen": 0,
		"Supportability/InfiniteTracing/Span/Sent": 1,
	})
}

const runToken = "aRunToken"

func TestTraceObserverRestart(t *testing.T) {
	s := newTestObsServer(t, simpleRecordSpan)
	cfg := observerConfig{
		log:         logger.ShimLogger{},
		license:     testLicenseKey,
		queueSize:   20,
		appShutdown: make(chan struct{}),
		dialer:      s.dialer,
	}
	to, err := newTraceObserver(runToken, map[string]string{"INITIAL": "VALUE1"}, cfg)
	if nil != err {
		t.Fatal(err)
	}
	waitForTrObs(t, to)
	defer s.Close()

	// Make sure the server has received the new data
	to.consumeSpan(&spanEvent{})
	if !s.DidSpansArrive(t, 1, 150*time.Millisecond) {
		t.Error("Did not receive expected spans before timeout -- before restart")
	}

	s.ExpectMetadata(t, map[string]string{
		"agent_run_token": runToken,
		"license_key":     testLicenseKey,
		"initial":         "VALUE1",
	})
	newToken := "aNewRunToken"
	to.restart(internal.AgentRunID(newToken), map[string]string{"RESTART": "VALUE2"})

	// Make sure the server has received the new data
	to.consumeSpan(&spanEvent{})
	if !s.DidSpansArrive(t, 1, 150*time.Millisecond) {
		t.Error("Did not receive expected spans before timeout -- after restart")
	}

	s.ExpectMetadata(t, map[string]string{
		"agent_run_token": newToken,
		"license_key":     testLicenseKey,
		"restart":         "VALUE2",
	})
}

func TestTraceObserverShutdown(t *testing.T) {
	s, to := createServerAndObserver(t)

	s.ExpectMetadata(t, map[string]string{
		"agent_run_token": runToken,
		"license_key":     testLicenseKey,
	})
	if err := to.shutdown(time.Second); err != nil {
		t.Fatal(err)
	}
	to.consumeSpan(&spanEvent{})
	if s.DidSpansArrive(t, 1, 50*time.Millisecond) {
		t.Error("Got a span we did not expect to get")
	}
	s.Close()

	shutdownApp(to)

	to.consumeSpan(&spanEvent{})
	if s.DidSpansArrive(t, 1, 50*time.Millisecond) {
		t.Error("Got a span we did not expect to get")
	}
}

// shutdownApp simulates the whole app shutting down
func shutdownApp(to traceObserver) {
	close(to.(*gRPCtraceObserver).appShutdown)
}

func TestTraceObserverConsumeSpan(t *testing.T) {
	s, to := createServerAndObserver(t)
	defer s.Close()

	s.ExpectMetadata(t, map[string]string{
		"agent_run_token": runToken,
		"license_key":     testLicenseKey,
	})
	to.consumeSpan(&spanEvent{})
	to.consumeSpan(&spanEvent{})

	if !s.DidSpansArrive(t, 2, 50*time.Millisecond) {
		t.Error("Did not receive expected spans before timeout")
	}
}

func TestTraceObserverDumpSupportabilityMetrics(t *testing.T) {
	s, to := createServerAndObserver(t)
	defer s.Close()

	expectSupportabilityMetrics(t, to, map[string]float64{
		"Supportability/InfiniteTracing/Span/Seen": 0,
		"Supportability/InfiniteTracing/Span/Sent": 0,
	})

	to.consumeSpan(&spanEvent{})
	if !s.DidSpansArrive(t, 1, 50*time.Millisecond) {
		t.Error("Did not receive expected spans before timeout")
	}

	expectSupportabilityMetrics(t, to, map[string]float64{
		"Supportability/InfiniteTracing/Span/Seen": 1,
		"Supportability/InfiniteTracing/Span/Sent": 1,
	})

	// Ensure counts are reset
	expectSupportabilityMetrics(t, to, map[string]float64{
		"Supportability/InfiniteTracing/Span/Seen": 0,
		"Supportability/InfiniteTracing/Span/Sent": 0,
	})
}

func TestTraceObserverConnected(t *testing.T) {
	s := newTestObsServer(t, simpleRecordSpan)
	defer s.Close()
	readyChan := make(chan struct{})
	slowDialer := func(ctx context.Context, str string) (net.Conn, error) {
		<-readyChan
		return s.dialer(ctx, str)
	}
	cfg := observerConfig{
		log:         logger.ShimLogger{},
		license:     testLicenseKey,
		queueSize:   20,
		appShutdown: make(chan struct{}),
		dialer:      slowDialer,
	}
	to, err := newTraceObserver(runToken, nil, cfg)
	if nil != err {
		t.Fatal(err)
	}

	if to.initialConnCompleted() {
		t.Error("Didn't expect the trace observer to be connected, but it is")
	}
	readyChan <- struct{}{}
	waitForTrObs(t, to)

	if !to.initialConnCompleted() {
		t.Error("Expected the trace observer to be connected, but it isn't")
	}
}

func TestTrObsMultipleShutdowns(t *testing.T) {
	s, to := createServerAndObserver(t)
	defer s.Close()
	waitForTrObs(t, to)

	if err := to.shutdown(time.Second); err != nil {
		t.Fatal(err)
	}

	// Make sure we don't panic
	if err := to.shutdown(time.Second); err != nil {
		t.Error("error shutting down the trace observer:", err)
	}

	shutdownApp(to)
	// Make sure we don't panic
	if err := to.shutdown(time.Second); err != nil {
		t.Error("error shutting downt the trace observer after shutting down app:", err)
	}
}

func TestTrObsShutdownAndRestart(t *testing.T) {
	s, to := createServerAndObserver(t)
	defer s.Close()
	waitForTrObs(t, to)

	if err := to.shutdown(time.Second); err != nil {
		t.Fatal(err)
	}

	// Make sure we don't panic and don't send updated metadata
	to.restart("A New Run Token", map[string]string{"hello": "world"})
	s.ExpectMetadata(t, map[string]string{
		"agent_run_token": runToken,
		"license_key":     testLicenseKey,
	})

	shutdownApp(to)

	// Make sure we don't panic and don't send updated metadata
	to.restart("A New Run Token", map[string]string{"hello": "world"})
	s.ExpectMetadata(t, map[string]string{
		"agent_run_token": runToken,
		"license_key":     testLicenseKey,
	})
}

func TestTrObsShutdownAndInitialConnSuccessful(t *testing.T) {
	s, to := createServerAndObserver(t)
	defer s.Close()
	waitForTrObs(t, to)

	if err := to.shutdown(time.Second); err != nil {
		t.Fatal(err)
	}

	if !to.initialConnCompleted() {
		t.Error("Expected the initialConnCompleted call to return true after shutdown, " +
			"but returned false")
	}

	shutdownApp(to)

	if !to.initialConnCompleted() {
		t.Error("Expected the initialConnCompleted call to return true after app shutdown, " +
			"but returned false")
	}
}

func TestTrObsShutdownAndDumpSupportabilityMetrics(t *testing.T) {
	s, to := createServerAndObserver(t)
	defer s.Close()

	if err := to.shutdown(time.Second); err != nil {
		t.Fatal(err)
	}

	expectSupportabilityMetrics(t, to, map[string]float64{
		"Supportability/InfiniteTracing/Span/Seen": 0,
		"Supportability/InfiniteTracing/Span/Sent": 0,
		// the error metrics are from the EOF on the client.Recv
		"Supportability/InfiniteTracing/Span/Response/Error": 1,
		"Supportability/InfiniteTracing/Span/gRPC/UNKNOWN":   1,
	})

	shutdownApp(to)

	expectSupportabilityMetrics(t, to, nil)
}

func TestTrObsSlowConnectAndRestart(t *testing.T) {
	s := newTestObsServer(t, simpleRecordSpan)
	defer s.Close()
	readyChan := make(chan struct{})
	slowDialer := func(ctx context.Context, str string) (net.Conn, error) {
		<-readyChan
		return s.dialer(ctx, str)
	}
	cfg := observerConfig{
		log:         logger.ShimLogger{},
		license:     testLicenseKey,
		queueSize:   20,
		appShutdown: make(chan struct{}),
		dialer:      slowDialer,
	}
	to, err := newTraceObserver(runToken, map[string]string{"INITIAL": "ONE"}, cfg)
	if nil != err {
		t.Fatal(err)
	}

	newToken := "A New Run Token"
	to.restart(internal.AgentRunID(newToken), map[string]string{"RESTART": "TWO"})
	if s.DidSpansArrive(t, 1, 50*time.Millisecond) {
		t.Error("Got a span we did not expect to get")
	}
	s.ExpectMetadata(t, nil)

	close(readyChan)
	if s.DidSpansArrive(t, 1, 500*time.Millisecond) {
		t.Error("Got a span we did not expect to get")
	}
	s.ExpectMetadata(t, map[string]string{
		"agent_run_token": newToken,
		"license_key":     testLicenseKey,
		"restart":         "TWO",
	})
}

func TestTrObsSlowConnectAndConsumeSpan(t *testing.T) {
	s := newTestObsServer(t, simpleRecordSpan)
	defer s.Close()
	readyChan := make(chan struct{})
	slowDialer := func(ctx context.Context, str string) (net.Conn, error) {
		<-readyChan
		return s.dialer(ctx, str)
	}
	cfg := observerConfig{
		log:         logger.ShimLogger{},
		license:     testLicenseKey,
		queueSize:   20,
		appShutdown: make(chan struct{}),
		dialer:      slowDialer,
	}
	to, err := newTraceObserver(runToken, nil, cfg)
	if nil != err {
		t.Fatal(err)
	}

	to.consumeSpan(&spanEvent{})
	if s.DidSpansArrive(t, 1, 50*time.Millisecond) {
		t.Error("Got a span we did not expect to get")
	}

	close(readyChan)
	if !s.DidSpansArrive(t, 1, 50*time.Millisecond) {
		t.Error("Did not receive expected spans before timeout")
	}
}

func TestTrObsSlowConnectAndDumpSupportabilityMetrics(t *testing.T) {
	s := newTestObsServer(t, simpleRecordSpan)
	defer s.Close()
	readyChan := make(chan struct{})
	slowDialer := func(ctx context.Context, str string) (net.Conn, error) {
		<-readyChan
		return s.dialer(ctx, str)
	}
	cfg := observerConfig{
		log:         logger.ShimLogger{},
		license:     testLicenseKey,
		queueSize:   20,
		appShutdown: make(chan struct{}),
		dialer:      slowDialer,
	}
	to, err := newTraceObserver(runToken, nil, cfg)
	if nil != err {
		t.Fatal(err)
	}

	expectSupportabilityMetrics(t, to, map[string]float64{
		"Supportability/InfiniteTracing/Span/Seen": 0,
		"Supportability/InfiniteTracing/Span/Sent": 0,
	})

	to.consumeSpan(&spanEvent{})
	expectSupportabilityMetrics(t, to, map[string]float64{
		"Supportability/InfiniteTracing/Span/Seen": 1,
		"Supportability/InfiniteTracing/Span/Sent": 0,
	})

	close(readyChan)
	if !s.DidSpansArrive(t, 1, 50*time.Millisecond) {
		t.Error("Did not receive expected spans before timeout")
	}
	expectSupportabilityMetrics(t, to, map[string]float64{
		"Supportability/InfiniteTracing/Span/Seen": 0,
		"Supportability/InfiniteTracing/Span/Sent": 1,
	})
}

func toIsShutdown(to traceObserver) bool {
	// This sleep is so long because it is waiting on the deferred 500
	// millisecond sleep for closing the grpc conn.
	time.Sleep(550 * time.Millisecond)
	return to.(*gRPCtraceObserver).isShutdownComplete()
}

func TestTrObsSlowConnectAndShutdown(t *testing.T) {
	s := newTestObsServer(t, simpleRecordSpan)
	defer s.Close()
	readyChan := make(chan struct{})
	slowDialer := func(ctx context.Context, str string) (net.Conn, error) {
		<-readyChan
		return s.dialer(ctx, str)
	}
	cfg := observerConfig{
		log:         logger.ShimLogger{},
		license:     testLicenseKey,
		queueSize:   20,
		appShutdown: make(chan struct{}),
		dialer:      slowDialer,
	}
	to, err := newTraceObserver(runToken, nil, cfg)
	if nil != err {
		t.Fatal(err)
	}

	to.consumeSpan(&spanEvent{})

	if err := to.shutdown(time.Nanosecond); err == nil {
		t.Error("trace observer was able to shutdown when it shouldn't have")
	}

	close(readyChan)

	if !toIsShutdown(to) {
		t.Error("trace observer should be shutdown but it is not")
	}
	if !s.DidSpansArrive(t, 1, 50*time.Millisecond) {
		t.Error("span was not received")
	}
}

var (
	errUnimplemented    = status.Error(codes.Unimplemented, "unimplemented")
	errPermissionDenied = status.Error(codes.PermissionDenied, "I'm so sorry")
	errOK               = status.Error(codes.OK, "okay okay okay") // grpc turns this into nil
)

func TestTrObsRecordSpanReturnsError(t *testing.T) {
	s := newTestObsServer(t, simpleRecordSpan)
	defer s.Close()
	errDialer := func(context.Context, string) (net.Conn, error) {
		// It doesn't matter what error is returned here, grpc will translate
		// this into a code 14 error. This error is returned from RecordSpan
		// and since it is not an Unimplemented error, we will not shut down.
		return nil, errors.New("ooooops")
	}
	cfg := observerConfig{
		log:         logger.ShimLogger{},
		license:     testLicenseKey,
		queueSize:   20,
		appShutdown: make(chan struct{}),
		dialer:      errDialer,
	}
	to, err := newTraceObserver(runToken, nil, cfg)
	if nil != err {
		t.Fatal(err)
	}

	if toIsShutdown(to) {
		t.Error("trace observer should not be shutdown but it is")
	}
}

func TestTrObsRecvReturnsUnimplementedError(t *testing.T) {
	s := newTestObsServer(t, func(s *expectServer, stream v1.IngestService_RecordSpanServer) error {
		return errUnimplemented
	})
	defer s.Close()
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

	if !toIsShutdown(to) {
		t.Error("trace observer should be shutdown but it is not")
	}
}

func TestTrObsRecvReturnsOtherError(t *testing.T) {
	s := newTestObsServer(t, func(s *expectServer, stream v1.IngestService_RecordSpanServer) error {
		return errPermissionDenied
	})
	defer s.Close()
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

	if toIsShutdown(to) {
		t.Error("trace observer should not be shutdown but it is")
	}
}

func TestTrObsUnimplementedNoMoreSpansSent(t *testing.T) {
	s := newTestObsServer(t, func(s *expectServer, stream v1.IngestService_RecordSpanServer) error {
		stream.Recv()
		s.spansReceivedChan <- struct{}{}
		return errUnimplemented
	})
	cfg := observerConfig{
		log:           logger.ShimLogger{},
		license:       testLicenseKey,
		queueSize:     20,
		appShutdown:   make(chan struct{}),
		dialer:        s.dialer,
		removeBackoff: true,
	}
	to, err := newTraceObserver(runToken, nil, cfg)
	if nil != err {
		t.Fatal(err)
	}
	waitForTrObs(t, to)

	// First span should cause a shutdown to initiate;
	// the others should get queued but may or may not be not sent
	to.consumeSpan(&spanEvent{})
	to.consumeSpan(&spanEvent{})
	to.consumeSpan(&spanEvent{})

	if !s.DidSpansArrive(t, 1, time.Second) {
		t.Error("Did not receive expected span before timeout")
	}

	if !toIsShutdown(to) {
		t.Error("trace observer should be shutdown but it is not")
	}

	// Closing the server ensures that if a span was sent that it will be
	// received and read by the server
	s.Close()

	// Additional spans should not be delivered
	if s.DidSpansArrive(t, 1, 100*time.Millisecond) {
		t.Error("Received 1 spans after shutdown when we should not receive any")
	}
}

func TestTrObsPermissionDeniedMoreSpansSent(t *testing.T) {
	s := newTestObsServer(t, func(s *expectServer, stream v1.IngestService_RecordSpanServer) error {
		stream.Recv()
		s.spansReceivedChan <- struct{}{}
		return errPermissionDenied
	})
	cfg := observerConfig{
		log:           logger.ShimLogger{},
		license:       testLicenseKey,
		queueSize:     20,
		appShutdown:   make(chan struct{}),
		dialer:        s.dialer,
		removeBackoff: true,
	}
	to, err := newTraceObserver(runToken, nil, cfg)
	if nil != err {
		t.Fatal(err)
	}
	waitForTrObs(t, to)

	to.consumeSpan(&spanEvent{})
	to.consumeSpan(&spanEvent{})

	if !s.DidSpansArrive(t, 1, time.Second) {
		t.Error("Did not receive expected span before timeout")
	}

	if toIsShutdown(to) {
		t.Error("trace observer should not be shutdown but it is")
	}

	// Closing the server ensures that if a span was sent that it will be
	// received and read by the server
	s.Close()

	// Additional spans should be delivered
	if !s.DidSpansArrive(t, 1, time.Second) {
		t.Error("did not receive 1 expected spans")
	}
}

func TestTrObsDrainsMessagesOnShutdown(t *testing.T) {
	s := newTestObsServer(t, func(s *expectServer, stream v1.IngestService_RecordSpanServer) error {
		return errUnimplemented
	})
	defer s.Close()
	readyChan := make(chan struct{})
	slowDialer := func(ctx context.Context, str string) (net.Conn, error) {
		<-readyChan
		return s.dialer(ctx, str)
	}
	cfg := observerConfig{
		log:         logger.ShimLogger{},
		license:     testLicenseKey,
		queueSize:   20,
		appShutdown: make(chan struct{}),
		dialer:      slowDialer,
	}
	to, err := newTraceObserver(runToken, nil, cfg)
	if nil != err {
		t.Fatal(err)
	}

	numMsgs := func() int {
		return len(to.(*gRPCtraceObserver).messages)
	}

	for i := 0; i < 20; i++ {
		// We must consume a significant number of spans here because between
		// 2-5 of them will be sent before the unimplemented error is received.
		to.consumeSpan(&spanEvent{})
	}
	if num := numMsgs(); num != 20 {
		t.Errorf("there should be 20 spans waiting to be sent but there were %d", num)
	}

	close(readyChan)

	if !toIsShutdown(to) {
		t.Error("trace observer should be shutdown but it is not")
	}
	if num := numMsgs(); num != 0 {
		t.Errorf("there should be 0 spans waiting to be sent but there were %d", num)
	}
}

// Very rarely we would see a data race on shutdown; this test is to reproduce it before fixing it
// (and ensuring we don't bring it back in the future)
func TestTrObsDetectDataRaceOnShutdown(t *testing.T) {
	s, to := createServerAndObserver(t)
	defer s.Close()

	to.consumeSpan(&spanEvent{})
	to.consumeSpan(&spanEvent{})
	to.shutdown(15 * time.Millisecond)
	to.consumeSpan(&spanEvent{})
}

func TestTrObsConsumingAfterShutdown(t *testing.T) {
	s, to := createServerAndObserver(t)
	defer s.Close()

	for i := 0; i < 5; i++ {
		to.consumeSpan(&spanEvent{})
	}
	to.shutdown(time.Nanosecond)
	for i := 0; i < 5; i++ {
		to.consumeSpan(&spanEvent{})
	}
	if !s.DidSpansArrive(t, 5, time.Second) {
		t.Error("did not receive initial 5 spans sent before shutdown")
	}
	if s.DidSpansArrive(t, 1, time.Second) {
		t.Error("spans sent after shutdown was called")
	}
}

func TestTrObsOKSendBackoffNo(t *testing.T) {
	// In this test, the OK response will be noticed by sendSpan
	s := newTestObsServer(t, func(s *expectServer, stream v1.IngestService_RecordSpanServer) error {
		stream.Recv()
		s.spansReceivedChan <- struct{}{}
		return errOK
	})
	defer s.Close()
	cfg := observerConfig{
		log:           logger.ShimLogger{},
		license:       testLicenseKey,
		queueSize:     200,
		appShutdown:   make(chan struct{}),
		dialer:        s.dialer,
		removeBackoff: false, // ensure that the backoff remains for non-OK responses
	}
	to, err := newTraceObserver(runToken, nil, cfg)
	if nil != err {
		t.Fatal(err)
	}
	waitForTrObs(t, to)

	// The grpc client will internally cache spans before sending them to
	// ensure a minimum number of bytes are sent with each batch. Because of
	// this we'll queue up more than enough spans to force at least two of them
	// to get sent and received.
	for i := 0; i < 200; i++ {
		to.consumeSpan(&spanEvent{})
	}
	// If the default backoff of 15 seconds is used, the second span will not
	// be received in time.
	if !s.DidSpansArrive(t, 2, 8*time.Second) {
		t.Error("server did not receive 2 spans")
	}
}

func TestTrObsOKReceiveBackoffNo(t *testing.T) {
	// In this test, the OK response will be noticed by Recv
	var count int
	s := newTestObsServer(t, func(s *expectServer, stream v1.IngestService_RecordSpanServer) error {
		count++
		if count == 1 {
			return errOK
		}
		for {
			stream.Recv()
			s.spansReceivedChan <- struct{}{}
		}
	})
	defer s.Close()
	cfg := observerConfig{
		log:           logger.ShimLogger{},
		license:       testLicenseKey,
		queueSize:     200,
		appShutdown:   make(chan struct{}),
		dialer:        s.dialer,
		removeBackoff: false, // ensure that the backoff remains for non-OK responses
	}
	to, err := newTraceObserver(runToken, nil, cfg)
	if nil != err {
		t.Fatal(err)
	}
	waitForTrObs(t, to)

	// The grpc client will internally cache spans before sending them to
	// ensure a minimum number of bytes are sent with each batch. Because of
	// this we'll queue up more than enough spans to force at least two of them
	// to get sent and received.
	for i := 0; i < 200; i++ {
		to.consumeSpan(&spanEvent{})
	}
	// If the default backoff of 15 seconds is used, the second span will not
	// be received in time.
	if !s.DidSpansArrive(t, 2, time.Second) {
		t.Error("server did not receive 2 spans")
	}
}

func TestTrObsPermissionDeniedSendBackoffYes(t *testing.T) {
	// In this test, the Permission Denied response will be noticed by sendSpan
	s := newTestObsServer(t, func(s *expectServer, stream v1.IngestService_RecordSpanServer) error {
		stream.Recv()
		s.spansReceivedChan <- struct{}{}
		return errPermissionDenied
	})
	defer s.Close()
	cfg := observerConfig{
		log:           logger.ShimLogger{},
		license:       testLicenseKey,
		queueSize:     200,
		appShutdown:   make(chan struct{}),
		dialer:        s.dialer,
		removeBackoff: false, // ensure that the backoff remains for non-OK responses
	}
	to, err := newTraceObserver(runToken, nil, cfg)
	if nil != err {
		t.Fatal(err)
	}
	waitForTrObs(t, to)

	// The grpc client will internally cache spans before sending them to
	// ensure a minimum number of bytes are sent with each batch. Because of
	// this we'll queue up more than enough spans to force them to get sent.
	for i := 0; i < 200; i++ {
		to.consumeSpan(&spanEvent{})
	}
	if !s.DidSpansArrive(t, 1, time.Second) {
		t.Error("server did not receive initial span")
	}
	// Since the default backoff of 15 seconds is used, the second span will not
	// be received in time.
	if s.DidSpansArrive(t, 1, time.Second) {
		t.Error("server received a second span when it should not have")
	}
}

func TestTrObsPermissionDeniedReceiveBackoffYes(t *testing.T) {
	// In this test, the Permission Denied response will be noticed by Recv
	var count int
	s := newTestObsServer(t, func(s *expectServer, stream v1.IngestService_RecordSpanServer) error {
		count++
		if count == 1 {
			return errPermissionDenied
		}
		for {
			stream.Recv()
			s.spansReceivedChan <- struct{}{}
		}
	})
	defer s.Close()
	cfg := observerConfig{
		log:           logger.ShimLogger{},
		license:       testLicenseKey,
		queueSize:     200,
		appShutdown:   make(chan struct{}),
		dialer:        s.dialer,
		removeBackoff: false, // ensure that the backoff remains for non-OK responses
	}
	to, err := newTraceObserver(runToken, nil, cfg)
	if nil != err {
		t.Fatal(err)
	}
	waitForTrObs(t, to)

	// The grpc client will internally cache spans before sending them to
	// ensure a minimum number of bytes are sent with each batch. Because of
	// this we'll queue up more than enough spans to force them to get sent.
	for i := 0; i < 200; i++ {
		to.consumeSpan(&spanEvent{})
	}
	// Since the default backoff of 15 seconds is used, even the first span
	// will not be received in time.
	if s.DidSpansArrive(t, 1, time.Second) {
		t.Error("server received a span when it should not have")
	}
}

/********************
 * Integration test *
 ********************/

func TestTraceObserverRoundTrip(t *testing.T) {
	s := newTestObsServer(t, simpleRecordSpan)
	defer s.Close()
	runToken := "aRunToken"
	app := testAppBlockOnTrObs(DTReplyFieldsWithTrObsDialer(s.dialer, runToken), toCfgWithTrObserver, t)
	txn := app.StartTransaction("txn1")
	txn.StartSegment("seg1").End()
	txn.End()
	app.Shutdown(10 * time.Second)
	app.expectNoLoggedErrors(t)

	// Ensure no spans were sent the normal way
	app.ExpectSpanEvents(t, nil)

	if !s.DidSpansArrive(t, 2, time.Second) {
		t.Error("Did not receive expected spans before timeout")
	}
	s.ExpectMetadata(t, map[string]string{
		"agent_run_token": runToken,
		"license_key":     testLicenseKey,
	})
}
