package newrelic

import (
	"reflect"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/cat"
)

func TestShouldSaveTrace(t *testing.T) {
	for _, tc := range []struct {
		name          string
		expected      bool
		synthetics    bool
		tracerEnabled bool
		collectTraces bool
		duration      time.Duration
		threshold     time.Duration
	}{
		{
			name:          "insufficient duration, all disabled",
			expected:      false,
			synthetics:    false,
			tracerEnabled: false,
			collectTraces: false,
			duration:      1 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "insufficient duration, only synthetics enabled",
			expected:      false,
			synthetics:    true,
			tracerEnabled: false,
			collectTraces: false,
			duration:      1 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "insufficient duration, only tracer enabled",
			expected:      false,
			synthetics:    false,
			tracerEnabled: true,
			collectTraces: false,
			duration:      1 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "insufficient duration, only collect traces enabled",
			expected:      false,
			synthetics:    false,
			tracerEnabled: false,
			collectTraces: true,
			duration:      1 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "insufficient duration, all normal flags enabled",
			expected:      false,
			synthetics:    false,
			tracerEnabled: true,
			collectTraces: true,
			duration:      1 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "insufficient duration, all flags enabled",
			expected:      true,
			synthetics:    true,
			tracerEnabled: true,
			collectTraces: true,
			duration:      1 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "sufficient duration, all disabled",
			expected:      false,
			synthetics:    false,
			tracerEnabled: false,
			collectTraces: false,
			duration:      3 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "sufficient duration, only synthetics enabled",
			expected:      false,
			synthetics:    true,
			tracerEnabled: false,
			collectTraces: false,
			duration:      3 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "sufficient duration, only tracer enabled",
			expected:      false,
			synthetics:    false,
			tracerEnabled: true,
			collectTraces: false,
			duration:      3 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "sufficient duration, only collect traces enabled",
			expected:      false,
			synthetics:    false,
			tracerEnabled: false,
			collectTraces: true,
			duration:      3 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "sufficient duration, all normal flags enabled",
			expected:      true,
			synthetics:    false,
			tracerEnabled: true,
			collectTraces: true,
			duration:      3 * time.Second,
			threshold:     2 * time.Second,
		},
		{
			name:          "sufficient duration, all flags enabled",
			expected:      true,
			synthetics:    true,
			tracerEnabled: true,
			collectTraces: true,
			duration:      3 * time.Second,
			threshold:     2 * time.Second,
		},
	} {
		txn := &txn{}

		cfg := NewConfig("my app", "0123456789012345678901234567890123456789")
		cfg.TransactionTracer.Enabled = tc.tracerEnabled
		cfg.TransactionTracer.Threshold.Duration = tc.threshold
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		reply := internal.ConnectReplyDefaults()
		reply.CollectTraces = tc.collectTraces
		txn.appRun = newAppRun(cfg, reply)

		txn.Duration = tc.duration
		if tc.synthetics {
			txn.CrossProcess.Synthetics = &cat.SyntheticsHeader{}
			txn.CrossProcess.SetSynthetics(tc.synthetics)
		}

		if actual := txn.shouldSaveTrace(); actual != tc.expected {
			t.Errorf("%s: unexpected shouldSaveTrace value; expected %v; got %v", tc.name, tc.expected, actual)
		}
	}
}

func TestLazilyCalculateSampledTrue(t *testing.T) {
	tx := &txn{}
	tx.appRun = &appRun{}
	tx.BetterCAT.Priority = 0.5
	tx.sampledCalculated = false
	tx.BetterCAT.Enabled = true
	tx.Reply = &internal.ConnectReply{
		AdaptiveSampler: internal.SampleEverything{},
	}
	out := tx.lazilyCalculateSampled()
	if !out || !tx.BetterCAT.Sampled || !tx.sampledCalculated || tx.BetterCAT.Priority != 1.5 {
		t.Error(out, tx.BetterCAT.Sampled, tx.sampledCalculated, tx.BetterCAT.Priority)
	}
	tx.Reply.AdaptiveSampler = internal.SampleNothing{}
	out = tx.lazilyCalculateSampled()
	if !out || !tx.BetterCAT.Sampled || !tx.sampledCalculated || tx.BetterCAT.Priority != 1.5 {
		t.Error(out, tx.BetterCAT.Sampled, tx.sampledCalculated, tx.BetterCAT.Priority)
	}
}

func TestLazilyCalculateSampledFalse(t *testing.T) {
	tx := &txn{}
	tx.appRun = &appRun{}
	tx.BetterCAT.Priority = 0.5
	tx.sampledCalculated = false
	tx.BetterCAT.Enabled = true
	tx.Reply = &internal.ConnectReply{
		AdaptiveSampler: internal.SampleNothing{},
	}
	out := tx.lazilyCalculateSampled()
	if out || tx.BetterCAT.Sampled || !tx.sampledCalculated || tx.BetterCAT.Priority != 0.5 {
		t.Error(out, tx.BetterCAT.Sampled, tx.sampledCalculated, tx.BetterCAT.Priority)
	}
	tx.Reply.AdaptiveSampler = internal.SampleEverything{}
	out = tx.lazilyCalculateSampled()
	if out || tx.BetterCAT.Sampled || !tx.sampledCalculated || tx.BetterCAT.Priority != 0.5 {
		t.Error(out, tx.BetterCAT.Sampled, tx.sampledCalculated, tx.BetterCAT.Priority)
	}
}

