// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"
	"time"
)

func TestHarvestTimerAllFixed(t *testing.T) {
	now := time.Now()
	harvest := NewHarvest(now, &DfltHarvestCfgr{})
	timer := harvest.timer
	for _, tc := range []struct {
		Elapsed time.Duration
		Expect  HarvestTypes
	}{
		{60 * time.Second, 0},
		{61 * time.Second, HarvestTypesAll},
		{62 * time.Second, 0},
		{120 * time.Second, 0},
		{121 * time.Second, HarvestTypesAll},
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
	harvest := NewHarvest(now, &DfltHarvestCfgr{
		reportPeriods: map[HarvestTypes]time.Duration{
			HarvestMetricsTraces: FixedHarvestPeriod,
			HarvestTypesEvents:   time.Second * 30,
		},
		maxTxnEvents:    &one,
		maxCustomEvents: &two,
		maxSpanEvents:   &three,
		maxErrorEvents:  &four,
	})
	timer := harvest.timer
	for _, tc := range []struct {
		Elapsed time.Duration
		Expect  HarvestTypes
	}{
		{30 * time.Second, 0},
		{31 * time.Second, HarvestTypesEvents},
		{32 * time.Second, 0},
		{61 * time.Second, HarvestTypesAll},
		{62 * time.Second, 0},
		{91 * time.Second, HarvestTypesEvents},
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
	var nilHarvest *Harvest
	nilHarvest.CreateFinalMetrics(nil, &DfltHarvestCfgr{})
	emptyHarvest := &Harvest{}
	emptyHarvest.CreateFinalMetrics(nil, &DfltHarvestCfgr{})

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
	reply, err := ConstructConnectReply(replyJSON, PreconnectReply{})
	if err != nil {
		t.Fatal(err)
	}
	var txnEvents uint = 22
	var customEvents uint = 33
	var errorEvents uint = 44
	var spanEvents uint = 55
	cfgr := &DfltHarvestCfgr{
		reportPeriods: map[HarvestTypes]time.Duration{
			HarvestMetricsTraces: FixedHarvestPeriod,
			HarvestTypesEvents:   time.Second * 2,
		},
		maxTxnEvents:    &txnEvents,
		maxCustomEvents: &customEvents,
		maxErrorEvents:  &errorEvents,
		maxSpanEvents:   &spanEvents,
	}
	h := NewHarvest(now, cfgr)
	h.Metrics.addCount("rename_me", 1.0, unforced)
	h.CreateFinalMetrics(reply, cfgr)
	ExpectMetrics(t, h.Metrics, []WantMetric{
		{instanceReporting, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"been_renamed", "", false, []float64{1.0, 0, 0, 0, 0, 0}},
		{"Supportability/EventHarvest/ReportPeriod", "", true, []float64{1, 2, 2, 2, 2, 2 * 2}},
		{"Supportability/EventHarvest/AnalyticEventData/HarvestLimit", "", true, []float64{1, 22, 22, 22, 22, 22 * 22}},
		{"Supportability/EventHarvest/CustomEventData/HarvestLimit", "", true, []float64{1, 33, 33, 33, 33, 33 * 33}},
		{"Supportability/EventHarvest/ErrorEventData/HarvestLimit", "", true, []float64{1, 44, 44, 44, 44, 44 * 44}},
		{"Supportability/EventHarvest/SpanEventData/HarvestLimit", "", true, []float64{1, 55, 55, 55, 55, 55 * 55}},
	})

	// Test again without any metric rules or event_harvest_config.

	replyJSON = []byte(`{"return_value":{
	}}`)
	reply, err = ConstructConnectReply(replyJSON, PreconnectReply{})
	if err != nil {
		t.Fatal(err)
	}
	h = NewHarvest(now, &DfltHarvestCfgr{})
	h.Metrics.addCount("rename_me", 1.0, unforced)
	h.CreateFinalMetrics(reply, &DfltHarvestCfgr{})
	ExpectMetrics(t, h.Metrics, []WantMetric{
		{instanceReporting, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"rename_me", "", false, []float64{1.0, 0, 0, 0, 0, 0}},
		{"Supportability/EventHarvest/ReportPeriod", "", true, []float64{1, 60, 60, 60, 60, 60 * 60}},
		{"Supportability/EventHarvest/AnalyticEventData/HarvestLimit", "", true, []float64{1, 10 * 1000, 10 * 1000, 10 * 1000, 10 * 1000, 10 * 1000 * 10 * 1000}},
		{"Supportability/EventHarvest/CustomEventData/HarvestLimit", "", true, []float64{1, 10 * 1000, 10 * 1000, 10 * 1000, 10 * 1000, 10 * 1000 * 10 * 1000}},
		{"Supportability/EventHarvest/ErrorEventData/HarvestLimit", "", true, []float64{1, 100, 100, 100, 100, 100 * 100}},
		{"Supportability/EventHarvest/SpanEventData/HarvestLimit", "", true, []float64{1, 1000, 1000, 1000, 1000, 1000 * 1000}},
	})
}

func TestEmptyPayloads(t *testing.T) {
	h := NewHarvest(time.Now(), &DfltHarvestCfgr{})
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
	var nilHarvest *Harvest
	payloads := nilHarvest.Payloads(true)
	if len(payloads) != 0 {
		t.Error(len(payloads))
	}
}

func TestPayloadsEmptyHarvest(t *testing.T) {
	h := &Harvest{}
	payloads := h.Payloads(true)
	if len(payloads) != 0 {
		t.Error(len(payloads))
	}
}

func TestHarvestNothingReady(t *testing.T) {
	now := time.Now()
	h := NewHarvest(now, &DfltHarvestCfgr{})
	ready := h.Ready(now.Add(10 * time.Second))
	if ready != nil {
		t.Error("harvest should be nil")
	}
	payloads := ready.Payloads(true)
	if len(payloads) != 0 {
		t.Error(payloads)
	}
	ExpectMetrics(t, h.Metrics, []WantMetric{})
}

func TestHarvestCustomEventsReady(t *testing.T) {
	now := time.Now()
	fixedHarvestTypes := HarvestMetricsTraces & HarvestTxnEvents & HarvestSpanEvents & HarvestErrorEvents
	h := NewHarvest(now, &DfltHarvestCfgr{
		reportPeriods: map[HarvestTypes]time.Duration{
			fixedHarvestTypes:   FixedHarvestPeriod,
			HarvestCustomEvents: time.Second * 5,
		},
		maxCustomEvents: &three,
	})
	params := map[string]interface{}{"zip": 1}
	ce, _ := CreateCustomEvent("myEvent", params, time.Now())
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
	ExpectCustomEvents(t, ready.CustomEvents, []WantEvent{{
		Intrinsics:     map[string]interface{}{"type": "myEvent", "timestamp": MatchAnything},
		UserAttributes: params,
	}})
	ExpectMetrics(t, h.Metrics, []WantMetric{
		{customEventsSeen, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{customEventsSent, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestHarvestTxnEventsReady(t *testing.T) {
	now := time.Now()
	fixedHarvestTypes := HarvestMetricsTraces & HarvestCustomEvents & HarvestSpanEvents & HarvestErrorEvents
	h := NewHarvest(now, &DfltHarvestCfgr{
		reportPeriods: map[HarvestTypes]time.Duration{
			fixedHarvestTypes: FixedHarvestPeriod,
			HarvestTxnEvents:  time.Second * 5,
		},
		maxTxnEvents: &three,
	})
	h.TxnEvents.AddTxnEvent(&TxnEvent{
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
	ExpectTxnEvents(t, ready.TxnEvents, []WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":      "finalName",
			"totalTime": 2.0,
		},
	}})
	ExpectMetrics(t, h.Metrics, []WantMetric{
		{txnEventsSeen, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{txnEventsSent, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestHarvestErrorEventsReady(t *testing.T) {
	now := time.Now()
	fixedHarvestTypes := HarvestMetricsTraces & HarvestCustomEvents & HarvestSpanEvents & HarvestTxnEvents
	h := NewHarvest(now, &DfltHarvestCfgr{
		reportPeriods: map[HarvestTypes]time.Duration{
			fixedHarvestTypes:  FixedHarvestPeriod,
			HarvestErrorEvents: time.Second * 5,
		},
		maxErrorEvents: &three,
	})
	h.ErrorEvents.Add(&ErrorEvent{
		ErrorData: ErrorData{Klass: "klass", Msg: "msg", When: time.Now()},
		TxnEvent:  TxnEvent{FinalName: "finalName", Duration: 1 * time.Second},
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
	ExpectErrorEvents(t, ready.ErrorEvents, []WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "klass",
			"error.message":   "msg",
			"transactionName": "finalName",
		},
	}})
	ExpectMetrics(t, h.Metrics, []WantMetric{
		{errorEventsSeen, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{errorEventsSent, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestHarvestSpanEventsReady(t *testing.T) {
	now := time.Now()
	fixedHarvestTypes := HarvestMetricsTraces & HarvestCustomEvents & HarvestTxnEvents & HarvestErrorEvents
	h := NewHarvest(now, &DfltHarvestCfgr{
		reportPeriods: map[HarvestTypes]time.Duration{
			fixedHarvestTypes: FixedHarvestPeriod,
			HarvestSpanEvents: time.Second * 5,
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
	ExpectSpanEvents(t, ready.SpanEvents, []WantEvent{{
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
	ExpectMetrics(t, h.Metrics, []WantMetric{
		{spanEventsSeen, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{spanEventsSent, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestHarvestMetricsTracesReady(t *testing.T) {
	now := time.Now()
	h := NewHarvest(now, &DfltHarvestCfgr{
		reportPeriods: map[HarvestTypes]time.Duration{
			HarvestMetricsTraces: FixedHarvestPeriod,
			HarvestTypesEvents:   time.Second * 65,
		},
		maxTxnEvents:    &one,
		maxCustomEvents: &one,
		maxErrorEvents:  &one,
		maxSpanEvents:   &one,
	})
	h.Metrics.addCount("zip", 1, forced)

	ers := NewTxnErrors(10)
	ers.Add(ErrorData{When: time.Now(), Msg: "msg", Klass: "klass", Stack: GetStackTrace()})
	MergeTxnErrors(&h.ErrorTraces, ers, TxnEvent{FinalName: "finalName", Attrs: nil})

	h.TxnTraces.Witness(HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     time.Now(),
			Duration:  20 * time.Second,
			TotalTime: 30 * time.Second,
			FinalName: "WebTransaction/Go/hello",
		},
		Trace: TxnTrace{},
	})

	slows := newSlowQueries(maxTxnSlowQueries)
	slows.observeInstance(slowQueryInstance{
		Duration:           2 * time.Second,
		DatastoreMetric:    "Datastore/statement/MySQL/users/INSERT",
		ParameterizedQuery: "INSERT users",
	})
	h.SlowSQLs.Merge(slows, TxnEvent{FinalName: "finalName", Attrs: nil})

	ready := h.Ready(now.Add(61 * time.Second))
	payloads := ready.Payloads(true)
	if len(payloads) != 4 {
		t.Fatal(payloads)
	}

	ExpectMetrics(t, ready.Metrics, []WantMetric{
		{"zip", "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
	ExpectMetrics(t, h.Metrics, []WantMetric{})

	ExpectErrors(t, ready.ErrorTraces, []WantError{{
		TxnName: "finalName",
		Msg:     "msg",
		Klass:   "klass",
	}})
	ExpectErrors(t, h.ErrorTraces, []WantError{})

	ExpectSlowQueries(t, ready.SlowSQLs, []WantSlowQuery{{
		Count:      1,
		MetricName: "Datastore/statement/MySQL/users/INSERT",
		Query:      "INSERT users",
		TxnName:    "finalName",
	}})
	ExpectSlowQueries(t, h.SlowSQLs, []WantSlowQuery{})

	ExpectTxnTraces(t, ready.TxnTraces, []WantTxnTrace{{
		MetricName: "WebTransaction/Go/hello",
	}})
	ExpectTxnTraces(t, h.TxnTraces, []WantTxnTrace{})
}

func TestMergeFailedHarvest(t *testing.T) {
	start1 := time.Now()
	start2 := start1.Add(1 * time.Minute)

	h := NewHarvest(start1, &DfltHarvestCfgr{})
	h.Metrics.addCount("zip", 1, forced)
	h.TxnEvents.AddTxnEvent(&TxnEvent{
		FinalName: "finalName",
		Start:     time.Now(),
		Duration:  1 * time.Second,
		TotalTime: 2 * time.Second,
	}, 0)
	customEventParams := map[string]interface{}{"zip": 1}
	ce, err := CreateCustomEvent("myEvent", customEventParams, time.Now())
	if nil != err {
		t.Fatal(err)
	}
	h.CustomEvents.Add(ce)
	h.ErrorEvents.Add(&ErrorEvent{
		ErrorData: ErrorData{
			Klass: "klass",
			Msg:   "msg",
			When:  time.Now(),
		},
		TxnEvent: TxnEvent{
			FinalName: "finalName",
			Duration:  1 * time.Second,
		},
	}, 0)

	ers := NewTxnErrors(10)
	ers.Add(ErrorData{
		When:  time.Now(),
		Msg:   "msg",
		Klass: "klass",
		Stack: GetStackTrace(),
	})
	MergeTxnErrors(&h.ErrorTraces, ers, TxnEvent{
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
	ExpectMetrics(t, h.Metrics, []WantMetric{
		{"zip", "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
	ExpectCustomEvents(t, h.CustomEvents, []WantEvent{{
		Intrinsics: map[string]interface{}{
			"type":      "myEvent",
			"timestamp": MatchAnything,
		},
		UserAttributes: customEventParams,
	}})
	ExpectErrorEvents(t, h.ErrorEvents, []WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "klass",
			"error.message":   "msg",
			"transactionName": "finalName",
		},
	}})
	ExpectTxnEvents(t, h.TxnEvents, []WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":      "finalName",
			"totalTime": 2.0,
		},
	}})
	ExpectSpanEvents(t, h.SpanEvents, []WantEvent{{
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
	ExpectErrors(t, h.ErrorTraces, []WantError{{
		TxnName: "finalName",
		Msg:     "msg",
		Klass:   "klass",
	}})

	nextHarvest := NewHarvest(start2, &DfltHarvestCfgr{})
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
	ExpectMetrics(t, nextHarvest.Metrics, []WantMetric{
		{"zip", "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
	ExpectCustomEvents(t, nextHarvest.CustomEvents, []WantEvent{{
		Intrinsics: map[string]interface{}{
			"type":      "myEvent",
			"timestamp": MatchAnything,
		},
		UserAttributes: customEventParams,
	}})
	ExpectErrorEvents(t, nextHarvest.ErrorEvents, []WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "klass",
			"error.message":   "msg",
			"transactionName": "finalName",
		},
	}})
	ExpectTxnEvents(t, nextHarvest.TxnEvents, []WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":      "finalName",
			"totalTime": 2.0,
		},
	}})
	ExpectSpanEvents(t, h.SpanEvents, []WantEvent{{
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
	ExpectErrors(t, nextHarvest.ErrorTraces, []WantError{})
}

func TestCreateTxnMetrics(t *testing.T) {
	txnErr := &ErrorData{}
	txnErrors := []*ErrorData{txnErr}
	webName := "WebTransaction/zip/zap"
	backgroundName := "OtherTransaction/zip/zap"
	args := &TxnData{}
	args.Duration = 123 * time.Second
	args.TotalTime = 150 * time.Second
	args.ApdexThreshold = 2 * time.Second

	args.BetterCAT.Enabled = true

	args.FinalName = webName
	args.IsWeb = true
	args.Errors = txnErrors
	args.Zone = ApdexTolerating
	metrics := newMetricTable(100, time.Now())
	CreateTxnMetrics(args, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
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
	args.Zone = ApdexTolerating
	metrics = newMetricTable(100, time.Now())
	CreateTxnMetrics(args, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
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
	args.Zone = ApdexNone
	metrics = newMetricTable(100, time.Now())
	CreateTxnMetrics(args, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
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
	args.Zone = ApdexNone
	metrics = newMetricTable(100, time.Now())
	CreateTxnMetrics(args, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
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
	h := NewHarvest(now, &DfltHarvestCfgr{})
	for i := 0; i < MaxTxnEvents; i++ {
		h.TxnEvents.AddTxnEvent(&TxnEvent{}, Priority(float32(i)))
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
	txnErr := &ErrorData{}
	txnErrors := []*ErrorData{txnErr}
	webName := "WebTransaction/zip/zap"
	backgroundName := "OtherTransaction/zip/zap"
	args := &TxnData{}
	args.Duration = 123 * time.Second
	args.TotalTime = 150 * time.Second
	args.ApdexThreshold = 2 * time.Second

	// When BetterCAT is disabled, affirm that the caller metrics are not created.
	args.BetterCAT.Enabled = false

	args.FinalName = webName
	args.IsWeb = true
	args.Errors = txnErrors
	args.Zone = ApdexTolerating
	metrics := newMetricTable(100, time.Now())
	CreateTxnMetrics(args, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
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
	args.Zone = ApdexTolerating
	metrics = newMetricTable(100, time.Now())
	CreateTxnMetrics(args, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
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
	args.Zone = ApdexNone
	metrics = newMetricTable(100, time.Now())
	CreateTxnMetrics(args, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
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
	args.Zone = ApdexNone
	metrics = newMetricTable(100, time.Now())
	CreateTxnMetrics(args, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
		{backgroundName, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{backgroundRollup, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{"OtherTransactionTotalTime", "", true, []float64{1, 150, 150, 150, 150, 150 * 150}},
		{"OtherTransactionTotalTime/zip/zap", "", false, []float64{1, 150, 150, 150, 150, 150 * 150}},
	})
}

func TestNewHarvestSetsDefaultValues(t *testing.T) {
	now := time.Now()
	h := NewHarvest(now, &DfltHarvestCfgr{})

	if cp := h.TxnEvents.capacity(); cp != MaxTxnEvents {
		t.Error("wrong txn event capacity", cp)
	}
	if cp := h.CustomEvents.capacity(); cp != MaxCustomEvents {
		t.Error("wrong custom event capacity", cp)
	}
	if cp := h.ErrorEvents.capacity(); cp != MaxErrorEvents {
		t.Error("wrong error event capacity", cp)
	}
	if cp := h.SpanEvents.capacity(); cp != MaxSpanEvents {
		t.Error("wrong span event capacity", cp)
	}
}

func TestNewHarvestUsesConnectReply(t *testing.T) {
	now := time.Now()
	h := NewHarvest(now, &DfltHarvestCfgr{
		reportPeriods: map[HarvestTypes]time.Duration{
			HarvestMetricsTraces: FixedHarvestPeriod,
			HarvestTypesEvents:   time.Second * 5,
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
	h := NewHarvest(now, &DfltHarvestCfgr{
		reportPeriods: map[HarvestTypes]time.Duration{
			HarvestMetricsTraces: FixedHarvestPeriod,
			HarvestTypesEvents:   time.Second * 5,
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
	h.TxnEvents.AddTxnEvent(&TxnEvent{}, 1.0)
	h.CustomEvents.Add(&CustomEvent{})
	h.ErrorEvents.Add(&ErrorEvent{}, 1.0)
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
