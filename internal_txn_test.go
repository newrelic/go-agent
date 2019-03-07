package newrelic

import (
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
			expected:      true,
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
			expected:      true,
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
		txn.Config.TransactionTracer.Enabled = tc.tracerEnabled
		txn.Config.TransactionTracer.Threshold.Duration = tc.threshold
		txn.Reply = &internal.ConnectReply{CollectTraces: tc.collectTraces}
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