func TestLazilyCalculateSampledCATDisabled(t *testing.T) {
	tx := &txn{}
	tx.appRun = &appRun{}
	tx.BetterCAT.Priority = 0.5
	tx.sampledCalculated = false
	tx.BetterCAT.Enabled = false
	tx.Reply = &internal.ConnectReply{
		AdaptiveSampler: internal.SampleEverything{},
	}
	out := tx.lazilyCalculateSampled()
	if out || tx.BetterCAT.Sampled || tx.sampledCalculated || tx.BetterCAT.Priority != 0.5 {
		t.Error(out, tx.BetterCAT.Sampled, tx.sampledCalculated, tx.BetterCAT.Priority)
	}
	out = tx.lazilyCalculateSampled()
	if out || tx.BetterCAT.Sampled || tx.sampledCalculated || tx.BetterCAT.Priority != 0.5 {
		t.Error(out, tx.BetterCAT.Sampled, tx.sampledCalculated, tx.BetterCAT.Priority)
	}
}

type expectTxnTimes struct {
	txn       *txn
	testName  string
	start     time.Time
	stop      time.Time
	duration  time.Duration
	totalTime time.Duration
}

func TestTransactionDurationTotalTime(t *testing.T) {
	// These tests touch internal txn structures rather than the public API:
	// Testing duration and total time is tough because our API functions do
	// not take fixed times.
	start := time.Now()
	testTxnTimes := func(expect expectTxnTimes) {
		if expect.txn.Start != expect.start {
			t.Error("start time", expect.testName, expect.txn.Start, expect.start)
		}
		if expect.txn.Stop != expect.stop {
			t.Error("stop time", expect.testName, expect.txn.Stop, expect.stop)
		}
		if expect.txn.Duration != expect.duration {
			t.Error("duration", expect.testName, expect.txn.Duration, expect.duration)
		}
		if expect.txn.TotalTime != expect.totalTime {
			t.Error("total time", expect.testName, expect.txn.TotalTime, expect.totalTime)
		}
	}

	// Basic transaction with no async activity.
	tx := &txn{}
	tx.markStart(start)
	segmentStart := internal.StartSegment(&tx.TxnData, &tx.mainThread, start.Add(1*time.Second))
	internal.EndBasicSegment(&tx.TxnData, &tx.mainThread, segmentStart, start.Add(2*time.Second), "name")
	tx.markEnd(start.Add(3*time.Second), &tx.mainThread)
	testTxnTimes(expectTxnTimes{
		txn:       tx,
		testName:  "basic transaction",
		start:     start,
		stop:      start.Add(3 * time.Second),
		duration:  3 * time.Second,
		totalTime: 3 * time.Second,
	})

	// Transaction with async activity.
	tx = &txn{}
	tx.markStart(start)
	segmentStart = internal.StartSegment(&tx.TxnData, &tx.mainThread, start.Add(1*time.Second))
	internal.EndBasicSegment(&tx.TxnData, &tx.mainThread, segmentStart, start.Add(2*time.Second), "name")
	asyncThread := createThread(tx)
	asyncSegmentStart := internal.StartSegment(&tx.TxnData, asyncThread, start.Add(1*time.Second))
	internal.EndBasicSegment(&tx.TxnData, asyncThread, asyncSegmentStart, start.Add(2*time.Second), "name")
	tx.markEnd(start.Add(3*time.Second), &tx.mainThread)
	testTxnTimes(expectTxnTimes{
		txn:       tx,
		testName:  "transaction with async activity",
		start:     start,
		stop:      start.Add(3 * time.Second),
		duration:  3 * time.Second,
		totalTime: 4 * time.Second,
	})

	// Transaction ended on async thread.
	tx = &txn{}
	tx.markStart(start)
	segmentStart = internal.StartSegment(&tx.TxnData, &tx.mainThread, start.Add(1*time.Second))
	internal.EndBasicSegment(&tx.TxnData, &tx.mainThread, segmentStart, start.Add(2*time.Second), "name")
	asyncThread = createThread(tx)
	asyncSegmentStart = internal.StartSegment(&tx.TxnData, asyncThread, start.Add(1*time.Second))
	internal.EndBasicSegment(&tx.TxnData, asyncThread, asyncSegmentStart, start.Add(2*time.Second), "name")
	tx.markEnd(start.Add(3*time.Second), asyncThread)
	testTxnTimes(expectTxnTimes{
		txn:       tx,
		testName:  "transaction ended on async thread",
		start:     start,
		stop:      start.Add(3 * time.Second),
		duration:  3 * time.Second,
		totalTime: 4 * time.Second,
	})

	// Duration exceeds TotalTime.
	tx = &txn{}
	tx.markStart(start)
	segmentStart = internal.StartSegment(&tx.TxnData, &tx.mainThread, start.Add(0*time.Second))
	internal.EndBasicSegment(&tx.TxnData, &tx.mainThread, segmentStart, start.Add(1*time.Second), "name")
	asyncThread = createThread(tx)
	asyncSegmentStart = internal.StartSegment(&tx.TxnData, asyncThread, start.Add(2*time.Second))
	internal.EndBasicSegment(&tx.TxnData, asyncThread, asyncSegmentStart, start.Add(3*time.Second), "name")
	tx.markEnd(start.Add(3*time.Second), asyncThread)
	testTxnTimes(expectTxnTimes{
		txn:       tx,
		testName:  "TotalTime should be at least Duration",
		start:     start,
		stop:      start.Add(3 * time.Second),
		duration:  3 * time.Second,
		totalTime: 3 * time.Second,
	})
}

