// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
)

func TestResponseCodeIsError(t *testing.T) {
	cfg := config{Config: defaultConfig()}
	cfg.ErrorCollector.IgnoreStatusCodes = append(cfg.ErrorCollector.IgnoreStatusCodes, 504)
	run := newAppRun(cfg, internal.ConnectReplyDefaults())

	for _, tc := range []struct {
		Code    int
		IsError bool
	}{
		{Code: 0, IsError: false}, // gRPC
		{Code: 1, IsError: true},  // gRPC
		{Code: 5, IsError: false}, // gRPC
		{Code: 6, IsError: true},  // gRPC
		{Code: 99, IsError: true},
		{Code: 100, IsError: false},
		{Code: 199, IsError: false},
		{Code: 200, IsError: false},
		{Code: 300, IsError: false},
		{Code: 399, IsError: false},
		{Code: 400, IsError: true},
		{Code: 404, IsError: false},
		{Code: 503, IsError: true},
		{Code: 504, IsError: false},
	} {
		if is := run.responseCodeIsError(tc.Code); is != tc.IsError {
			t.Errorf("responseCodeIsError for %d, wanted=%v got=%v",
				tc.Code, tc.IsError, is)
		}
	}
}

func TestResponseCodeIsExpected(t *testing.T) {
	cfg := config{Config: defaultConfig()}
	cfg.ErrorCollector.ExpectStatusCodes = []int{400, 503, 504}
	run := newAppRun(cfg, internal.ConnectReplyDefaults())

	for _, tc := range []struct {
		Code    int
		IsError bool
	}{
		{Code: 0, IsError: false}, // gRPC
		{Code: 1, IsError: false}, // gRPC
		{Code: 400, IsError: true},
		{Code: 404, IsError: false},
		{Code: 503, IsError: true},
		{Code: 504, IsError: true},
	} {
		if is := run.responseCodeIsExpected(tc.Code); is != tc.IsError {
			t.Errorf("responseCodeIsError for %d, wanted=%v got=%v",
				tc.Code, tc.IsError, is)
		}
	}
}

func BenchmarkResponseCodeIsExpectedHit(b *testing.B) {
	cfg := config{Config: defaultConfig()}
	cfg.ErrorCollector.ExpectStatusCodes = []int{400, 503, 504}
	run := newAppRun(cfg, internal.ConnectReplyDefaults())

	b.ResetTimer()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		run.responseCodeIsExpected(400)
	}
}

func TestCrossAppTracingEnabled(t *testing.T) {
	// CAT should NOT be enabled by default.
	cfg := config{Config: defaultConfig()}
	run := newAppRun(cfg, internal.ConnectReplyDefaults())
	if enabled := run.Config.CrossApplicationTracer.Enabled; enabled {
		t.Error(enabled)
	}

	// DT gets priority over CAT.
	cfg = config{Config: defaultConfig()}
	cfg.DistributedTracer.Enabled = true
	cfg.CrossApplicationTracer.Enabled = true
	run = newAppRun(cfg, internal.ConnectReplyDefaults())
	if enabled := run.Config.CrossApplicationTracer.Enabled; enabled {
		t.Error(enabled)
	}

	cfg = config{Config: defaultConfig()}
	cfg.DistributedTracer.Enabled = false
	cfg.CrossApplicationTracer.Enabled = false
	run = newAppRun(cfg, internal.ConnectReplyDefaults())
	if enabled := run.Config.CrossApplicationTracer.Enabled; enabled {
		t.Error(enabled)
	}

	cfg = config{Config: defaultConfig()}
	cfg.DistributedTracer.Enabled = false
	cfg.CrossApplicationTracer.Enabled = true
	run = newAppRun(cfg, internal.ConnectReplyDefaults())
	if enabled := run.Config.CrossApplicationTracer.Enabled; !enabled {
		t.Error(enabled)
	}
}

