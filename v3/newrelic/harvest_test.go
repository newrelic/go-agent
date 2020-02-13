package newrelic

import (
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
)

func TestHarvestTimerAllFixed(t *testing.T) {
	now := time.Now()
	harvest := newHarvest(now, &dfltHarvestCfgr{})
	timer := harvest.timer
	for _, tc := range []struct {
		Elapsed time.Duration
		Expect  harvestTypes
	}{
		{60 * time.Second, 0},
		{61 * time.Second, harvestTypesAll},
		{62 * time.Second, 0},
		{120 * time.Second, 0},
		{121 * time.Second, harvestTypesAll},
		{122 * time.Second, 0},
	} {
		if ready := timer.ready(now.Add(tc.Elapsed)); ready != tc.Expect {
			t.Error(tc.Elapsed, ready, tc.Expect)
		}
	}
}

var one uint = 1
var two uint = 2
var three uint = 3
var four uint = 4

func TestHarvestTimerAllConfigurable(t *testing.T) {
	now := time.Now()
	harvest := newHarvest(now, &dfltHarvestCfgr{
		reportPeriods: map[harvestTypes]time.Duration{
			harvestMetricsTraces: fixedHarvestPeriod,
			harvestTypesEvents:   time.Second * 30,
		},
		maxTxnEvents:    &one,
		maxCustomEvents: &two,
		maxSpanEvents:   &three,
		maxErrorEvents:  &four,
	})
	timer := harvest.timer
	for _, tc := range []struct {
		Elapsed time.Duration
		Expect  harvestTypes
	}{
		{30 * time.Second, 0},
		{31 * time.Second, harvestTypesEvents},
		{32 * time.Second, 0},
		{61 * time.Second, harvestTypesAll},
		{62 * time.Second, 0},
		{91 * time.Second, harvestTypesEvents},
		{92 * time.Second, 0},
	} {
		if ready := timer.ready(now.Add(tc.Elapsed)); ready != tc.Expect {
			t.Error(tc.Elapsed, ready, tc.Expect)
		}
	}
}

