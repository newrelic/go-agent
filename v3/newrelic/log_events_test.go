// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
)

var (
	testGUID             = "testGUID"
	testEntityName       = "testEntityName"
	testHostname         = "testHostname"
	testCommonAttributes = commonAttributes{
		entityGUID: testGUID,
		entityName: testEntityName,
		hostname:   testHostname,
	}

	commonJSON = `[{"common":{"attributes":{"entity.guid":"testGUID","entity.name":"testEntityName","hostname":"testHostname"}},"logs":[`

	infoLevel = "INFO"
)

func loggingConfigEnabled(limit int) loggingConfig {
	return loggingConfig{
		loggingEnabled:  true,
		localEnrichment: true,
		collectEvents:   true,
		collectMetrics:  true,
		maxLogEvents:    limit,
	}
}

func sampleLogEvent(priority priority, severity, message string, attributes map[string]any) *logEvent {
	return &logEvent{
		priority:   priority,
		severity:   severity,
		message:    message,
		attributes: attributes,
		timestamp:  123456,
	}
}

func TestBasicLogEvents(t *testing.T) {
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(5))
	events.Add(sampleLogEvent(0.5, infoLevel, "message1", nil))
	events.Add(sampleLogEvent(0.5, infoLevel, "message2", nil))

	json, err := events.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}

	expected := commonJSON +
		`{"level":"INFO","message":"message1","timestamp":123456},` +
		`{"level":"INFO","message":"message2","timestamp":123456}]}]`

	if string(json) != expected {
		t.Error(string(json), expected)
	}
	if events.numSeen != 2 {
		t.Error(events.numSeen)
	}
	if events.NumSaved() != 2 {
		t.Error(events.NumSaved())
	}
}

type testStruct struct {
	A string
	B int
	C c
}

type c struct {
	D string
}

func TestBasicLogEventWithAttributes(t *testing.T) {
	st := testStruct{
		A: "a",
		B: 1,
		C: c{"hello"},
	}

	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(6))
	events.Add(sampleLogEvent(0.5, infoLevel, "message1", map[string]any{"two": "hi"}))
	events.Add(sampleLogEvent(0.5, infoLevel, "message2", map[string]any{"struct": st}))
	events.Add(sampleLogEvent(0.5, infoLevel, "message3", map[string]any{"map": map[string]string{"hi": "hello"}}))
	events.Add(sampleLogEvent(0.5, infoLevel, "message4", map[string]any{"slice": []string{"hi", "hello", "test"}}))
	events.Add(sampleLogEvent(0.5, infoLevel, "message5", map[string]any{"array": [2]int{1, 2}}))
	events.Add(sampleLogEvent(0.5, infoLevel, "message6", map[string]any{"error": errors.New("test error")}))

	json, err := events.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}

	expected := commonJSON +
		`{"level":"INFO","message":"message1","timestamp":123456,"attributes":{"two":"hi"}},` +
		`{"level":"INFO","message":"message2","timestamp":123456,"attributes":{"struct":"{\"A\":\"a\",\"B\":1,\"C\":{\"D\":\"hello\"}}"}},` +
		`{"level":"INFO","message":"message3","timestamp":123456,"attributes":{"map":"{\"hi\":\"hello\"}"}},` +
		`{"level":"INFO","message":"message4","timestamp":123456,"attributes":{"slice":"[\"hi\",\"hello\",\"test\"]"}},` +
		`{"level":"INFO","message":"message5","timestamp":123456,"attributes":{"array":"[1,2]"}},` +
		`{"level":"INFO","message":"message6","timestamp":123456,"attributes":{"error":"test error"}}]}]`

	if string(json) != expected {
		t.Error("actual not equal to expected:\n", string(json), "\n", expected)
	}
	if events.numSeen != 6 {
		t.Error(events.numSeen)
	}
	if events.NumSaved() != 6 {
		t.Error(events.NumSaved())
	}
}

func TestEmptyLogEvents(t *testing.T) {
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(10))
	json, err := events.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}
	if nil != json {
		t.Error(string(json))
	}
	if events.numSeen != 0 {
		t.Error(events.numSeen)
	}
	if events.NumSaved() != 0 {
		t.Error(events.NumSaved())
	}
}