func TestTxnTraceThreshold(t *testing.T) {
	// Test that the default txn trace threshold is the failing apdex.
	cfg := config{Config: defaultConfig()}
	run := newAppRun(cfg, internal.ConnectReplyDefaults())
	threshold := run.txnTraceThreshold(1 * time.Second)
	if threshold != 4*time.Second {
		t.Error(threshold)
	}

	// Test that the trace threshold can be assigned to a fixed value.
	cfg = config{Config: defaultConfig()}
	cfg.TransactionTracer.Threshold.IsApdexFailing = false
	cfg.TransactionTracer.Threshold.Duration = 3 * time.Second
	run = newAppRun(cfg, internal.ConnectReplyDefaults())
	threshold = run.txnTraceThreshold(1 * time.Second)
	if threshold != 3*time.Second {
		t.Error(threshold)
	}

	// Test that the trace threshold can be overwritten by server-side-config.
	// with "apdex_f".
	cfg = config{Config: defaultConfig()}
	cfg.TransactionTracer.Threshold.IsApdexFailing = false
	cfg.TransactionTracer.Threshold.Duration = 3 * time.Second
	reply := internal.ConnectReplyDefaults()
	json.Unmarshal([]byte(`{"agent_config":{"transaction_tracer.transaction_threshold":"apdex_f"}}`), &reply)
	run = newAppRun(cfg, reply)
	threshold = run.txnTraceThreshold(1 * time.Second)
	if threshold != 4*time.Second {
		t.Error(threshold)
	}

	// Test that the trace threshold can be overwritten by server-side-config.
	// with a numberic value.
	cfg = config{Config: defaultConfig()}
	reply = internal.ConnectReplyDefaults()
	json.Unmarshal([]byte(`{"agent_config":{"transaction_tracer.transaction_threshold":3}}`), &reply)
	run = newAppRun(cfg, reply)
	threshold = run.txnTraceThreshold(1 * time.Second)
	if threshold != 3*time.Second {
		t.Error(threshold)
	}
}

func TestEmptyReplyEventHarvestDefaults(t *testing.T) {
	run := newAppRun(config{Config: defaultConfig()}, &internal.ConnectReply{})
	assertHarvestConfig(t, run.harvestConfig, expectHarvestConfig{
		maxTxnEvents:    internal.MaxTxnEvents,
		maxCustomEvents: internal.MaxCustomEvents,
		maxErrorEvents:  internal.MaxErrorEvents,
		maxSpanEvents:   run.Config.DistributedTracer.ReservoirLimit,
		maxLogEvents:    internal.MaxLogEvents,

		periods: map[harvestTypes]time.Duration{
			harvestTypesAll: 60 * time.Second,
			0:               60 * time.Second,
		},
	})
}

func TestEventHarvestFieldsAllPopulated(t *testing.T) {
	reply, err := internal.UnmarshalConnectReply([]byte(`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 5000,
				"harvest_limits": {
					"analytic_event_data": 1,
					"custom_event_data": 2,
					"log_event_data": 3,
					"error_event_data": 4
				}
			},
			"span_event_harvest_config":{
				"report_period_ms": 10000,
				"harvest_limit": 5
			}
		}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	run := newAppRun(config{Config: defaultConfig()}, reply)
	assertHarvestConfig(t, run.harvestConfig, expectHarvestConfig{
		maxTxnEvents:    1,
		maxCustomEvents: 2,
		maxLogEvents:    3,
		maxErrorEvents:  4,
		maxSpanEvents:   5,
		periods: map[harvestTypes]time.Duration{
			harvestMetricsTraces: 60 * time.Second,
			harvestTypesEvents:   5 * time.Second,
		},
	})
}

func TestZeroReportPeriod(t *testing.T) {
	reply, err := internal.UnmarshalConnectReply([]byte(`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 0
			}
		}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	run := newAppRun(config{Config: defaultConfig()}, reply)
	assertHarvestConfig(t, run.harvestConfig, expectHarvestConfig{
		maxTxnEvents:    internal.MaxTxnEvents,
		maxCustomEvents: internal.MaxCustomEvents,
		maxLogEvents:    internal.MaxLogEvents,
		maxErrorEvents:  internal.MaxErrorEvents,
		maxSpanEvents:   internal.MaxSpanEvents,
		periods: map[harvestTypes]time.Duration{
			harvestTypesAll: 60 * time.Second,
			0:               60 * time.Second,
		},
	})
}

