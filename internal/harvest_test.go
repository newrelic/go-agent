package internal

import (
	"testing"
	"time"
)

func TestCreateFinalMetrics(t *testing.T) {
	now := time.Now()

	h := NewHarvest(now)
	h.CreateFinalMetrics()
	ExpectMetrics(t, h.Metrics, []WantMetric{
		{instanceReporting, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{customEventsSeen, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{customEventsSent, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{txnEventsSeen, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{txnEventsSent, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{errorEventsSeen, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{errorEventsSent, "", true, []float64{0, 0, 0, 0, 0, 0}},
	})

	h = NewHarvest(now)
	h.Metrics = newMetricTable(0, now)
	h.CustomEvents = newCustomEvents(1)
	h.TxnEvents = newTxnEvents(1)
	h.ErrorEvents = newErrorEvents(1)

	h.Metrics.addSingleCount("drop me!", unforced)

	customE, err := CreateCustomEvent("my event type", map[string]interface{}{"zip": 1}, time.Now())
	if nil != err {
		t.Fatal(err)
	}
	h.CustomEvents.Add(customE)
	h.CustomEvents.Add(customE)

	txnE := &TxnEvent{}
	h.TxnEvents.AddTxnEvent(txnE)
	h.TxnEvents.AddTxnEvent(txnE)

	h.ErrorEvents.Add(&ErrorEvent{})
	h.ErrorEvents.Add(&ErrorEvent{})

	h.CreateFinalMetrics()
	ExpectMetrics(t, h.Metrics, []WantMetric{
		{instanceReporting, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{customEventsSeen, "", true, []float64{2, 0, 0, 0, 0, 0}},
		{customEventsSent, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{txnEventsSeen, "", true, []float64{2, 0, 0, 0, 0, 0}},
		{txnEventsSent, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{errorEventsSeen, "", true, []float64{2, 0, 0, 0, 0, 0}},
		{errorEventsSent, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{supportabilityDropped, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestEmptyPayloads(t *testing.T) {
	h := NewHarvest(time.Now())
	payloads := h.Payloads()
	for _, p := range payloads {
		d, err := p.Data("agentRunID", time.Now())
		if d != nil || err != nil {
			t.Error(d, err)
		}
	}
}

func TestMergeFailedHarvest(t *testing.T) {
	start1 := time.Now()
	start2 := start1.Add(1 * time.Minute)
	h := NewHarvest(start1)
	h.Metrics.addCount("zip", 1, forced)
	h.TxnEvents.AddTxnEvent(&TxnEvent{
		Name:      "finalName",
		Timestamp: time.Now(),
		Duration:  1 * time.Second,
	})
	customEventParams := map[string]interface{}{"zip": 1}
	ce, err := CreateCustomEvent("myEvent", customEventParams, time.Now())
	if nil != err {
		t.Fatal(err)
	}
	h.CustomEvents.Add(ce)
	h.ErrorEvents.Add(&ErrorEvent{
		Klass:    "klass",
		Msg:      "msg",
		When:     time.Now(),
		TxnName:  "finalName",
		Duration: 1 * time.Second,
	})
	e := &TxnError{
		When:  time.Now(),
		Msg:   "msg",
		Klass: "klass",
		Stack: GetStackTrace(0),
	}
	addTxnError(h.ErrorTraces, e, "finalName", "requestURI", nil)

	if start1 != h.Metrics.metricPeriodStart {
		t.Error(h.Metrics.metricPeriodStart)
	}
	if 0 != h.Metrics.failedHarvests {
		t.Error(h.Metrics.failedHarvests)
	}
	if 0 != h.CustomEvents.events.failedHarvests {
		t.Error(h.CustomEvents.events.failedHarvests)
	}
	if 0 != h.TxnEvents.events.failedHarvests {
		t.Error(h.TxnEvents.events.failedHarvests)
	}
	if 0 != h.ErrorEvents.events.failedHarvests {
		t.Error(h.ErrorEvents.events.failedHarvests)
	}
	ExpectMetrics(t, h.Metrics, []WantMetric{
		{"zip", "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
	ExpectCustomEvents(t, h.CustomEvents, []WantCustomEvent{
		{Type: "myEvent", Params: customEventParams},
	})
	ExpectErrorEvents(t, h.ErrorEvents, []WantErrorEvent{
		{TxnName: "finalName", Msg: "msg", Klass: "klass"},
	})
	ExpectTxnEvents(t, h.TxnEvents, []WantTxnEvent{
		{Name: "finalName"},
	})
	ExpectErrors(t, h.ErrorTraces, []WantError{{
		TxnName: "finalName",
		Msg:     "msg",
		Klass:   "klass",
		Caller:  "internal.TestMergeFailedHarvest",
		URL:     "requestURI",
	}})

	nextHarvest := NewHarvest(start2)
	if start2 != nextHarvest.Metrics.metricPeriodStart {
		t.Error(nextHarvest.Metrics.metricPeriodStart)
	}
	payloads := h.Payloads()
	for _, p := range payloads {
		p.MergeIntoHarvest(nextHarvest)
	}

	if start1 != nextHarvest.Metrics.metricPeriodStart {
		t.Error(nextHarvest.Metrics.metricPeriodStart)
	}
	if 1 != nextHarvest.Metrics.failedHarvests {
		t.Error(nextHarvest.Metrics.failedHarvests)
	}
	if 1 != nextHarvest.CustomEvents.events.failedHarvests {
		t.Error(nextHarvest.CustomEvents.events.failedHarvests)
	}
	if 1 != nextHarvest.TxnEvents.events.failedHarvests {
		t.Error(nextHarvest.TxnEvents.events.failedHarvests)
	}
	if 1 != nextHarvest.ErrorEvents.events.failedHarvests {
		t.Error(nextHarvest.ErrorEvents.events.failedHarvests)
	}
	ExpectMetrics(t, nextHarvest.Metrics, []WantMetric{
		{"zip", "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
	ExpectCustomEvents(t, nextHarvest.CustomEvents, []WantCustomEvent{
		{Type: "myEvent", Params: customEventParams},
	})
	ExpectErrorEvents(t, nextHarvest.ErrorEvents, []WantErrorEvent{
		{TxnName: "finalName", Msg: "msg", Klass: "klass"},
	})
	ExpectTxnEvents(t, nextHarvest.TxnEvents, []WantTxnEvent{
		{Name: "finalName"},
	})
	ExpectErrors(t, nextHarvest.ErrorTraces, []WantError{})
}

func TestCreateTxnMetrics(t *testing.T) {
	webName := "WebTransaction/zip/zap"
	backgroundName := "OtherTransaction/zip/zap"
	args := CreateTxnMetricsArgs{
		Duration:       123 * time.Second,
		Exclusive:      109 * time.Second,
		ApdexThreshold: 2 * time.Second,
	}

	args.Name = webName
	args.IsWeb = true
	args.HasErrors = true
	args.Zone = ApdexTolerating
	metrics := newMetricTable(100, time.Now())
	CreateTxnMetrics(args, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
		{webName, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{webRollup, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{dispatcherMetric, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{errorsAll, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{errorsWeb, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Errors/" + webName, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{apdexRollup, "", true, []float64{0, 1, 0, 2, 2, 0}},
		{"Apdex/zip/zap", "", false, []float64{0, 1, 0, 2, 2, 0}},
	})

	args.Name = webName
	args.IsWeb = true
	args.HasErrors = false
	args.Zone = ApdexTolerating
	metrics = newMetricTable(100, time.Now())
	CreateTxnMetrics(args, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
		{webName, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{webRollup, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{dispatcherMetric, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{apdexRollup, "", true, []float64{0, 1, 0, 2, 2, 0}},
		{"Apdex/zip/zap", "", false, []float64{0, 1, 0, 2, 2, 0}},
	})

	args.Name = backgroundName
	args.IsWeb = false
	args.HasErrors = true
	args.Zone = ApdexNone
	metrics = newMetricTable(100, time.Now())
	CreateTxnMetrics(args, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
		{backgroundName, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{backgroundRollup, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{errorsAll, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{errorsBackground, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Errors/" + backgroundName, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})

	args.Name = backgroundName
	args.IsWeb = false
	args.HasErrors = false
	args.Zone = ApdexNone
	metrics = newMetricTable(100, time.Now())
	CreateTxnMetrics(args, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
		{backgroundName, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{backgroundRollup, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
	})

}