// The events with the highest priority should make it: a, c, e
func TestSamplingLogEvents(t *testing.T) {
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(3))

	events.Add(sampleLogEvent(0.999999, infoLevel, "a", nil))
	events.Add(sampleLogEvent(0.1, infoLevel, "b", nil))
	events.Add(sampleLogEvent(0.9, infoLevel, "c", nil))
	events.Add(sampleLogEvent(0.2, infoLevel, "d", nil))
	events.Add(sampleLogEvent(0.8, infoLevel, "e", nil))
	events.Add(sampleLogEvent(0.3, infoLevel, "f", nil))

	json, err := events.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}
	expect := commonJSON +
		`{"level":"INFO","message":"e","timestamp":123456},` +
		`{"level":"INFO","message":"a","timestamp":123456},` +
		`{"level":"INFO","message":"c","timestamp":123456}]}` +
		`]`
	if string(json) != expect {
		t.Error(string(json), expect)
	}
	if 6 != events.numSeen {
		t.Error(events.numSeen)
	}
	if 3 != events.NumSaved() {
		t.Error(events.NumSaved())
	}
}

func TestMergeEmptyLogEvents(t *testing.T) {
	e1 := newLogEvents(testCommonAttributes, loggingConfigEnabled(10))
	e2 := newLogEvents(testCommonAttributes, loggingConfigEnabled(10))
	e1.Merge(e2)
	json, err := e1.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}
	if nil != json {
		t.Error(string(json))
	}
	if 0 != e1.numSeen {
		t.Error(e1.numSeen)
	}
	if 0 != e1.NumSaved() {
		t.Error(e1.NumSaved())
	}
}

func TestMergeFullLogEvents(t *testing.T) {
	e1 := newLogEvents(testCommonAttributes, loggingConfigEnabled(2))
	e2 := newLogEvents(testCommonAttributes, loggingConfigEnabled(3))

	e1.Add(sampleLogEvent(0.1, infoLevel, "a", nil))
	e1.Add(sampleLogEvent(0.15, infoLevel, "b", nil))
	e1.Add(sampleLogEvent(0.25, infoLevel, "c", nil))

	e2.Add(sampleLogEvent(0.06, infoLevel, "d", nil))
	e2.Add(sampleLogEvent(0.12, infoLevel, "e", nil))
	e2.Add(sampleLogEvent(0.18, infoLevel, "f", nil))
	e2.Add(sampleLogEvent(0.24, infoLevel, "g", nil))

	e1.Merge(e2)
	json, err := e1.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}

	// expect the highest priority events: c, g
	expect := commonJSON +
		`{"level":"INFO","message":"g","timestamp":123456},` +
		`{"level":"INFO","message":"c","timestamp":123456}]}]`

	if string(json) != expect {
		t.Error(string(json))
	}
	if e1.numSeen != 7 {
		t.Error(e1.numSeen)
	}
	if e1.NumSaved() != 2 {
		t.Error(e1.NumSaved())
	}
}

func TestLogEventMergeFailedSuccess(t *testing.T) {
	e1 := newLogEvents(testCommonAttributes, loggingConfigEnabled(2))
	e2 := newLogEvents(testCommonAttributes, loggingConfigEnabled(3))

	e1.Add(sampleLogEvent(0.1, infoLevel, "a", nil))
	e1.Add(sampleLogEvent(0.15, infoLevel, "b", nil))
	e1.Add(sampleLogEvent(0.25, infoLevel, "c", nil))

	e2.Add(sampleLogEvent(0.06, infoLevel, "d", nil))
	e2.Add(sampleLogEvent(0.12, infoLevel, "e", nil))
	e2.Add(sampleLogEvent(0.18, infoLevel, "f", nil))
	e2.Add(sampleLogEvent(0.24, infoLevel, "g", nil))

	e1.mergeFailed(e2)

	json, err := e1.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}
	// expect the highest priority events: c, g
	expect := commonJSON +
		`{"level":"INFO","message":"g","timestamp":123456},` +
		`{"level":"INFO","message":"c","timestamp":123456}]}]`

	if string(json) != expect {
		t.Error(string(json))
	}
	if 7 != e1.numSeen {
		t.Error(e1.numSeen)
	}
	if 2 != e1.NumSaved() {
		t.Error(e1.NumSaved())
	}
	if 1 != e1.failedHarvests {
		t.Error(e1.failedHarvests)
	}
}

func TestLogEventMergeFailedLimitReached(t *testing.T) {
	e1 := newLogEvents(testCommonAttributes, loggingConfigEnabled(2))
	e2 := newLogEvents(testCommonAttributes, loggingConfigEnabled(3))

	e1.Add(sampleLogEvent(0.1, infoLevel, "a", nil))
	e1.Add(sampleLogEvent(0.15, infoLevel, "b", nil))
	e1.Add(sampleLogEvent(0.25, infoLevel, "c", nil))

	e2.Add(sampleLogEvent(0.06, infoLevel, "d", nil))
	e2.Add(sampleLogEvent(0.12, infoLevel, "e", nil))
	e2.Add(sampleLogEvent(0.18, infoLevel, "f", nil))
	e2.Add(sampleLogEvent(0.24, infoLevel, "g", nil))

	e2.failedHarvests = failedEventsAttemptsLimit

	e1.mergeFailed(e2)

	json, err := e1.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}
	expect := commonJSON +
		`{"level":"INFO","message":"b","timestamp":123456},` +
		`{"level":"INFO","message":"c","timestamp":123456}]}]`

	if string(json) != expect {
		t.Error(string(json))
	}
	if 3 != e1.numSeen {
		t.Error(e1.numSeen)
	}
	if 2 != e1.NumSaved() {
		t.Error(e1.NumSaved())
	}
	if 0 != e1.failedHarvests {
		t.Error(e1.failedHarvests)
	}
}