func TestConnectResponseOnlySpanEvents(t *testing.T) {
	reply, err := internal.UnmarshalConnectReply([]byte(`{"return_value":{
			"span_event_harvest_config":{
				"report_period_ms": 10000,
				"harvest_limit": 3
			}}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	run := newAppRun(config{Config: defaultConfig()}, reply)
	assertHarvestConfig(t, run.harvestConfig, expectHarvestConfig{
		maxTxnEvents:    internal.MaxTxnEvents,
		maxCustomEvents: internal.MaxCustomEvents,
		maxLogEvents:    internal.MaxLogEvents,
		maxErrorEvents:  internal.MaxErrorEvents,
		maxSpanEvents:   3,
		periods: map[harvestTypes]time.Duration{
			harvestTypesAll ^ harvestSpanEvents: 60 * time.Second,
			2:                                   60 * time.Second,
		},
	})
}

func TestEventHarvestFieldsOnlyTxnEvents(t *testing.T) {
	reply, err := internal.UnmarshalConnectReply([]byte(`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 5000,
				"harvest_limits": { "analytic_event_data": 3 }
			}}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	run := newAppRun(config{Config: defaultConfig()}, reply)
	assertHarvestConfig(t, run.harvestConfig, expectHarvestConfig{
		maxTxnEvents:    3,
		maxCustomEvents: internal.MaxCustomEvents,
		maxErrorEvents:  internal.MaxErrorEvents,
		maxSpanEvents:   run.Config.DistributedTracer.ReservoirLimit,
		maxLogEvents:    internal.MaxLogEvents,
		periods: map[harvestTypes]time.Duration{
			harvestTypesAll ^ harvestTxnEvents: 60 * time.Second,
			harvestTxnEvents:                   5 * time.Second,
		},
	})
}

func TestEventHarvestFieldsOnlyErrorEvents(t *testing.T) {
	reply, err := internal.UnmarshalConnectReply([]byte(`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 5000,
				"harvest_limits": { "error_event_data": 3 }
			}}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	run := newAppRun(config{Config: defaultConfig()}, reply)
	assertHarvestConfig(t, run.harvestConfig, expectHarvestConfig{
		maxTxnEvents:    internal.MaxTxnEvents,
		maxCustomEvents: internal.MaxCustomEvents,
		maxLogEvents:    internal.MaxLogEvents,
		maxErrorEvents:  3,
		maxSpanEvents:   run.Config.DistributedTracer.ReservoirLimit,
		periods: map[harvestTypes]time.Duration{
			harvestTypesAll ^ harvestErrorEvents: 60 * time.Second,
			harvestErrorEvents:                   5 * time.Second,
		},
	})
}

func TestEventHarvestFieldsOnlyCustomEvents(t *testing.T) {
	reply, err := internal.UnmarshalConnectReply([]byte(`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 5000,
				"harvest_limits": { "custom_event_data": 3 }
			}}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	run := newAppRun(config{Config: defaultConfig()}, reply)
	assertHarvestConfig(t, run.harvestConfig, expectHarvestConfig{
		maxTxnEvents:    internal.MaxTxnEvents,
		maxCustomEvents: 3,
		maxLogEvents:    internal.MaxLogEvents,
		maxErrorEvents:  internal.MaxErrorEvents,
		maxSpanEvents:   run.Config.DistributedTracer.ReservoirLimit,
		periods: map[harvestTypes]time.Duration{
			harvestTypesAll ^ harvestCustomEvents: 60 * time.Second,
			harvestCustomEvents:                   5 * time.Second,
		},
	})
}
func TestConfigurableHarvestNegativeReportPeriod(t *testing.T) {
	h, err := internal.UnmarshalConnectReply([]byte(`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": -1
			}}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	expect := time.Duration(internal.DefaultConfigurableEventHarvestMs) * time.Millisecond
	if period := h.ConfigurablePeriod(); period != expect {
		t.Fatal(expect, period)
	}
}

func TestReplyTraceIDGenerator(t *testing.T) {
	// Test that the default connect reply has a populated trace id
	// generator that works.
	reply := internal.ConnectReplyDefaults()
	id1 := reply.TraceIDGenerator.GenerateTraceID()
	id2 := reply.TraceIDGenerator.GenerateTraceID()
	if len(id1) != 32 || len(id2) != 32 || id1 == id2 {
		t.Error(id1, id2)
	}
	spanID1 := reply.TraceIDGenerator.GenerateSpanID()
	spanID2 := reply.TraceIDGenerator.GenerateSpanID()
	if len(spanID1) != 16 || len(spanID2) != 16 || spanID1 == spanID2 {
		t.Error(spanID1, spanID2)
	}
}

func TestConfigurableTxnEvents_withCollResponse(t *testing.T) {
	h, err := internal.UnmarshalConnectReply([]byte(
		`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 10000,
                "harvest_limits": {
             		"analytic_event_data": 15
                }
			}
        }}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	run := newAppRun(config{Config: defaultConfig()}, h)
	// changed this line because I believe we are not changing the local config based on the response but just the harvest config
	if run.harvestConfig.MaxTxnEvents != 15 {
		t.Errorf("Unexpected max number of txn events, expected %d but got %d", 15, run.harvestConfig.MaxTxnEvents)
	}
}

