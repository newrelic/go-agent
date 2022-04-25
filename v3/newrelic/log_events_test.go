// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"testing"
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

	infoLevel    = "INFO"
	unknownLevel = "UNKNOWN"
)

func sampleLogEvent(priority priority, severity, message string) *logEvent {
	return &logEvent{
		priority,
		severity,
		message,
		"AF02332",
		"0024483",
		123456,
	}
}

// NOTE: this is going to make the tests run really slow due to heap allocation
func sampleLogEventNoParent(priority priority, severity, message string) *logEvent {
	return &logEvent{
		priority,
		severity,
		message,
		"",
		"",
		123456,
	}
}

func TestBasicLogEvents(t *testing.T) {
	events := newLogEvents(testCommonAttributes, 5)
	events.Add(sampleLogEvent(0.5, infoLevel, "message1"))
	events.Add(sampleLogEventNoParent(0.1, infoLevel, "message2"))

	json, err := events.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}

	expected := commonJSON +
		`{"severity":"INFO","message":"message1","span.id":"AF02332","trace.id":"0024483","timestamp":123456},` +
		`{"severity":"INFO","message":"message2","timestamp":123456}]}` +
		`]`

	if string(json) != expected {
		t.Error(string(json), expected)
	}
	if 2 != events.numSeen {
		t.Error(events.numSeen)
	}
	if 2 != events.NumSaved() {
		t.Error(events.NumSaved())
	}
}

func TestEmptyLogEvents(t *testing.T) {
	events := newLogEvents(testCommonAttributes, 10)
	json, err := events.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}
	if nil != json {
		t.Error(string(json))
	}
	if 0 != events.numSeen {
		t.Error(events.numSeen)
	}
	if 0 != events.NumSaved() {
		t.Error(events.NumSaved())
	}
}

// The events with the highest priority should make it: a, c, e
func TestSamplingLogEvents(t *testing.T) {
	events := newLogEvents(testCommonAttributes, 3)

	events.Add(sampleLogEvent(0.999999, infoLevel, "a"))
	events.Add(sampleLogEvent(0.1, infoLevel, "b"))
	events.Add(sampleLogEvent(0.9, infoLevel, "c"))
	events.Add(sampleLogEvent(0.2, infoLevel, "d"))
	events.Add(sampleLogEvent(0.8, infoLevel, "e"))
	events.Add(sampleLogEvent(0.3, infoLevel, "f"))

	json, err := events.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}
	expect := commonJSON +
		`{"severity":"INFO","message":"e","span.id":"AF02332","trace.id":"0024483","timestamp":123456},` +
		`{"severity":"INFO","message":"a","span.id":"AF02332","trace.id":"0024483","timestamp":123456},` +
		`{"severity":"INFO","message":"c","span.id":"AF02332","trace.id":"0024483","timestamp":123456}]}` +
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
	e1 := newLogEvents(testCommonAttributes, 10)
	e2 := newLogEvents(testCommonAttributes, 10)
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
	e1 := newLogEvents(testCommonAttributes, 2)
	e2 := newLogEvents(testCommonAttributes, 3)

	e1.Add(sampleLogEvent(0.1, infoLevel, "a"))
	e1.Add(sampleLogEvent(0.15, infoLevel, "b"))
	e1.Add(sampleLogEvent(0.25, infoLevel, "c"))
	e2.Add(sampleLogEvent(0.06, infoLevel, "d"))
	e2.Add(sampleLogEvent(0.12, infoLevel, "e"))
	e2.Add(sampleLogEvent(0.18, infoLevel, "f"))
	e2.Add(sampleLogEvent(0.24, infoLevel, "g"))

	e1.Merge(e2)
	json, err := e1.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}

	// expect the highest priority events: c, g
	expect := commonJSON +
		`{"severity":"INFO","message":"g","span.id":"AF02332","trace.id":"0024483","timestamp":123456},` +
		`{"severity":"INFO","message":"c","span.id":"AF02332","trace.id":"0024483","timestamp":123456}]}]`

	if string(json) != expect {
		t.Error(string(json))
	}
	if 7 != e1.numSeen {
		t.Error(e1.numSeen)
	}
	if 2 != e1.NumSaved() {
		t.Error(e1.NumSaved())
	}
}

func TestLogEventMergeFailedSuccess(t *testing.T) {
	e1 := newLogEvents(testCommonAttributes, 2)
	e2 := newLogEvents(testCommonAttributes, 3)

	e1.Add(sampleLogEvent(0.1, infoLevel, "a"))
	e1.Add(sampleLogEvent(0.15, infoLevel, "b"))
	e1.Add(sampleLogEvent(0.25, infoLevel, "c"))

	e2.Add(sampleLogEvent(0.06, infoLevel, "d"))
	e2.Add(sampleLogEvent(0.12, infoLevel, "e"))
	e2.Add(sampleLogEvent(0.18, infoLevel, "f"))
	e2.Add(sampleLogEvent(0.24, infoLevel, "g"))

	e1.mergeFailed(e2)

	json, err := e1.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}
	// expect the highest priority events: c, g
	expect := commonJSON +
		`{"severity":"INFO","message":"g","span.id":"AF02332","trace.id":"0024483","timestamp":123456},` +
		`{"severity":"INFO","message":"c","span.id":"AF02332","trace.id":"0024483","timestamp":123456}]}]`

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
	e1 := newLogEvents(testCommonAttributes, 2)
	e2 := newLogEvents(testCommonAttributes, 3)

	e1.Add(sampleLogEvent(0.1, infoLevel, "a"))
	e1.Add(sampleLogEvent(0.15, infoLevel, "b"))
	e1.Add(sampleLogEvent(0.25, infoLevel, "c"))

	e2.Add(sampleLogEvent(0.06, infoLevel, "d"))
	e2.Add(sampleLogEvent(0.12, infoLevel, "e"))
	e2.Add(sampleLogEvent(0.18, infoLevel, "f"))
	e2.Add(sampleLogEvent(0.24, infoLevel, "g"))

	e2.failedHarvests = failedEventsAttemptsLimit

	e1.mergeFailed(e2)

	json, err := e1.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}
	expect := commonJSON +
		`{"severity":"INFO","message":"b","span.id":"AF02332","trace.id":"0024483","timestamp":123456},` +
		`{"severity":"INFO","message":"c","span.id":"AF02332","trace.id":"0024483","timestamp":123456}]}]`

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

