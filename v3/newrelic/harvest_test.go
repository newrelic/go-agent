// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/logger"
)

var (
	// This is for testing only
	testHarvestCfgr = generateTestHarvestConfig()
)

func generateTestHarvestConfig() harvestConfig {
	cfg := dfltHarvestCfgr

	// Enable logging features for testing (not enabled by default)
	loggingCfg := loggingConfigEnabled(internal.MaxLogEvents)
	cfg.LoggingConfig = loggingCfg
	return cfg
}

func TestHarvestTimerAllFixed(t *testing.T) {
	now := time.Now()
	harvest := newHarvest(now, testHarvestCfgr)
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

func TestHarvestTimerAllConfigurable(t *testing.T) {
	now := time.Now()
	harvest := newHarvest(now, harvestConfig{
		ReportPeriods: map[harvestTypes]time.Duration{
			harvestMetricsTraces: fixedHarvestPeriod,
			harvestTypesEvents:   time.Second * 30,
		},
		MaxTxnEvents:    1,
		MaxCustomEvents: 2,
		MaxSpanEvents:   3,
		MaxErrorEvents:  4,
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

	config := config{Config: defaultConfig()}

	run := newAppRun(config, internal.ConnectReplyDefaults())
	run.harvestConfig = testHarvestCfgr

	nilHarvest.CreateFinalMetrics(run, nil)
	emptyHarvest := &harvest{}
	emptyHarvest.CreateFinalMetrics(run, nil)

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
				"span_event_data": 55,
				"log_event_data":66
			}
		}
	}}`)
	reply, err := internal.UnmarshalConnectReply(replyJSON, internal.PreconnectReply{})
	if err != nil {
		t.Fatal(err)
	}
	cfgr := harvestConfig{
		ReportPeriods: map[harvestTypes]time.Duration{
			harvestMetricsTraces: fixedHarvestPeriod,
			harvestTypesEvents:   time.Second * 2,
		},
		MaxTxnEvents:    22,
		MaxCustomEvents: 33,
		MaxErrorEvents:  44,
		MaxSpanEvents:   55,
		LoggingConfig:   loggingConfigEnabled(66),
	}
	h := newHarvest(now, cfgr)
	h.Metrics.addCount("rename_me", 1.0, unforced)
	run = newAppRun(config, reply)
	run.harvestConfig = cfgr
	h.CreateFinalMetrics(run, nil)
	expectMetrics(t, h.Metrics, []internal.WantMetric{
		{Name: instanceReporting, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "been_renamed", Scope: "", Forced: false, Data: []float64{1.0, 0, 0, 0, 0, 0}},
		{Name: "Supportability/EventHarvest/ReportPeriod", Scope: "", Forced: true, Data: []float64{1, 2, 2, 2, 2, 2 * 2}},
		{Name: "Supportability/EventHarvest/AnalyticEventData/HarvestLimit", Scope: "", Forced: true, Data: []float64{1, 22, 22, 22, 22, 22 * 22}},
		{Name: "Supportability/EventHarvest/CustomEventData/HarvestLimit", Scope: "", Forced: true, Data: []float64{1, 33, 33, 33, 33, 33 * 33}},
		{Name: "Supportability/EventHarvest/ErrorEventData/HarvestLimit", Scope: "", Forced: true, Data: []float64{1, 44, 44, 44, 44, 44 * 44}},
		{Name: "Supportability/EventHarvest/SpanEventData/HarvestLimit", Scope: "", Forced: true, Data: []float64{1, 55, 55, 55, 55, 55 * 55}},
		{Name: "Supportability/EventHarvest/LogEventData/HarvestLimit", Scope: "", Forced: true, Data: []float64{1, 66, 66, 66, 66, 66 * 66}},
		{Name: "Supportability/Go/Version/" + Version, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Supportability/Go/Runtime/Version/" + goVersionSimple, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Supportability/Go/gRPC/Version/" + grpcVersion, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Supportability/Logging/Golang", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Supportability/Logging/Forwarding/Golang", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Supportability/Logging/Metrics/Golang", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Supportability/Logging/LocalDecorating/Golang", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})

	// Test again without any metric rules or event_harvest_config.

	replyJSON = []byte(`{"return_value":{
	}}`)
	reply, err = internal.UnmarshalConnectReply(replyJSON, internal.PreconnectReply{})
	if err != nil {
		t.Fatal(err)
	}
	run = newAppRun(config, reply)
	run.harvestConfig = testHarvestCfgr
	h = newHarvest(now, testHarvestCfgr)
	h.Metrics.addCount("rename_me", 1.0, unforced)
	h.CreateFinalMetrics(run, nil)
	expectMetrics(t, h.Metrics, []internal.WantMetric{
		{Name: instanceReporting, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "rename_me", Scope: "", Forced: false, Data: []float64{1.0, 0, 0, 0, 0, 0}},
		{Name: "Supportability/EventHarvest/ReportPeriod", Scope: "", Forced: true, Data: []float64{1, 60, 60, 60, 60, 60 * 60}},
		{Name: "Supportability/EventHarvest/AnalyticEventData/HarvestLimit", Scope: "", Forced: true, Data: []float64{1, 10 * 1000, 10 * 1000, 10 * 1000, 10 * 1000, 10 * 1000 * 10 * 1000}},
		{Name: "Supportability/EventHarvest/CustomEventData/HarvestLimit", Scope: "", Forced: true, Data: []float64{1, internal.MaxCustomEvents, internal.MaxCustomEvents, internal.MaxCustomEvents, internal.MaxCustomEvents, internal.MaxCustomEvents * internal.MaxCustomEvents}},
		{Name: "Supportability/EventHarvest/ErrorEventData/HarvestLimit", Scope: "", Forced: true, Data: []float64{1, 100, 100, 100, 100, 100 * 100}},
		{Name: "Supportability/EventHarvest/SpanEventData/HarvestLimit", Scope: "", Forced: true, Data: []float64{1, internal.MaxSpanEvents, internal.MaxSpanEvents, internal.MaxSpanEvents, internal.MaxSpanEvents, internal.MaxSpanEvents * internal.MaxSpanEvents}},
		{Name: "Supportability/EventHarvest/LogEventData/HarvestLimit", Scope: "", Forced: true, Data: []float64{1, 10000, 10000, 10000, 10000, 10000 * 10000}},
		{Name: "Supportability/Go/Version/" + Version, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Supportability/Go/Runtime/Version/" + goVersionSimple, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Supportability/Go/gRPC/Version/" + grpcVersion, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Supportability/Logging/Golang", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Supportability/Logging/Forwarding/Golang", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Supportability/Logging/Metrics/Golang", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Supportability/Logging/LocalDecorating/Golang", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestCreateFinalMetricsTraceObserver(t *testing.T) {
	if !versionSupports8T {
		t.Skip("go version does not support 8T")
	}

	replyJSON := []byte(`{"return_value":{}}`)
	reply, err := internal.UnmarshalConnectReply(replyJSON, internal.PreconnectReply{})
	if err != nil {
		t.Fatal(err)
	}

	run := newAppRun(config{Config: defaultConfig()}, reply)
	run.harvestConfig = testHarvestCfgr

	to, _ := newTraceObserver(
		internal.AgentRunID("runid"), nil,
		observerConfig{
			log: logger.ShimLogger{},
		},
	)
	h := newHarvest(now, testHarvestCfgr)
	h.CreateFinalMetrics(run, to)
	expectMetrics(t, h.Metrics, []internal.WantMetric{
		{Name: instanceReporting, Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/EventHarvest/ReportPeriod", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/EventHarvest/AnalyticEventData/HarvestLimit", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/EventHarvest/CustomEventData/HarvestLimit", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/EventHarvest/ErrorEventData/HarvestLimit", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/EventHarvest/SpanEventData/HarvestLimit", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/EventHarvest/LogEventData/HarvestLimit", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/Logging/Golang", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/Logging/Forwarding/Golang", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/Logging/Metrics/Golang", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/Logging/LocalDecorating/Golang", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/Go/Version/" + Version, Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/Go/Runtime/Version/" + goVersionSimple, Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/Go/gRPC/Version/" + grpcVersion, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Supportability/InfiniteTracing/Span/Seen", Scope: "", Forced: true, Data: []float64{0, 0, 0, 0, 0, 0}},
		{Name: "Supportability/InfiniteTracing/Span/Sent", Scope: "", Forced: true, Data: []float64{0, 0, 0, 0, 0, 0}},
	})
}

func TestEmptyPayloads(t *testing.T) {
	h := newHarvest(time.Now(), testHarvestCfgr)
	payloads := h.Payloads(true)
	if len(payloads) != 9 {
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
	h := newHarvest(now, testHarvestCfgr)
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
	h := newHarvest(now, harvestConfig{
		ReportPeriods: map[harvestTypes]time.Duration{
			fixedHarvestTypes:   fixedHarvestPeriod,
			harvestCustomEvents: time.Second * 5,
		},
		MaxCustomEvents: 3,
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
		{Name: customEventsSeen, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: customEventsSent, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestHarvestLogEventsReady(t *testing.T) {
	now := time.Now()
	fixedHarvestTypes := harvestMetricsTraces & harvestTxnEvents & harvestSpanEvents & harvestLogEvents
	h := newHarvest(now, harvestConfig{
		ReportPeriods: map[harvestTypes]time.Duration{
			fixedHarvestTypes: fixedHarvestPeriod,
			harvestLogEvents:  time.Second * 5,
		},
		LoggingConfig: loggingConfigEnabled(3),
	})

	logEvent := logEvent{
		0.5,
		123456,
		"INFO",
		"User 'xyz' logged in",
		"123456789ADF",
		"ADF09876565",
	}

	h.LogEvents.Add(&logEvent)
	ready := h.Ready(now.Add(10 * time.Second))
	payloads := ready.Payloads(true)
	if len(payloads) == 0 {
		t.Fatal("no payloads generated")
	} else if len(payloads) > 1 {
		t.Fatalf("too many payloads: %d", len(payloads))
	}
	p := payloads[0]
	if m := p.EndpointMethod(); m != "log_event_data" {
		t.Error(m)
	}
	data, err := p.Data("agentRunID", now)
	if nil != err || nil == data {
		t.Error(err, data)
	}
	if h.LogEvents.capacity() != 3 || h.LogEvents.NumSaved() != 0 {
		t.Fatal("log events not correctly reset")
	}

	sampleLogEvent := internal.WantLog{
		Severity:  logEvent.severity,
		Message:   logEvent.message,
		SpanID:    logEvent.spanID,
		TraceID:   logEvent.traceID,
		Timestamp: logEvent.timestamp,
	}

	expectLogEvents(t, ready.LogEvents, []internal.WantLog{sampleLogEvent})
	expectMetrics(t, h.Metrics, []internal.WantMetric{
		{Name: logsSeen, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: logsSeen + "/" + logEvent.severity, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: logsDropped, Scope: "", Forced: true, Data: []float64{0, 0, 0, 0, 0, 0}},
	})
}

func TestHarvestTxnEventsReady(t *testing.T) {
	now := time.Now()
	fixedHarvestTypes := harvestMetricsTraces & harvestCustomEvents & harvestSpanEvents & harvestErrorEvents
	h := newHarvest(now, harvestConfig{
		ReportPeriods: map[harvestTypes]time.Duration{
			fixedHarvestTypes: fixedHarvestPeriod,
			harvestTxnEvents:  time.Second * 5,
		},
		MaxTxnEvents: 3,
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
		{Name: txnEventsSeen, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: txnEventsSent, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestHarvestErrorEventsReady(t *testing.T) {
	now := time.Now()
	fixedHarvestTypes := harvestMetricsTraces & harvestCustomEvents & harvestSpanEvents & harvestTxnEvents
	h := newHarvest(now, harvestConfig{
		ReportPeriods: map[harvestTypes]time.Duration{
			fixedHarvestTypes:  fixedHarvestPeriod,
			harvestErrorEvents: time.Second * 5,
		},
		MaxErrorEvents: 3,
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
		{Name: errorEventsSeen, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: errorEventsSent, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestHarvestSpanEventsReady(t *testing.T) {
	now := time.Now()
	fixedHarvestTypes := harvestMetricsTraces & harvestCustomEvents & harvestTxnEvents & harvestErrorEvents
	h := newHarvest(now, harvestConfig{
		ReportPeriods: map[harvestTypes]time.Duration{
			fixedHarvestTypes: fixedHarvestPeriod,
			harvestSpanEvents: time.Second * 5,
		},
		MaxSpanEvents: 3,
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
		{Name: spanEventsSeen, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: spanEventsSent, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestHarvestMetricsTracesReady(t *testing.T) {
	now := time.Now()
	h := newHarvest(now, harvestConfig{
		ReportPeriods: map[harvestTypes]time.Duration{
			harvestMetricsTraces: fixedHarvestPeriod,
			harvestTypesEvents:   time.Second * 65,
		},
		MaxTxnEvents:    1,
		MaxCustomEvents: 1,
		MaxErrorEvents:  1,
		MaxSpanEvents:   1,
		LoggingConfig:   loggingConfigEnabled(1),
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
		{Name: "zip", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
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

	h := newHarvest(start1, testHarvestCfgr)
	h.Metrics.addCount("zip", 1, forced)
	h.TxnEvents.AddTxnEvent(&txnEvent{
		FinalName: "finalName",
		Start:     time.Now(),
		Duration:  1 * time.Second,
		TotalTime: 2 * time.Second,
	}, 0)

	logEvent := logEvent{
		0.5,
		123456,
		"INFO",
		"User 'xyz' logged in",
		"123456789ADF",
		"ADF09876565",
	}

	h.LogEvents.Add(&logEvent)
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
	if 0 != h.LogEvents.failedHarvests {
		t.Error(h.LogEvents.failedHarvests)
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
		{Name: "zip", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})
	expectCustomEvents(t, h.CustomEvents, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"type":      "myEvent",
			"timestamp": internal.MatchAnything,
		},
		UserAttributes: customEventParams,
	}})
	expectLogEvents(t, h.LogEvents, []internal.WantLog{
		{
			Severity:  logEvent.severity,
			Message:   logEvent.message,
			SpanID:    logEvent.spanID,
			TraceID:   logEvent.traceID,
			Timestamp: logEvent.timestamp,
		},
	})
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

	nextHarvest := newHarvest(start2, testHarvestCfgr)
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
	if 1 != nextHarvest.LogEvents.failedHarvests {
		t.Error(nextHarvest.LogEvents.failedHarvests)
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
		{Name: "zip", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})
	expectCustomEvents(t, nextHarvest.CustomEvents, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"type":      "myEvent",
			"timestamp": internal.MatchAnything,
		},
		UserAttributes: customEventParams,
	}})
	expectLogEvents(t, nextHarvest.LogEvents, []internal.WantLog{
		{
			Severity:  logEvent.severity,
			Message:   logEvent.message,
			SpanID:    logEvent.spanID,
			TraceID:   logEvent.traceID,
			Timestamp: logEvent.timestamp,
		},
	})
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
	args.noticeErrors = true
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
		{Name: webName, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: webRollup, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: dispatcherMetric, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "WebTransactionTotalTime/zip/zap", Scope: "", Forced: false, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "Errors/all", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/" + webName, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: apdexRollup, Scope: "", Forced: true, Data: []float64{0, 1, 0, 2, 2, 0}},
		{Name: "Apdex/zip/zap", Scope: "", Forced: false, Data: []float64{0, 1, 0, 2, 2, 0}},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: []float64{1, 123, 123, 123, 123, 123 * 123}},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: []float64{1, 123, 123, 123, 123, 123 * 123}},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: []float64{1, 0, 0, 0, 0, 0}},
	})

	args.FinalName = webName
	args.IsWeb = true
	args.Errors = nil
	args.noticeErrors = false
	args.Zone = apdexTolerating
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: webName, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: webRollup, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: dispatcherMetric, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "WebTransactionTotalTime/zip/zap", Scope: "", Forced: false, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: apdexRollup, Scope: "", Forced: true, Data: []float64{0, 1, 0, 2, 2, 0}},
		{Name: "Apdex/zip/zap", Scope: "", Forced: false, Data: []float64{0, 1, 0, 2, 2, 0}},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: []float64{1, 123, 123, 123, 123, 123 * 123}},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Scope: "", Forced: false, Data: []float64{1, 123, 123, 123, 123, 123 * 123}},
	})

	args.FinalName = backgroundName
	args.IsWeb = false
	args.Errors = txnErrors
	args.noticeErrors = true
	args.Zone = apdexNone
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: backgroundName, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: backgroundRollup, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "OtherTransactionTotalTime/zip/zap", Scope: "", Forced: false, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "Errors/all", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/" + backgroundName, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: []float64{1, 123, 123, 123, 123, 123 * 123}},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: []float64{1, 123, 123, 123, 123, 123 * 123}},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: []float64{1, 0, 0, 0, 0, 0}},
	})

	// Verify expected errors metrics
	args.FinalName = backgroundName
	args.IsWeb = false
	args.Errors = txnErrors
	args.noticeErrors = false
	args.expectedErrors = true
	args.Zone = apdexNone
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: backgroundName, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: backgroundRollup, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "OtherTransactionTotalTime/zip/zap", Scope: "", Forced: false, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "ErrorsExpected/all", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: []float64{1, 123, 123, 123, 123, 123 * 123}},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: []float64{1, 123, 123, 123, 123, 123 * 123}},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: []float64{1, 0, 0, 0, 0, 0}},
	})

	args.FinalName = backgroundName
	args.IsWeb = false
	args.Errors = nil
	args.noticeErrors = false
	args.expectedErrors = false
	args.Zone = apdexNone
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: backgroundName, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: backgroundRollup, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "OtherTransactionTotalTime/zip/zap", Scope: "", Forced: false, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: []float64{1, 123, 123, 123, 123, 123 * 123}},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: []float64{1, 123, 123, 123, 123, 123 * 123}},
	})

}

func TestHarvestSplitTxnEvents(t *testing.T) {
	now := time.Now()
	h := newHarvest(now, testHarvestCfgr)
	for i := 0; i < internal.MaxTxnEvents; i++ {
		h.TxnEvents.AddTxnEvent(&txnEvent{}, priority(float32(i)))
	}

	payloadsWithSplit := h.Payloads(true)
	payloadsWithoutSplit := h.Payloads(false)

	if len(payloadsWithSplit) != 10 {
		t.Error(len(payloadsWithSplit))
	}
	if len(payloadsWithoutSplit) != 9 {
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
	args.noticeErrors = true
	args.Zone = apdexTolerating
	metrics := newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: webName, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: webRollup, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: dispatcherMetric, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "WebTransactionTotalTime/zip/zap", Scope: "", Forced: false, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "Errors/all", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/" + webName, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: apdexRollup, Scope: "", Forced: true, Data: []float64{0, 1, 0, 2, 2, 0}},
		{Name: "Apdex/zip/zap", Scope: "", Forced: false, Data: []float64{0, 1, 0, 2, 2, 0}},
	})

	args.FinalName = webName
	args.IsWeb = true
	args.Errors = nil
	args.noticeErrors = false
	args.Zone = apdexTolerating
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: webName, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: webRollup, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: dispatcherMetric, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "WebTransactionTotalTime/zip/zap", Scope: "", Forced: false, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: apdexRollup, Scope: "", Forced: true, Data: []float64{0, 1, 0, 2, 2, 0}},
		{Name: "Apdex/zip/zap", Scope: "", Forced: false, Data: []float64{0, 1, 0, 2, 2, 0}},
	})

	args.FinalName = backgroundName
	args.IsWeb = false
	args.Errors = txnErrors
	args.noticeErrors = true
	args.Zone = apdexNone
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: backgroundName, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: backgroundRollup, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "OtherTransactionTotalTime/zip/zap", Scope: "", Forced: false, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "Errors/all", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/" + backgroundName, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})

	args.FinalName = backgroundName
	args.IsWeb = false
	args.Errors = nil
	args.noticeErrors = false
	args.Zone = apdexNone
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: backgroundName, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: backgroundRollup, Scope: "", Forced: true, Data: []float64{1, 123, 0, 123, 123, 123 * 123}},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
		{Name: "OtherTransactionTotalTime/zip/zap", Scope: "", Forced: false, Data: []float64{1, 150, 150, 150, 150, 150 * 150}},
	})
}

func TestNewHarvestSetsDefaultValues(t *testing.T) {
	now := time.Now()
	h := newHarvest(now, testHarvestCfgr)

	if cp := h.TxnEvents.capacity(); cp != internal.MaxTxnEvents {
		t.Error("wrong txn event capacity", cp)
	}
	if cp := h.CustomEvents.capacity(); cp != internal.MaxCustomEvents {
		t.Error("wrong custom event capacity", cp)
	}
	if cp := h.LogEvents.capacity(); cp != internal.MaxLogEvents {
		t.Error("wrong log event capacity", cp)
	}
	if cp := h.ErrorEvents.capacity(); cp != internal.MaxErrorEvents {
		t.Error("wrong error event capacity", cp)
	}
	if cp := h.SpanEvents.capacity(); cp != internal.MaxSpanEvents {
		t.Error("wrong span event capacity", cp)
	}
}

func TestNewHarvestUsesConnectReply(t *testing.T) {
	now := time.Now()
	h := newHarvest(now, harvestConfig{
		ReportPeriods: map[harvestTypes]time.Duration{
			harvestMetricsTraces: fixedHarvestPeriod,
			harvestTypesEvents:   time.Second * 5,
		},
		MaxTxnEvents:    1,
		MaxCustomEvents: 2,
		MaxErrorEvents:  3,
		MaxSpanEvents:   4,
		LoggingConfig:   loggingConfigEnabled(5),
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
	if cp := h.LogEvents.capacity(); cp != 5 {
		t.Error("wrong log event capacity", cp)
	}
}

func TestConfigurableHarvestZeroHarvestLimits(t *testing.T) {
	now := time.Now()

	h := newHarvest(now, harvestConfig{
		ReportPeriods: map[harvestTypes]time.Duration{
			harvestMetricsTraces: fixedHarvestPeriod,
			harvestTypesEvents:   time.Second * 5,
		},
		MaxTxnEvents:    0,
		MaxCustomEvents: 0,
		MaxErrorEvents:  0,
		MaxSpanEvents:   0,
		LoggingConfig:   loggingConfigEnabled(0),
	})
	if cp := h.TxnEvents.capacity(); cp != 0 {
		t.Error("wrong txn event capacity", cp)
	}
	if cp := h.CustomEvents.capacity(); cp != 0 {
		t.Error("wrong custom event capacity", cp)
	}
	if cp := h.LogEvents.capacity(); cp != 0 {
		t.Error("wrong log event capacity", cp)
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
	h.LogEvents.Add(&logEvent{})
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