func TestGetTraceMetadataDistributedTracingDisabled(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	metadata := txn.GetTraceMetadata()
	if metadata.SpanID != "" {
		t.Error(metadata.SpanID)
	}
	if metadata.TraceID != "" {
		t.Error(metadata.TraceID)
	}
	if m := metadata.Map(); !reflect.DeepEqual(m, map[string]interface{}{}) {
		t.Error(m)
	}
}

func TestGetTraceMetadataSuccess(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
		reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	metadata := txn.GetTraceMetadata()
	if metadata.SpanID != "bcfb32e050b264b8" {
		t.Error(metadata.SpanID)
	}
	if metadata.TraceID != "d9466896a525ccbf" {
		t.Error(metadata.TraceID)
	}
	StartSegment(txn, "name")
	// Span id should be different now that a segment has started.
	metadata = txn.GetTraceMetadata()
	if metadata.SpanID != "0e97aeb2f79d5d27" {
		t.Error(metadata.SpanID)
	}
	if metadata.TraceID != "d9466896a525ccbf" {
		t.Error(metadata.TraceID)
	}
	if m := metadata.Map(); !reflect.DeepEqual(m, map[string]interface{}{
		"span_id":  "0e97aeb2f79d5d27",
		"trace_id": "d9466896a525ccbf",
	}) {
		t.Error(m)
	}
}

func TestGetTraceMetadataEnded(t *testing.T) {
	// Test that GetTraceMetadata returns empty strings if the transaction
	// has been finished.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
		reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	txn.End()
	metadata := txn.GetTraceMetadata()
	if metadata.SpanID != "" {
		t.Error(metadata.SpanID)
	}
	if metadata.TraceID != "" {
		t.Error(metadata.TraceID)
	}
	if m := metadata.Map(); !reflect.DeepEqual(m, map[string]interface{}{}) {
		t.Error(m)
	}
}

func TestGetTraceMetadataNotSampled(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleNothing{}
		reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	metadata := txn.GetTraceMetadata()
	if metadata.SpanID != "" {
		t.Error(metadata.SpanID)
	}
	if metadata.TraceID != "d9466896a525ccbf" {
		t.Error(metadata.TraceID)
	}
	if m := metadata.Map(); !reflect.DeepEqual(m, map[string]interface{}{
		"trace_id": "d9466896a525ccbf",
	}) {
		t.Error(m)
	}
}

func TestGetTraceMetadataSpanEventsDisabled(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
		reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	metadata := txn.GetTraceMetadata()
	if metadata.SpanID != "" {
		t.Error(metadata.SpanID)
	}
	if metadata.TraceID != "d9466896a525ccbf" {
		t.Error(metadata.TraceID)
	}
	if m := metadata.Map(); !reflect.DeepEqual(m, map[string]interface{}{
		"trace_id": "d9466896a525ccbf",
	}) {
		t.Error(m)
	}
}

func TestGetTraceMetadataInboundPayload(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
		reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
		reply.AccountID = "account-id"
		reply.TrustedAccountKey = "trust-key"
		reply.PrimaryAppID = "app-id"
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	payload := app.StartTransaction("hello", nil, nil).CreateDistributedTracePayload()
	p := payload.(internal.Payload)
	p.TracedID = "trace-id"

	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, p)
	if nil != err {
		t.Error(err)
	}
	metadata := txn.GetTraceMetadata()
	if metadata.SpanID != "9d2c19bd03daf755" {
		t.Error(metadata.SpanID)
	}
	if metadata.TraceID != "trace-id" {
		t.Error(metadata.TraceID)
	}
	if m := metadata.Map(); !reflect.DeepEqual(m, map[string]interface{}{
		"span_id":  "9d2c19bd03daf755",
		"trace_id": "trace-id",
	}) {
		t.Error(m)
	}
}