/*
func logEventBenchmarkHelper(b *testing.B, w jsonWriter) {
	events := newLogEvents(testCommonAttributes, internal.MaxTxnEvents)
	event := logEvent{0, w}
	for n := 0; n < internal.MaxTxnEvents; n++ {
		events.addEvent(event)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		js, err := events.CollectorJSON(agentRunID)
		if nil != err {
			b.Fatal(err, js)
		}
	}
}


func BenchmarkTxnEventsCollectorJSON(b *testing.B) {
	event := &txnEvent{
		FinalName: "WebTransaction/Go/zip/zap",
		Start:     time.Now(),
		Duration:  2 * time.Second,
		Queuing:   1 * time.Second,
		Zone:      apdexSatisfying,
		Attrs:     nil,
	}
	analyticsEventBenchmarkHelper(b, event)
}

func BenchmarkCustomEventsCollectorJSON(b *testing.B) {
	now := time.Now()
	ce, err := createCustomEvent("myEventType", map[string]interface{}{
		"string": "myString",
		"bool":   true,
		"int64":  int64(123),
	}, now)
	if nil != err {
		b.Fatal(err)
	}
	analyticsEventBenchmarkHelper(b, ce)
}

func BenchmarkErrorEventsCollectorJSON(b *testing.B) {
	e := txnErrorFromResponseCode(time.Now(), 503)
	e.Stack = getStackTrace()

	txnName := "WebTransaction/Go/zip/zap"
	event := &errorEvent{
		errorData: e,
		txnEvent: txnEvent{
			FinalName: txnName,
			Duration:  3 * time.Second,
			Attrs:     nil,
		},
	}
	analyticsEventBenchmarkHelper(b, event)
}


func TestSplitFull(t *testing.T) {
	events := newLogEvents(testCommonAttributes, 10)
	for i := 0; i < 15; i++ {
		events.addEvent(sampleLogEvent(priority(float32(i) / 10.0)))
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
	if string(j1) != `["12345",{"reservoir_size":5,"events_seen":5},[0.5,0.7,0.6,0.8,0.9]]` {
		t.Error(string(j1))
	}
	if string(j2) != `["12345",{"reservoir_size":5,"events_seen":10},[1.1,1.4,1,1.3,1.2]]` {
		t.Error(string(j2))
	}
}

func TestSplitNotFullOdd(t *testing.T) {
	events := newLogEvents(testCommonAttributes, 10)
	for i := 0; i < 7; i++ {
		events.addEvent(sampleLogEvent(priority(float32(i) / 10.0)))
	}
	e1, e2 := events.split()
	j1, err1 := e1.CollectorJSON(agentRunID)
	j2, err2 := e2.CollectorJSON(agentRunID)
	if err1 != nil || err2 != nil {
		t.Fatal(err1, err2)
	}
	if string(j1) != `["12345",{"reservoir_size":3,"events_seen":3},[0,0.1,0.2]]` {
		t.Error(string(j1))
	}
	if string(j2) != `["12345",{"reservoir_size":4,"events_seen":4},[0.3,0.4,0.5,0.6]]` {
		t.Error(string(j2))
	}
}

func TestSplitNotFullEven(t *testing.T) {
	events := newLogEvents(testCommonAttributes, 10)
	for i := 0; i < 8; i++ {
		events.addEvent(sampleLogEvent(priority(float32(i) / 10.0)))
	}
	e1, e2 := events.split()
	j1, err1 := e1.CollectorJSON(agentRunID)
	j2, err2 := e2.CollectorJSON(agentRunID)
	if err1 != nil || err2 != nil {
		t.Fatal(err1, err2)
	}
	if string(j1) != `["12345",{"reservoir_size":4,"events_seen":4},[0,0.1,0.2,0.3]]` {
		t.Error(string(j1))
	}
	if string(j2) != `["12345",{"reservoir_size":4,"events_seen":4},[0.4,0.5,0.6,0.7]]` {
		t.Error(string(j2))
	}
}

func TestLogEventsZeroCapacity(t *testing.T) {
	// Analytics events methods should be safe when configurable harvest
	// settings have an event limit of zero.
	events := newLogEvents(testCommonAttributes, 0)
	if 0 != events.NumSeen() || 0 != events.NumSaved() || 0 != events.capacity() {
		t.Error(events.NumSeen(), events.NumSaved(), events.capacity())
	}
	events.addEvent(sampleLogEvent(0.5))
	if 1 != events.NumSeen() || 0 != events.NumSaved() || 0 != events.capacity() {
		t.Error(events.NumSeen(), events.NumSaved(), events.capacity())
	}
	js, err := events.CollectorJSON("agentRunID")
	if err != nil || js != nil {
		t.Error(err, string(js))
	}
}
*/