func TestLogEventsSplitFull(t *testing.T) {
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(10))
	for i := 0; i < 15; i++ {
		priority := priority(float32(i) / 10.0)
		events.Add(sampleLogEvent(priority, "INFO", fmt.Sprint(priority), nil))
	}
	// Test that the capacity cannot exceed the max.
	if 10 != events.capacity() {
		t.Error(events.capacity())
	}
	e1, e2 := events.split()
	j1, err1 := e1.CollectorJSON(agentRunID)
	j2, err2 := e2.CollectorJSON(agentRunID)
	if err1 != nil || err2 != nil {
		t.Fatal(err1, err2)
	}
	expect1 := commonJSON +
		`{"level":"INFO","message":"0.5","timestamp":123456},` +
		`{"level":"INFO","message":"0.7","timestamp":123456},` +
		`{"level":"INFO","message":"0.6","timestamp":123456},` +
		`{"level":"INFO","message":"0.8","timestamp":123456},` +
		`{"level":"INFO","message":"0.9","timestamp":123456}]}]`
	if string(j1) != expect1 {
		t.Error(string(j1))
	}

	expect2 := commonJSON +
		`{"level":"INFO","message":"1.1","timestamp":123456},` +
		`{"level":"INFO","message":"1.4","timestamp":123456},` +
		`{"level":"INFO","message":"1","timestamp":123456},` +
		`{"level":"INFO","message":"1.3","timestamp":123456},` +
		`{"level":"INFO","message":"1.2","timestamp":123456}]}]`
	if string(j2) != expect2 {
		t.Error(string(j2))
	}
}

// TODO: When miniumu supported go version is 1.18, make an event heap in GO generics and remove all this duplicate code
// interfaces are too slow :(
func TestLogEventsSplitNotFullOdd(t *testing.T) {
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(10))
	for i := 0; i < 7; i++ {
		priority := priority(float32(i) / 10.0)
		events.Add(sampleLogEvent(priority, "INFO", fmt.Sprint(priority), nil))
	}
	e1, e2 := events.split()
	j1, err1 := e1.CollectorJSON(agentRunID)
	j2, err2 := e2.CollectorJSON(agentRunID)
	if err1 != nil || err2 != nil {
		t.Fatal(err1, err2)
	}
	expect1 := commonJSON +
		`{"level":"INFO","message":"0","timestamp":123456},` +
		`{"level":"INFO","message":"0.1","timestamp":123456},` +
		`{"level":"INFO","message":"0.2","timestamp":123456}]}]`
	if string(j1) != expect1 {
		t.Error(string(j1))
	}

	expect2 := commonJSON +
		`{"level":"INFO","message":"0.3","timestamp":123456},` +
		`{"level":"INFO","message":"0.4","timestamp":123456},` +
		`{"level":"INFO","message":"0.5","timestamp":123456},` +
		`{"level":"INFO","message":"0.6","timestamp":123456}]}]`
	if string(j2) != expect2 {
		t.Error(string(j2))
	}
}

func TestLogEventsSplitNotFullEven(t *testing.T) {
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(10))
	for i := 0; i < 8; i++ {
		priority := priority(float32(i) / 10.0)
		events.Add(sampleLogEvent(priority, "INFO", fmt.Sprint(priority), nil))
	}
	e1, e2 := events.split()
	j1, err1 := e1.CollectorJSON(agentRunID)
	j2, err2 := e2.CollectorJSON(agentRunID)
	if err1 != nil || err2 != nil {
		t.Fatal(err1, err2)
	}
	expect1 := commonJSON +
		`{"level":"INFO","message":"0","timestamp":123456},` +
		`{"level":"INFO","message":"0.1","timestamp":123456},` +
		`{"level":"INFO","message":"0.2","timestamp":123456},` +
		`{"level":"INFO","message":"0.3","timestamp":123456}]}]`
	if string(j1) != expect1 {
		t.Error(string(j1))
	}

	expect2 := commonJSON +
		`{"level":"INFO","message":"0.4","timestamp":123456},` +
		`{"level":"INFO","message":"0.5","timestamp":123456},` +
		`{"level":"INFO","message":"0.6","timestamp":123456},` +
		`{"level":"INFO","message":"0.7","timestamp":123456}]}]`
	if string(j2) != expect2 {
		t.Error(string(j2))
	}
}

