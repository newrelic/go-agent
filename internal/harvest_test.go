package internal

import (
	"testing"
	"time"
)

func TestCreateFinalMetrics(t *testing.T) {
	now := time.Now()

	h := newHarvest(now)
	h.createFinalMetrics()
	expectMetrics(t, h.metrics, []WantMetric{
		{instanceReporting, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{customEventsSeen, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{customEventsSent, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{txnEventsSeen, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{txnEventsSent, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{errorEventsSeen, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{errorEventsSent, "", true, []float64{0, 0, 0, 0, 0, 0}},
	})

	h = newHarvest(now)
	h.metrics = newMetricTable(0, now)
	h.customEvents = newCustomEvents(1)
	h.txnEvents = newTxnEvents(1)
	h.errorEvents = newErrorEvents(1)

	h.metrics.addSingleCount("drop me!", unforced)

	customE, err := createCustomEvent("my event type", map[string]interface{}{"zip": 1}, time.Now())
	if nil != err {
		t.Fatal(err)
	}
	h.customEvents.Add(customE)
	h.customEvents.Add(customE)

	txnE := &txnEvent{}
	h.txnEvents.AddTxnEvent(txnE)
	h.txnEvents.AddTxnEvent(txnE)

	h.errorEvents.Add(&errorEvent{})
	h.errorEvents.Add(&errorEvent{})

	h.createFinalMetrics()
	expectMetrics(t, h.metrics, []WantMetric{
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

func TestCreateTxnMetrics(t *testing.T) {
	webName := "WebTransaction/zip/zap"
	backgroundName := "OtherTransaction/zip/zap"
	args := createTxnMetricsArgs{
		Duration:       123 * time.Second,
		Exclusive:      109 * time.Second,
		ApdexThreshold: 2 * time.Second,
	}

	args.Name = webName
	args.IsWeb = true
	args.ErrorsSeen = 1
	args.Zone = apdexTolerating
	h := newHarvest(time.Now())
	h.createTxnMetrics(args)
	expectMetrics(t, h.metrics, []WantMetric{
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
	args.ErrorsSeen = 0
	args.Zone = apdexTolerating
	h = newHarvest(time.Now())
	h.createTxnMetrics(args)
	expectMetrics(t, h.metrics, []WantMetric{
		{webName, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{webRollup, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{dispatcherMetric, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{apdexRollup, "", true, []float64{0, 1, 0, 2, 2, 0}},
		{"Apdex/zip/zap", "", false, []float64{0, 1, 0, 2, 2, 0}},
	})

	args.Name = backgroundName
	args.IsWeb = false
	args.ErrorsSeen = 1
	args.Zone = apdexNone
	h = newHarvest(time.Now())
	h.createTxnMetrics(args)
	expectMetrics(t, h.metrics, []WantMetric{
		{backgroundName, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{backgroundRollup, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{errorsAll, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{errorsBackground, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Errors/" + backgroundName, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})

	args.Name = backgroundName
	args.IsWeb = false
	args.ErrorsSeen = 0
	args.Zone = apdexNone
	h = newHarvest(time.Now())
	h.createTxnMetrics(args)
	expectMetrics(t, h.metrics, []WantMetric{
		{backgroundName, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{backgroundRollup, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
	})

}