func TestCreateFinalMetrics(t *testing.T) {
	now := time.Now()

	// If the harvest or metrics is nil then CreateFinalMetrics should
	// not panic.
	var nilHarvest *harvest
	nilHarvest.CreateFinalMetrics(nil, &dfltHarvestCfgr{})
	emptyHarvest := &harvest{}
	emptyHarvest.CreateFinalMetrics(nil, &dfltHarvestCfgr{})

	replyJSON := []byte(`{"return_value":{
		"metric_name_rules":[{
			"match_expression": "rename_me",
			"replacement": "been_renamed"
		}],
		"event_harvest_config":{
			"report_period_ms": 2000,
			"harvest_limits": {
				"analytic_event_data": 22,
				"custom_event_data": 33,
				"error_event_data": 44,
				"span_event_data": 55
			}
		}
	}}`)
	reply, err := internal.UnmarshalConnectReply(replyJSON, internal.PreconnectReply{})
	if err != nil {
		t.Fatal(err)
	}
	var txnEvents uint = 22
	var customEvents uint = 33
	var errorEvents uint = 44
	var spanEvents uint = 55
	cfgr := &dfltHarvestCfgr{
		reportPeriods: map[harvestTypes]time.Duration{
			harvestMetricsTraces: fixedHarvestPeriod,
			harvestTypesEvents:   time.Second * 2,
		},
		maxTxnEvents:    &txnEvents,
		maxCustomEvents: &customEvents,
		maxErrorEvents:  &errorEvents,
		maxSpanEvents:   &spanEvents,
	}
	h := newHarvest(now, cfgr)
	h.Metrics.addCount("rename_me", 1.0, unforced)
	h.CreateFinalMetrics(reply, cfgr)
	expectMetrics(t, h.Metrics, []internal.WantMetric{
		{instanceReporting, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"been_renamed", "", false, []float64{1.0, 0, 0, 0, 0, 0}},
		{"Supportability/EventHarvest/ReportPeriod", "", true, []float64{1, 2, 2, 2, 2, 2 * 2}},
		{"Supportability/EventHarvest/AnalyticEventData/HarvestLimit", "", true, []float64{1, 22, 22, 22, 22, 22 * 22}},
		{"Supportability/EventHarvest/CustomEventData/HarvestLimit", "", true, []float64{1, 33, 33, 33, 33, 33 * 33}},
		{"Supportability/EventHarvest/ErrorEventData/HarvestLimit", "", true, []float64{1, 44, 44, 44, 44, 44 * 44}},
		{"Supportability/EventHarvest/SpanEventData/HarvestLimit", "", true, []float64{1, 55, 55, 55, 55, 55 * 55}},
		{"Supportability/Go/Version/" + Version, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Supportability/Go/Runtime/Version/" + goVersionSimple, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})

	// Test again without any metric rules or event_harvest_config.

	replyJSON = []byte(`{"return_value":{
	}}`)
	reply, err = internal.UnmarshalConnectReply(replyJSON, internal.PreconnectReply{})
	if err != nil {
		t.Fatal(err)
	}
	h = newHarvest(now, &dfltHarvestCfgr{})
	h.Metrics.addCount("rename_me", 1.0, unforced)
	h.CreateFinalMetrics(reply, &dfltHarvestCfgr{})
	expectMetrics(t, h.Metrics, []internal.WantMetric{
		{instanceReporting, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"rename_me", "", false, []float64{1.0, 0, 0, 0, 0, 0}},
		{"Supportability/EventHarvest/ReportPeriod", "", true, []float64{1, 60, 60, 60, 60, 60 * 60}},
		{"Supportability/EventHarvest/AnalyticEventData/HarvestLimit", "", true, []float64{1, 10 * 1000, 10 * 1000, 10 * 1000, 10 * 1000, 10 * 1000 * 10 * 1000}},
		{"Supportability/EventHarvest/CustomEventData/HarvestLimit", "", true, []float64{1, 10 * 1000, 10 * 1000, 10 * 1000, 10 * 1000, 10 * 1000 * 10 * 1000}},
		{"Supportability/EventHarvest/ErrorEventData/HarvestLimit", "", true, []float64{1, 100, 100, 100, 100, 100 * 100}},
		{"Supportability/EventHarvest/SpanEventData/HarvestLimit", "", true, []float64{1, 1000, 1000, 1000, 1000, 1000 * 1000}},
		{"Supportability/Go/Version/" + Version, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Supportability/Go/Runtime/Version/" + goVersionSimple, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestEmptyPayloads(t *testing.T) {
	h := newHarvest(time.Now(), &dfltHarvestCfgr{})
	payloads := h.Payloads(true)
	if len(payloads) != 8 {
		t.Error(len(payloads))
	}
	for _, p := range payloads {
		d, err := p.Data("agentRunID", time.Now())
		if d != nil || err != nil {
			t.Error(d, err)
		}
	}
}
func TestPayloadsNilHarvest(t *testing.T) {
	var nilHarvest *harvest
	payloads := nilHarvest.Payloads(true)
	if len(payloads) != 0 {
		t.Error(len(payloads))
	}
}

func TestPayloadsEmptyHarvest(t *testing.T) {
	h := &harvest{}
	payloads := h.Payloads(true)
	if len(payloads) != 0 {
		t.Error(len(payloads))
	}
}

func TestHarvestNothingReady(t *testing.T) {
	now := time.Now()
	h := newHarvest(now, &dfltHarvestCfgr{})
	ready := h.Ready(now.Add(10 * time.Second))
	if ready != nil {
		t.Error("harvest should be nil")
	}
	payloads := ready.Payloads(true)
	if len(payloads) != 0 {
		t.Error(payloads)
	}
	expectMetrics(t, h.Metrics, []internal.WantMetric{})
}

func TestHarvestCustomEventsReady(t *testing.T) {
	now := time.Now()
	fixedHarvestTypes := harvestMetricsTraces & harvestTxnEvents & harvestSpanEvents & harvestErrorEvents
	h := newHarvest(now, &dfltHarvestCfgr{
		reportPeriods: map[harvestTypes]time.Duration{
			fixedHarvestTypes:   fixedHarvestPeriod,
			harvestCustomEvents: time.Second * 5,
		},
		maxCustomEvents: &three,
	})
	params := map[string]interface{}{"zip": 1}
	ce, _ := createCustomEvent("myEvent", params, time.Now())
	h.CustomEvents.Add(ce)
	ready := h.Ready(now.Add(10 * time.Second))
	payloads := ready.Payloads(true)
	if len(payloads) != 1 {
		t.Fatal(payloads)
	}
	p := payloads[0]
	if m := p.EndpointMethod(); m != "custom_event_data" {
		t.Error(m)
	}
	data, err := p.Data("agentRunID", now)
	if nil != err || nil == data {
		t.Error(err, data)
	}
	if h.CustomEvents.capacity() != 3 || h.CustomEvents.NumSaved() != 0 {
		t.Fatal("custom events not correctly reset")
	}
	expectCustomEvents(t, ready.CustomEvents, []internal.WantEvent{{
		Intrinsics:     map[string]interface{}{"type": "myEvent", "timestamp": internal.MatchAnything},
		UserAttributes: params,
	}})
	expectMetrics(t, h.Metrics, []internal.WantMetric{
		{customEventsSeen, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{customEventsSent, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestHarvestTxnEventsReady(t *testing.T) {
	now := time.Now()
	fixedHarvestTypes := harvestMetricsTraces & harvestCustomEvents & harvestSpanEvents & harvestErrorEvents
	h := newHarvest(now, &dfltHarvestCfgr{
		reportPeriods: map[harvestTypes]time.Duration{
			fixedHarvestTypes: fixedHarvestPeriod,
			harvestTxnEvents:  time.Second * 5,
		},
		maxTxnEvents: &three,
	})
	h.TxnEvents.AddTxnEvent(&txnEvent{
		FinalName: "finalName",
		Start:     time.Now(),
		Duration:  1 * time.Second,
		TotalTime: 2 * time.Second,
	}, 0)
	ready := h.Ready(now.Add(10 * time.Second))
	payloads := ready.Payloads(true)
	if len(payloads) != 1 {
		t.Fatal(payloads)
	}
	p := payloads[0]
	if m := p.EndpointMethod(); m != "analytic_event_data" {
		t.Error(m)
	}
	data, err := p.Data("agentRunID", now)
	if nil != err || nil == data {
		t.Error(err, data)
	}
	if h.TxnEvents.capacity() != 3 || h.TxnEvents.NumSaved() != 0 {
		t.Fatal("txn events not correctly reset")
	}
	expectTxnEvents(t, ready.TxnEvents, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":      "finalName",
			"totalTime": 2.0,
		},
	}})
	expectMetrics(t, h.Metrics, []internal.WantMetric{
		{txnEventsSeen, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{txnEventsSent, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestHarvestErrorEventsReady(t *testing.T) {
	now := time.Now()
	fixedHarvestTypes := harvestMetricsTraces & harvestCustomEvents & harvestSpanEvents & harvestTxnEvents
	h := newHarvest(now, &dfltHarvestCfgr{
		reportPeriods: map[harvestTypes]time.Duration{
			fixedHarvestTypes:  fixedHarvestPeriod,
			harvestErrorEvents: time.Second * 5,
		},
		maxErrorEvents: &three,
	})
	h.ErrorEvents.Add(&errorEvent{
		errorData: errorData{Klass: "klass", Msg: "msg", When: time.Now()},
		txnEvent:  txnEvent{FinalName: "finalName", Duration: 1 * time.Second},
	}, 0)
	ready := h.Ready(now.Add(10 * time.Second))
	payloads := ready.Payloads(true)
	if len(payloads) != 1 {
		t.Fatal(payloads)
	}
	p := payloads[0]
	if m := p.EndpointMethod(); m != "error_event_data" {
		t.Error(m)
	}
	data, err := p.Data("agentRunID", now)
	if nil != err || nil == data {
		t.Error(err, data)
	}
	if h.ErrorEvents.capacity() != 3 || h.ErrorEvents.NumSaved() != 0 {
		t.Fatal("error events not correctly reset")
	}
	expectErrorEvents(t, ready.ErrorEvents, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "klass",
			"error.message":   "msg",
			"transactionName": "finalName",
		},
	}})
	expectMetrics(t, h.Metrics, []internal.WantMetric{
		{errorEventsSeen, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{errorEventsSent, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestHarvestSpanEventsReady(t *testing.T) {
	now := time.Now()
	fixedHarvestTypes := harvestMetricsTraces & harvestCustomEvents & harvestTxnEvents & harvestErrorEvents
	h := newHarvest(now, &dfltHarvestCfgr{
		reportPeriods: map[harvestTypes]time.Duration{
			fixedHarvestTypes: fixedHarvestPeriod,
			harvestSpanEvents: time.Second * 5,
		},
		maxSpanEvents: &three,
	})
	h.SpanEvents.addEventPopulated(&sampleSpanEvent)
	ready := h.Ready(now.Add(10 * time.Second))
	payloads := ready.Payloads(true)
	if len(payloads) != 1 {
		t.Fatal(payloads)
	}
	p := payloads[0]
	if m := p.EndpointMethod(); m != "span_event_data" {
		t.Error(m)
	}
	data, err := p.Data("agentRunID", now)
	if nil != err || nil == data {
		t.Error(err, data)
	}
	if h.SpanEvents.capacity() != 3 || h.SpanEvents.NumSaved() != 0 {
		t.Fatal("span events not correctly reset")
	}
	expectSpanEvents(t, ready.SpanEvents, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"type":          "Span",
			"name":          "myName",
			"sampled":       true,
			"priority":      0.5,
			"category":      spanCategoryGeneric,
			"nr.entryPoint": true,
			"guid":          "guid",
			"transactionId": "txn-id",
			"traceId":       "trace-id",
		},
	}})
	expectMetrics(t, h.Metrics, []internal.WantMetric{
		{spanEventsSeen, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{spanEventsSent, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestHarvestMetricsTracesReady(t *testing.T) {
	now := time.Now()
	h := newHarvest(now, &dfltHarvestCfgr{
		reportPeriods: map[harvestTypes]time.Duration{
			harvestMetricsTraces: fixedHarvestPeriod,
			harvestTypesEvents:   time.Second * 65,
		},
		maxTxnEvents:    &one,
		maxCustomEvents: &one,
		maxErrorEvents:  &one,
		maxSpanEvents:   &one,
	})
	h.Metrics.addCount("zip", 1, forced)

	ers := newTxnErrors(10)
	ers.Add(errorData{When: time.Now(), Msg: "msg", Klass: "klass", Stack: getStackTrace()})
	mergeTxnErrors(&h.ErrorTraces, ers, txnEvent{FinalName: "finalName", Attrs: nil})

	h.TxnTraces.Witness(harvestTrace{
		txnEvent: txnEvent{
			Start:     time.Now(),
			Duration:  20 * time.Second,
			TotalTime: 30 * time.Second,
			FinalName: "WebTransaction/Go/hello",
		},
		Trace: txnTrace{},
	})

	slows := newSlowQueries(maxTxnSlowQueries)
	slows.observeInstance(slowQueryInstance{
		Duration:           2 * time.Second,
		DatastoreMetric:    "Datastore/statement/MySQL/users/INSERT",
		ParameterizedQuery: "INSERT users",
	})
	h.SlowSQLs.Merge(slows, txnEvent{FinalName: "finalName", Attrs: nil})

	ready := h.Ready(now.Add(61 * time.Second))
	payloads := ready.Payloads(true)
	if len(payloads) != 4 {
		t.Fatal(payloads)
	}

	expectMetrics(t, ready.Metrics, []internal.WantMetric{
		{"zip", "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
	expectMetrics(t, h.Metrics, []internal.WantMetric{})

	expectErrors(t, ready.ErrorTraces, []internal.WantError{{
		TxnName: "finalName",
		Msg:     "msg",
		Klass:   "klass",
	}})
	expectErrors(t, h.ErrorTraces, []internal.WantError{})

	expectSlowQueries(t, ready.SlowSQLs, []internal.WantSlowQuery{{
		Count:      1,
		MetricName: "Datastore/statement/MySQL/users/INSERT",
		Query:      "INSERT users",
		TxnName:    "finalName",
	}})
	expectSlowQueries(t, h.SlowSQLs, []internal.WantSlowQuery{})

	expectTxnTraces(t, ready.TxnTraces, []internal.WantTxnTrace{{
		MetricName: "WebTransaction/Go/hello",
	}})
	expectTxnTraces(t, h.TxnTraces, []internal.WantTxnTrace{})
}

func TestMergeFailedHarvest(t *testing.T) {
	start1 := time.Now()
	start2 := start1.Add(1 * time.Minute)

	h := newHarvest(start1, &dfltHarvestCfgr{})
	h.Metrics.addCount("zip", 1, forced)
	h.TxnEvents.AddTxnEvent(&txnEvent{
		FinalName: "finalName",
		Start:     time.Now(),
		Duration:  1 * time.Second,
		TotalTime: 2 * time.Second,
	}, 0)
	customEventParams := map[string]interface{}{"zip": 1}
	ce, err := createCustomEvent("myEvent", customEventParams, time.Now())
	if nil != err {
		t.Fatal(err)
	}
	h.CustomEvents.Add(ce)
	h.ErrorEvents.Add(&errorEvent{
		errorData: errorData{
			Klass: "klass",
			Msg:   "msg",
			When:  time.Now(),
		},
		txnEvent: txnEvent{
			FinalName: "finalName",
			Duration:  1 * time.Second,
		},
	}, 0)

	ers := newTxnErrors(10)
	ers.Add(errorData{
		When:  time.Now(),
		Msg:   "msg",
		Klass: "klass",
		Stack: getStackTrace(),
	})
	mergeTxnErrors(&h.ErrorTraces, ers, txnEvent{
		FinalName: "finalName",
		Attrs:     nil,
	})
	h.SpanEvents.addEventPopulated(&sampleSpanEvent)

	if start1 != h.Metrics.metricPeriodStart {
		t.Error(h.Metrics.metricPeriodStart)
	}
	if 0 != h.Metrics.failedHarvests {
		t.Error(h.Metrics.failedHarvests)
	}
	if 0 != h.CustomEvents.analyticsEvents.failedHarvests {
		t.Error(h.CustomEvents.analyticsEvents.failedHarvests)
	}
	if 0 != h.TxnEvents.analyticsEvents.failedHarvests {
		t.Error(h.TxnEvents.analyticsEvents.failedHarvests)
	}
	if 0 != h.ErrorEvents.analyticsEvents.failedHarvests {
		t.Error(h.ErrorEvents.analyticsEvents.failedHarvests)
	}
	if 0 != h.SpanEvents.analyticsEvents.failedHarvests {
		t.Error(h.SpanEvents.analyticsEvents.failedHarvests)
	}
	expectMetrics(t, h.Metrics, []internal.WantMetric{
		{"zip", "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
	expectCustomEvents(t, h.CustomEvents, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"type":      "myEvent",
			"timestamp": internal.MatchAnything,
		},
		UserAttributes: customEventParams,
	}})
	expectErrorEvents(t, h.ErrorEvents, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "klass",
			"error.message":   "msg",
			"transactionName": "finalName",
		},
	}})
	expectTxnEvents(t, h.TxnEvents, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":      "finalName",
			"totalTime": 2.0,
		},
	}})
	expectSpanEvents(t, h.SpanEvents, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"type":          "Span",
			"name":          "myName",
			"sampled":       true,
			"priority":      0.5,
			"category":      spanCategoryGeneric,
			"nr.entryPoint": true,
			"guid":          "guid",
			"transactionId": "txn-id",
			"traceId":       "trace-id",
		},
	}})
	expectErrors(t, h.ErrorTraces, []internal.WantError{{
		TxnName: "finalName",
		Msg:     "msg",
		Klass:   "klass",
	}})

	nextHarvest := newHarvest(start2, &dfltHarvestCfgr{})
	if start2 != nextHarvest.Metrics.metricPeriodStart {
		t.Error(nextHarvest.Metrics.metricPeriodStart)
	}
	payloads := h.Payloads(true)
	for _, p := range payloads {
		p.MergeIntoHarvest(nextHarvest)
	}

	if start1 != nextHarvest.Metrics.metricPeriodStart {
		t.Error(nextHarvest.Metrics.metricPeriodStart)
	}
	if 1 != nextHarvest.Metrics.failedHarvests {
		t.Error(nextHarvest.Metrics.failedHarvests)
	}
	if 1 != nextHarvest.CustomEvents.analyticsEvents.failedHarvests {
		t.Error(nextHarvest.CustomEvents.analyticsEvents.failedHarvests)
	}
	if 1 != nextHarvest.TxnEvents.analyticsEvents.failedHarvests {
		t.Error(nextHarvest.TxnEvents.analyticsEvents.failedHarvests)
	}
	if 1 != nextHarvest.ErrorEvents.analyticsEvents.failedHarvests {
		t.Error(nextHarvest.ErrorEvents.analyticsEvents.failedHarvests)
	}
	if 1 != nextHarvest.SpanEvents.analyticsEvents.failedHarvests {
		t.Error(nextHarvest.SpanEvents.analyticsEvents.failedHarvests)
	}
	expectMetrics(t, nextHarvest.Metrics, []internal.WantMetric{
		{"zip", "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
	expectCustomEvents(t, nextHarvest.CustomEvents, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"type":      "myEvent",
			"timestamp": internal.MatchAnything,
		},
		UserAttributes: customEventParams,
	}})
	expectErrorEvents(t, nextHarvest.ErrorEvents, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "klass",
			"error.message":   "msg",
			"transactionName": "finalName",
		},
	}})
	expectTxnEvents(t, nextHarvest.TxnEvents, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":      "finalName",
			"totalTime": 2.0,
		},
	}})
	expectSpanEvents(t, h.SpanEvents, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"type":          "Span",
			"name":          "myName",
			"sampled":       true,
			"priority":      0.5,
			"category":      spanCategoryGeneric,
			"nr.entryPoint": true,
			"guid":          "guid",
			"transactionId": "txn-id",
			"traceId":       "trace-id",
		},
	}})
	expectErrors(t, nextHarvest.ErrorTraces, []internal.WantError{})
}

func TestCreateTxnMetrics(t *testing.T) {
	txnErr := &errorData{}
	txnErrors := []*errorData{txnErr}
	webName := "WebTransaction/zip/zap"
	backgroundName := "OtherTransaction/zip/zap"
	args := &txnData{}
	args.Duration = 123 * time.Second
	args.TotalTime = 150 * time.Second
	args.ApdexThreshold = 2 * time.Second

	args.BetterCAT.Enabled = true

	args.FinalName = webName
	args.IsWeb = true
	args.Errors = txnErrors
	args.Zone = apdexTolerating
	metrics := newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{webName, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{webRollup, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{dispatcherMetric, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{"WebTransactionTotalTime", "", true, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{"WebTransactionTotalTime/zip/zap", "", false, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Errors/allWeb", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Errors/" + webName, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{apdexRollup, "", true, []float64{0, 1, 0, 2, 2, 0}},
		{"Apdex/zip/zap", "", false, []float64{0, 1, 0, 2, 2, 0}},
		{"DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", "", false, []float64{1, 123, 123, 123, 123, 123 * 123}},
		{"DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", "", false, []float64{1, 123, 123, 123, 123, 123 * 123}},
		{"ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/all", "", false, []float64{1, 0, 0, 0, 0, 0}},
		{"ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", "", false, []float64{1, 0, 0, 0, 0, 0}},
	})

	args.FinalName = webName
	args.IsWeb = true
	args.Errors = nil
	args.Zone = apdexTolerating
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{webName, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{webRollup, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{dispatcherMetric, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{"WebTransactionTotalTime", "", true, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{"WebTransactionTotalTime/zip/zap", "", false, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{apdexRollup, "", true, []float64{0, 1, 0, 2, 2, 0}},
		{"Apdex/zip/zap", "", false, []float64{0, 1, 0, 2, 2, 0}},
		{"DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", "", false, []float64{1, 123, 123, 123, 123, 123 * 123}},
		{"DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", "", false, []float64{1, 123, 123, 123, 123, 123 * 123}},
	})

	args.FinalName = backgroundName
	args.IsWeb = false
	args.Errors = txnErrors
	args.Zone = apdexNone
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{backgroundName, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{backgroundRollup, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{"OtherTransactionTotalTime", "", true, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{"OtherTransactionTotalTime/zip/zap", "", false, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Errors/" + backgroundName, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", "", false, []float64{1, 123, 123, 123, 123, 123 * 123}},
		{"DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", "", false, []float64{1, 123, 123, 123, 123, 123 * 123}},
		{"ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/all", "", false, []float64{1, 0, 0, 0, 0, 0}},
		{"ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/allOther", "", false, []float64{1, 0, 0, 0, 0, 0}},
	})

	args.FinalName = backgroundName
	args.IsWeb = false
	args.Errors = nil
	args.Zone = apdexNone
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{backgroundName, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{backgroundRollup, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{"OtherTransactionTotalTime", "", true, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{"OtherTransactionTotalTime/zip/zap", "", false, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{"DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", "", false, []float64{1, 123, 123, 123, 123, 123 * 123}},
		{"DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", "", false, []float64{1, 123, 123, 123, 123, 123 * 123}},
	})

}

func TestHarvestSplitTxnEvents(t *testing.T) {
	now := time.Now()
	h := newHarvest(now, &dfltHarvestCfgr{})
	for i := 0; i < internal.MaxTxnEvents; i++ {
		h.TxnEvents.AddTxnEvent(&txnEvent{}, priority(float32(i)))
	}

	payloadsWithSplit := h.Payloads(true)
	payloadsWithoutSplit := h.Payloads(false)

	if len(payloadsWithSplit) != 9 {
		t.Error(len(payloadsWithSplit))
	}
	if len(payloadsWithoutSplit) != 8 {
		t.Error(len(payloadsWithoutSplit))
	}
}

func TestCreateTxnMetricsOldCAT(t *testing.T) {
	txnErr := &errorData{}
	txnErrors := []*errorData{txnErr}
	webName := "WebTransaction/zip/zap"
	backgroundName := "OtherTransaction/zip/zap"
	args := &txnData{}
	args.Duration = 123 * time.Second
	args.TotalTime = 150 * time.Second
	args.ApdexThreshold = 2 * time.Second

	// When BetterCAT is disabled, affirm that the caller metrics are not created.
	args.BetterCAT.Enabled = false

	args.FinalName = webName
	args.IsWeb = true
	args.Errors = txnErrors
	args.Zone = apdexTolerating
	metrics := newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{webName, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{webRollup, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{dispatcherMetric, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{"WebTransactionTotalTime", "", true, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{"WebTransactionTotalTime/zip/zap", "", false, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Errors/allWeb", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Errors/" + webName, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{apdexRollup, "", true, []float64{0, 1, 0, 2, 2, 0}},
		{"Apdex/zip/zap", "", false, []float64{0, 1, 0, 2, 2, 0}},
	})

	args.FinalName = webName
	args.IsWeb = true
	args.Errors = nil
	args.Zone = apdexTolerating
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{webName, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{webRollup, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{dispatcherMetric, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{"WebTransactionTotalTime", "", true, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{"WebTransactionTotalTime/zip/zap", "", false, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{apdexRollup, "", true, []float64{0, 1, 0, 2, 2, 0}},
		{"Apdex/zip/zap", "", false, []float64{0, 1, 0, 2, 2, 0}},
	})

	args.FinalName = backgroundName
	args.IsWeb = false
	args.Errors = txnErrors
	args.Zone = apdexNone
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{backgroundName, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{backgroundRollup, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{"OtherTransactionTotalTime", "", true, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{"OtherTransactionTotalTime/zip/zap", "", false, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{"Errors/all", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Errors/allOther", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Errors/" + backgroundName, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})

	args.FinalName = backgroundName
	args.IsWeb = false
	args.Errors = nil
	args.Zone = apdexNone
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{backgroundName, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{backgroundRollup, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{"OtherTransactionTotalTime", "", true, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{"OtherTransactionTotalTime/zip/zap", "", false, []float64{1, 150, 150, 150, 150, 150 * 150}},
	})
}

func TestNewHarvestSetsDefaultValues(t *testing.T) {
	now := time.Now()
	h := newHarvest(now, &dfltHarvestCfgr{})

	if cp := h.TxnEvents.capacity(); cp != internal.MaxTxnEvents {
		t.Error("wrong txn event capacity", cp)
	}
	if cp := h.CustomEvents.capacity(); cp != internal.MaxCustomEvents {
		t.Error("wrong custom event capacity", cp)
	}
	if cp := h.ErrorEvents.capacity(); cp != internal.MaxErrorEvents {
		t.Error("wrong error event capacity", cp)
	}
	if cp := h.SpanEvents.capacity(); cp != maxSpanEvents {
		t.Error("wrong span event capacity", cp)
	}
}

func TestNewHarvestUsesConnectReply(t *testing.T) {
	now := time.Now()
	h := newHarvest(now, &dfltHarvestCfgr{
		reportPeriods: map[harvestTypes]time.Duration{
			harvestMetricsTraces: fixedHarvestPeriod,
			harvestTypesEvents:   time.Second * 5,
		},
		maxTxnEvents:    &one,
		maxCustomEvents: &two,
		maxErrorEvents:  &three,
		maxSpanEvents:   &four,
	})

	if cp := h.TxnEvents.capacity(); cp != 1 {
		t.Error("wrong txn event capacity", cp)
	}
	if cp := h.CustomEvents.capacity(); cp != 2 {
		t.Error("wrong custom event capacity", cp)
	}
	if cp := h.ErrorEvents.capacity(); cp != 3 {
		t.Error("wrong error event capacity", cp)
	}
	if cp := h.SpanEvents.capacity(); cp != 4 {
		t.Error("wrong span event capacity", cp)
	}
}

func TestConfigurableHarvestZeroHarvestLimits(t *testing.T) {
	now := time.Now()

	var zero uint
	h := newHarvest(now, &dfltHarvestCfgr{
		reportPeriods: map[harvestTypes]time.Duration{
			harvestMetricsTraces: fixedHarvestPeriod,
			harvestTypesEvents:   time.Second * 5,
		},
		maxTxnEvents:    &zero,
		maxCustomEvents: &zero,
		maxErrorEvents:  &zero,
		maxSpanEvents:   &zero,
	})
	if cp := h.TxnEvents.capacity(); cp != 0 {
		t.Error("wrong txn event capacity", cp)
	}
	if cp := h.CustomEvents.capacity(); cp != 0 {
		t.Error("wrong custom event capacity", cp)
	}
	if cp := h.ErrorEvents.capacity(); cp != 0 {
		t.Error("wrong error event capacity", cp)
	}
	if cp := h.SpanEvents.capacity(); cp != 0 {
		t.Error("wrong error event capacity", cp)
	}

	// Add events to ensure that adding events to zero-capacity pools is
	// safe.
	h.TxnEvents.AddTxnEvent(&txnEvent{}, 1.0)
	h.CustomEvents.Add(&customEvent{})
	h.ErrorEvents.Add(&errorEvent{}, 1.0)
	h.SpanEvents.addEventPopulated(&sampleSpanEvent)

	// Create the payloads to ensure doing so with zero-capacity pools is
	// safe.
	payloads := h.Ready(now.Add(2 * time.Minute)).Payloads(false)
	for _, p := range payloads {
		js, err := p.Data("agentRunID", now.Add(2*time.Minute))
		if nil != err {
			t.Error(err)
			continue
		}
		// Only metric data should be present.
		if (p.EndpointMethod() == "metric_data") !=
			(string(js) != "") {
			t.Error(p.EndpointMethod(), string(js))
		}
	}
}