func TestLogEventsZeroCapacity(t *testing.T) {
	// Analytics events methods should be safe when configurable harvest
	// settings have an event limit of zero.
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(0))
	if 0 != events.NumSeen() || 0 != events.NumSaved() || 0 != events.capacity() {
		t.Error(events.NumSeen(), events.NumSaved(), events.capacity())
	}
	events.Add(sampleLogEvent(0.5, "INFO", "TEST", nil))
	if 1 != events.NumSeen() || 0 != events.NumSaved() || 0 != events.capacity() {
		t.Error(events.NumSeen(), events.NumSaved(), events.capacity())
	}
	js, err := events.CollectorJSON("agentRunID")
	if err != nil || js != nil {
		t.Error(err, string(js))
	}
}

func TestLogEventCollectionDisabled(t *testing.T) {
	// Analytics events methods should be safe when configurable harvest
	// settings have an event limit of zero.
	config := loggingConfigEnabled(5)
	config.collectEvents = false
	events := newLogEvents(testCommonAttributes, config)
	if 0 != events.NumSeen() || 0 != len(events.severityCount) || 0 != events.NumSaved() || 5 != events.capacity() {
		t.Error(events.NumSeen(), len(events.severityCount), events.NumSaved(), events.capacity())
	}
	events.Add(sampleLogEvent(0.5, "INFO", "TEST", nil))
	if 1 != events.NumSeen() || 1 != len(events.severityCount) || 0 != events.NumSaved() || 5 != events.capacity() {
		t.Error(events.NumSeen(), len(events.severityCount), events.NumSaved(), events.capacity())
	}
	js, err := events.CollectorJSON("agentRunID")
	if err != nil || js != nil {
		t.Error(err, string(js))
	}
}

func BenchmarkLogEventsAdd(b *testing.B) {
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(internal.MaxLogEvents))
	event := &logEvent{
		priority:  newPriority(),
		timestamp: 123456,
		severity:  "INFO",
		message:   "test message",
		spanID:    "Ad300dra7re89",
		traceID:   "2234iIhfLlejrJ0",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		events.Add(event)
	}
}

func BenchmarkLogEventsCollectorJSON(b *testing.B) {
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(internal.MaxLogEvents))
	for i := 0; i < internal.MaxLogEvents; i++ {
		event := &logEvent{
			priority:  newPriority(),
			timestamp: 123456,
			severity:  "INFO",
			message:   "This is a log message that represents an estimate for how long the average log message is. The average log payload is 700 bytese.",
			spanID:    "Ad300dra7re89",
			traceID:   "2234iIhfLlejrJ0",
		}

		events.Add(event)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		js, err := events.CollectorJSON(agentRunID)
		if nil != err {
			b.Fatal(err, js)
		}
	}
}

func BenchmarkLogEventCollectorJSON_OneEvent(b *testing.B) {
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(internal.MaxLogEvents))
	event := &logEvent{
		priority:  newPriority(),
		timestamp: 123456,
		severity:  "INFO",
		message:   "This is a log message that represents an estimate for how long the average log message is. The average log payload is 700 bytes.",
		spanID:    "Ad300dra7re89",
		traceID:   "2234iIhfLlejrJ0",
	}

	events.Add(event)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		js, err := events.CollectorJSON(agentRunID)
		if nil != err {
			b.Fatal(err, js)
		}
	}
}

func BenchmarkRecordLoggingMetrics(b *testing.B) {
	now := time.Now()
	fixedHarvestTypes := harvestMetricsTraces & harvestTxnEvents & harvestSpanEvents & harvestLogEvents
	h := newHarvest(now, harvestConfig{
		ReportPeriods: map[harvestTypes]time.Duration{
			fixedHarvestTypes: fixedHarvestPeriod,
			harvestLogEvents:  time.Second * 5,
		},
		LoggingConfig: loggingConfigEnabled(3),
	})

	for i := 0; i < internal.MaxLogEvents; i++ {
		logEvent := logEvent{
			nil,
			newPriority(),
			123456,
			"INFO",
			fmt.Sprintf("User 'xyz' logged in %d", i),
			"123456789ADF",
			"ADF09876565",
		}

		h.LogEvents.Add(&logEvent)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.ReportAllocs()
		h.LogEvents.RecordLoggingMetrics(h.Metrics)
	}
}
