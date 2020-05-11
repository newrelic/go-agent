// +build go1.9
// This build tag is necessary because Infinite Tracing is only supported for Go version 1.9 and up

package newrelic

import (
	"net"
	"reflect"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/newrelic/go-agent/v3/internal"
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
		{code: 1, expect: "CANCELED"},
		{code: 2, expect: "UNKNOWN"},
		{code: 3, expect: "INVALIDARGUMENT"},
		{code: 4, expect: "DEADLINEEXCEEDED"},
		{code: 5, expect: "NOTFOUND"},
		{code: 6, expect: "ALREADYEXISTS"},
		{code: 7, expect: "PERMISSIONDENIED"},
		{code: 8, expect: "RESOURCEEXHAUSTED"},
		{code: 9, expect: "FAILEDPRECONDITION"},
		{code: 10, expect: "ABORTED"},
		{code: 11, expect: "OUTOFRANGE"},
		{code: 12, expect: "UNIMPLEMENTED"},
		{code: 13, expect: "INTERNAL"},
		{code: 14, expect: "UNAVAILABLE"},
		{code: 15, expect: "DATALOSS"},
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

const runToken = "aRunToken"

func TestTraceObserverRestart(t *testing.T) {
	s, to := createServerAndObserver(t)
	defer s.Close()
	s.ExpectMetadata(t, map[string]string{
		"agent_run_token": runToken,
		"license_key":     testLicenseKey,
	})
	newToken := "aNewRunToken"
	to.restart(internal.AgentRunID(newToken))

	// Make sure the server has received the new data
	to.consumeSpan(&spanEvent{})
	if !s.WaitForSpans(t, 1, 50*time.Millisecond) {
		t.Error("Did not receive expected spans before timeout")
	}

	s.ExpectMetadata(t, map[string]string{
		"agent_run_token": newToken,
		"license_key":     testLicenseKey,
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
	if s.WaitForSpans(t, 1, 50*time.Millisecond) {
		t.Error("Got a span we did not expect to get")
	}
	s.Close()

	shutdownApp(to)

	to.consumeSpan(&spanEvent{})
	if s.WaitForSpans(t, 1, 50*time.Millisecond) {
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

	if !s.WaitForSpans(t, 2, 50*time.Millisecond) {
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
	if !s.WaitForSpans(t, 1, 50*time.Millisecond) {
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
	slowDialer := func(str string, d time.Duration) (net.Conn, error) {
		<-readyChan
		return s.dialer(str, d)
	}
	cfg := observerConfig{
		log:         logger.ShimLogger{},
		license:     testLicenseKey,
		queueSize:   20,
		appShutdown: make(chan struct{}),
		dialer:      slowDialer,
	}
	to, err := newTraceObserver(runToken, cfg)
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
	to.restart("A New Run Token")
	s.ExpectMetadata(t, map[string]string{
		"agent_run_token": runToken,
		"license_key":     testLicenseKey,
	})

	shutdownApp(to)

	// Make sure we don't panic and don't send updated metadata
	to.restart("A New Run Token")
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
	slowDialer := func(str string, d time.Duration) (net.Conn, error) {
		<-readyChan
		return s.dialer(str, d)
	}
	cfg := observerConfig{
		log:         logger.ShimLogger{},
		license:     testLicenseKey,
		queueSize:   20,
		appShutdown: make(chan struct{}),
		dialer:      slowDialer,
	}
	to, err := newTraceObserver(runToken, cfg)
	if nil != err {
		t.Fatal(err)
	}

	newToken := "A New Run Token"
	to.restart(internal.AgentRunID(newToken))
	if s.WaitForSpans(t, 1, 50*time.Millisecond) {
		t.Error("Got a span we did not expect to get")
	}
	s.ExpectMetadata(t, nil)

	close(readyChan)
	if s.WaitForSpans(t, 1, 500*time.Millisecond) {
		t.Error("Got a span we did not expect to get")
	}
	s.ExpectMetadata(t, map[string]string{
		"agent_run_token": newToken,
		"license_key":     testLicenseKey,
	})
}

func TestTrObsSlowConnectAndConsumeSpan(t *testing.T) {
	s := newTestObsServer(t, simpleRecordSpan)
	defer s.Close()
	readyChan := make(chan struct{})
	slowDialer := func(str string, d time.Duration) (net.Conn, error) {
		<-readyChan
		return s.dialer(str, d)
	}
	cfg := observerConfig{
		log:         logger.ShimLogger{},
		license:     testLicenseKey,
		queueSize:   20,
		appShutdown: make(chan struct{}),
		dialer:      slowDialer,
	}
	to, err := newTraceObserver(runToken, cfg)
	if nil != err {
		t.Fatal(err)
	}

	to.consumeSpan(&spanEvent{})
	if s.WaitForSpans(t, 1, 50*time.Millisecond) {
		t.Error("Got a span we did not expect to get")
	}

	close(readyChan)
	if !s.WaitForSpans(t, 1, 50*time.Millisecond) {
		t.Error("Did not receive expected spans before timeout")
	}
}

func TestTrObsSlowConnectAndDumpSupportabilityMetrics(t *testing.T) {
	s := newTestObsServer(t, simpleRecordSpan)
	defer s.Close()
	readyChan := make(chan struct{})
	slowDialer := func(str string, d time.Duration) (net.Conn, error) {
		<-readyChan
		return s.dialer(str, d)
	}
	cfg := observerConfig{
		log:         logger.ShimLogger{},
		license:     testLicenseKey,
		queueSize:   20,
		appShutdown: make(chan struct{}),
		dialer:      slowDialer,
	}
	to, err := newTraceObserver(runToken, cfg)
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
	if !s.WaitForSpans(t, 1, 50*time.Millisecond) {
		t.Error("Did not receive expected spans before timeout")
	}
	expectSupportabilityMetrics(t, to, map[string]float64{
		"Supportability/InfiniteTracing/Span/Seen": 0,
		"Supportability/InfiniteTracing/Span/Sent": 1,
	})
}

// TODO: come back to this when we have more brainpower.
// We need to figure out how to cancel the call to serviceClient.RecordSpan(ctx) if it doesn't connect
func TestTrObsSlowConnectAndShutdown(t *testing.T) {
	s := newTestObsServer(t, simpleRecordSpan)
	defer s.Close()
	readyChan := make(chan struct{})
	slowDialer := func(str string, d time.Duration) (net.Conn, error) {
		<-readyChan
		return s.dialer(str, d)
	}
	cfg := observerConfig{
		log:         logger.ShimLogger{},
		license:     testLicenseKey,
		queueSize:   20,
		appShutdown: make(chan struct{}),
		dialer:      slowDialer,
	}
	to, err := newTraceObserver(runToken, cfg)
	if nil != err {
		t.Fatal(err)
	}

	to.consumeSpan(&spanEvent{})

	if err := to.shutdown(time.Nanosecond); err == nil {
		t.Error("trace observer was able to shutdown when it shouldn't have")
	}

	close(readyChan)

	// TODO: This sleep is so long because it is waiting on the defered 500
	// millisecond sleep for closing the grpc conn.
	time.Sleep(550 * time.Millisecond)
	if !to.(*gRPCtraceObserver).isShutdownComplete() {
		t.Error("trace observer should be shutdown but it is not")
	}
	if !s.WaitForSpans(t, 1, 50*time.Millisecond) {
		t.Error("span was not received")
	}
}

/***********************
 * Integration test(s) *
 ***********************/
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

	s.WaitForSpans(t, 2, time.Second)
	s.ExpectMetadata(t, map[string]string{
		"agent_run_token": runToken,
		"license_key":     testLicenseKey,
	})
}