func TestConfigurableTxnEvents_notInCollResponse(t *testing.T) {
	reply, err := internal.UnmarshalConnectReply([]byte(
		`{"return_value":{
			"event_harvest_config": {
				"report_period_ms": 10000
			}
        }}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	expected := 10
	cfg := config{Config: defaultConfig()}
	cfg.TransactionEvents.MaxSamplesStored = expected
	run := newAppRun(cfg, reply)
	if run.Config.TransactionEvents.MaxSamplesStored != expected {
		t.Errorf("Unexpected max number of txn events, expected %d but got %d", expected, run.Config.TransactionEvents.MaxSamplesStored)
	}
}

type expectHarvestConfig struct {
	maxTxnEvents    int
	maxCustomEvents int
	maxErrorEvents  int
	maxSpanEvents   int
	maxLogEvents    int
	periods         map[harvestTypes]time.Duration
}

func errorExpectNotEqualActual(value string, actual, expect interface{}) error {
	return fmt.Errorf("Expected %s value does not match actual; actual: %+v expect: %+v", value, actual, expect)
}
func assertHarvestConfig(t testing.TB, hc harvestConfig, expect expectHarvestConfig) {
	if h, ok := t.(interface {
		Helper()
	}); ok {
		h.Helper()
	}
	if max := hc.MaxTxnEvents; max != expect.maxTxnEvents {
		t.Error(errorExpectNotEqualActual("maxTxnEvents", max, expect.maxTxnEvents))
	}
	if max := hc.MaxCustomEvents; max != expect.maxCustomEvents {
		t.Error(errorExpectNotEqualActual("MaxCustomEvents", max, expect.maxCustomEvents))
	}
	if max := hc.MaxSpanEvents; max != expect.maxSpanEvents {
		t.Error(errorExpectNotEqualActual("MaxSpanEvents", max, expect.maxSpanEvents))
	}
	if max := hc.MaxErrorEvents; max != expect.maxErrorEvents {
		t.Error(errorExpectNotEqualActual("MaxErrorEvents", max, expect.maxErrorEvents))
	}
	if max := hc.LoggingConfig.maxLogEvents; max != expect.maxLogEvents {
		t.Error(errorExpectNotEqualActual("MaxLogEvents", max, expect.maxErrorEvents))
	}
	if periods := hc.ReportPeriods; !reflect.DeepEqual(periods, expect.periods) {
		t.Error(errorExpectNotEqualActual("ReportPeriods", periods, expect.periods))
	}
}

func TestPlaceholderAppRunSampler(t *testing.T) {
	// Test that the placeholder run used before connect does not sample
	// transactions.
	run := newPlaceholderAppRun(config{Config: defaultConfig()})
	if sampled := run.adaptiveSampler.computeSampled(1.0, time.Now()); sampled {
		t.Fatal(sampled)
	}
}

func TestAppRunSampler(t *testing.T) {
	// Test that a default app run samples transactions.
	// Test that the default txn trace threshold is the failing apdex.
	cfg := config{Config: defaultConfig()}
	run := newAppRun(cfg, internal.ConnectReplyDefaults())
	if sampled := run.adaptiveSampler.computeSampled(1.0, time.Now()); !sampled {
		t.Fatal(sampled)
	}
	if run.adaptiveSampler.target != 10 || run.adaptiveSampler.period != 60*time.Second {
		t.Fatal("invalid sampler initialization",
			run.adaptiveSampler.target, run.adaptiveSampler.period)
	}
}

func TestCreateTransactionName(t *testing.T) {
	reply, err := internal.UnmarshalConnectReply([]byte(`{"return_value":{
		"url_rules":[
			{"match_expression":"zip","each_segment":true,"replacement":"zoop"}
		],
		"transaction_name_rules":[
			{"match_expression":"WebTransaction/Go/zap/zoop/zep",
			 "replacement":"WebTransaction/Go/zap/zoop/zep/zup/zyp"}
		],
		"transaction_segment_terms":[
			{"prefix": "WebTransaction/Go/",
			 "terms": ["zyp", "zoop", "zap"]}
		]
	}}`), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	run := newAppRun(config{Config: defaultConfig()}, reply)

	want := "WebTransaction/Go/zap/zoop/*/zyp"
	if out := run.createTransactionName("/zap/zip/zep", true); out != want {
		t.Error("wanted:", want, "got:", out)
	}
	// Check that the cache was populated as expected.
	if out := run.rulesCache.find("/zap/zip/zep", true); out != want {
		t.Error("wanted:", want, "got:", out)
	}
	// Check that the next call returns the same output.
	if out := run.createTransactionName("/zap/zip/zep", true); out != want {
		t.Error("wanted:", want, "got:", out)
	}
}

func testMockConnectReply(t *testing.T, retVal string) *internal.ConnectReply {
	h, err := internal.UnmarshalConnectReply([]byte(retVal), internal.PreconnectReply{})
	if nil != err {
		t.Fatal(err)
	}
	return h
}

func uintPtr(v uint) *uint {
	return &v
}

func Test_appRun_limit(t *testing.T) {
	tests := []struct {
		name                   string
		configMaxSamplesStored int
		fieldValue             *uint // nil means field() returns nil
		want                   int
	}{
		{
			name:                   "field returns nil, use config value",
			configMaxSamplesStored: 1000,
			fieldValue:             nil,
			want:                   1000,
		},
		{
			name:                   "field returns value, use field value",
			configMaxSamplesStored: 1000,
			fieldValue:             uintPtr(500),
			want:                   500,
		},
		{
			name:                   "field returns zero, use field value",
			configMaxSamplesStored: 1000,
			fieldValue:             uintPtr(0),
			want:                   0,
		},
		{
			name:                   "config is zero, field returns nil",
			configMaxSamplesStored: 0,
			fieldValue:             nil,
			want:                   0,
		},
		{
			name:                   "config is zero, field returns value",
			configMaxSamplesStored: 0,
			fieldValue:             uintPtr(100),
			want:                   100,
		},
		{
			name:                   "config is negative, field returns nil", // keeping this test so we know whatever value exists, we will use
			configMaxSamplesStored: -1,
			fieldValue:             nil,
			want:                   -1,
		},
		{
			name:                   "config is negative, field returns value",
			configMaxSamplesStored: -1,
			fieldValue:             uintPtr(200),
			want:                   200,
		},
		{
			name:                   "field returns large value",
			configMaxSamplesStored: 1000,
			fieldValue:             uintPtr(999999),
			want:                   999999,
		},
		{
			name:                   "field returns 1",
			configMaxSamplesStored: 1000,
			fieldValue:             uintPtr(1),
			want:                   1,
		},
		{
			name:                   "config and field both large values",
			configMaxSamplesStored: 50000,
			fieldValue:             uintPtr(60000),
			want:                   60000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &appRun{}

			// Create a field function that returns the test value
			fieldFunc := func() *uint {
				return tt.fieldValue
			}

			got := run.limit(tt.configMaxSamplesStored, fieldFunc)
			if got != tt.want {
				t.Errorf("limit() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Since we are using uint we are expecting a non-negative number in the harvester response.
// If there were to be a negative number in the test case, it would cause an error when
// unmarshaling the json response.  This test in only testing how the response is handled
// it does not matter what the limits are in this case
func Test_appRun_ptrEventsMethods(t *testing.T) {
	type eventTypeTest struct {
		name       string
		methodName string
		method     func(*appRun) *uint
		jsonKey    string
		configKey  string
	}

	eventTypes := []eventTypeTest{
		{
			name:       "TxnEvents",
			methodName: "ptrTxnEvents",
			method:     (*appRun).ptrTxnEvents,
			jsonKey:    "event_harvest_config",
			configKey:  `{"analytic_event_data": %s}`,
		},
		{
			name:       "CustomEvents",
			methodName: "ptrCustomEvents",
			method:     (*appRun).ptrCustomEvents,
			jsonKey:    "event_harvest_config",
			configKey:  `{"custom_event_data": %s}`,
		},
		{
			name:       "LogEvents",
			methodName: "ptrLogEvents",
			method:     (*appRun).ptrLogEvents,
			jsonKey:    "event_harvest_config",
			configKey:  `{"log_event_data": %s}`,
		},
		{
			name:       "ErrorEvents",
			methodName: "ptrErrorEvents",
			method:     (*appRun).ptrErrorEvents,
			jsonKey:    "event_harvest_config",
			configKey:  `{"error_event_data": %s}`,
		},
		{
			name:       "SpanEvents",
			methodName: "ptrSpanEvents",
			method:     (*appRun).ptrSpanEvents,
			jsonKey:    "span_event_harvest_config",
			configKey:  `%s`,
		},
	}

	testCases := []struct {
		name          string
		format        string
		harvest_limit string
		want          *uint
	}{
		{
			name:          "limit is set to 2000",
			format:        `{"return_value": {"%s": {"%s": %s}}}`,
			harvest_limit: "2000",
			want:          uintPtr(2000),
		},
		{
			name:          "limit is set to 0",
			format:        `{"return_value": {"%s": {"%s": %s}}}`,
			harvest_limit: "0",
			want:          uintPtr(0),
		},
		{
			name:          "limit is set to 1",
			format:        `{"return_value": {"%s": {"%s": %s}}}`,
			harvest_limit: "1",
			want:          uintPtr(1),
		},
		{
			name:          "limit is set to large value",
			format:        `{"return_value": {"%s": {"%s": %s}}}`,
			harvest_limit: "999999",
			want:          uintPtr(999999),
		},
		{
			name:          "config section is null",
			format:        `{"return_value": {"%s": {"%s": null}}}`,
			harvest_limit: "null",
			want:          nil,
		},
		{
			name:          "limit field is null",
			format:        `{"return_value": {"%s": {"%s": %s}}}`,
			harvest_limit: "null",
			want:          nil,
		},
		{
			name:          "config section is missing",
			format:        `{"return_value": {}}`,
			harvest_limit: "null",
			want:          nil,
		},
	}

	for _, eventType := range eventTypes {
		t.Run(eventType.name, func(t *testing.T) {
			harvestLimitField := "harvest_limits"
			if eventType.name == "SpanEvents" {
				harvestLimitField = "harvest_limit"
			}
			for _, tt := range testCases {
				t.Run(tt.name, func(t *testing.T) {
					var jsonStr string

					switch tt.name {
					case "config section is missing":
						jsonStr = tt.format
					case "config section is null":
						jsonStr = fmt.Sprintf(tt.format, harvestLimitField, eventType.jsonKey)
					default:
						harvestLimit := fmt.Sprintf(eventType.configKey, tt.harvest_limit)
						jsonStr = fmt.Sprintf(tt.format, eventType.jsonKey, harvestLimitField, harvestLimit)
					}

					reply := testMockConnectReply(t, jsonStr)
					run := &appRun{Reply: reply}
					got := eventType.method(run)

					if tt.want == nil {
						if got != nil {
							t.Errorf("%s() = %v, want nil", eventType.methodName, got)
						}
					} else {
						if got == nil {
							t.Errorf("%s() = nil, want %v", eventType.methodName, *tt.want)
						} else if *got != *tt.want {
							t.Errorf("%s() = %v, want %v", eventType.methodName, *got, *tt.want)
						}
					}
				})
			}
		})
	}
}
