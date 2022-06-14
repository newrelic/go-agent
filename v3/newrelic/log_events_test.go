// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
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

func sampleLogEvent(priority priority, severity, message string) *logEvent {
	return &logEvent{
		priority:  priority,
		severity:  severity,
		message:   message,
		timestamp: 123456,
	}
}

func TestBasicLogEvents(t *testing.T) {
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(5))
	events.Add(sampleLogEvent(0.5, infoLevel, "message1"))
	events.Add(sampleLogEvent(0.5, infoLevel, "message2"))

	buf := events.DataBuffer()
	err := events.CollectorJSON(buf, agentRunID)
	if nil != err {
		t.Fatal(err)
	}

	json := buf.Bytes()

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

func TestEmptyLogEvents(t *testing.T) {
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(10))
	buf := events.DataBuffer()
	err := events.CollectorJSON(buf, agentRunID)
	if nil != err {
		t.Fatal(err)
	}
	if nil != buf {
		t.Error(string(buf.Bytes()))
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
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(3))

	events.Add(sampleLogEvent(0.999999, infoLevel, "a"))
	events.Add(sampleLogEvent(0.1, infoLevel, "b"))
	events.Add(sampleLogEvent(0.9, infoLevel, "c"))
	events.Add(sampleLogEvent(0.2, infoLevel, "d"))
	events.Add(sampleLogEvent(0.8, infoLevel, "e"))
	events.Add(sampleLogEvent(0.3, infoLevel, "f"))

	buf := events.DataBuffer()
	err := events.CollectorJSON(buf, agentRunID)
	json := buf.Bytes()
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

	buf := e1.DataBuffer()
	err := e1.CollectorJSON(buf, agentRunID)

	if err != nil {
		t.Fatal(err)
	}
	if buf != nil {
		t.Error(string(buf.Bytes()))
	}
	if e1.numSeen != 0 {
		t.Error(e1.numSeen)
	}
	if e1.NumSaved() != 0 {
		t.Error(e1.NumSaved())
	}
}

func TestMergeFullLogEvents(t *testing.T) {
	e1 := newLogEvents(testCommonAttributes, loggingConfigEnabled(2))
	e2 := newLogEvents(testCommonAttributes, loggingConfigEnabled(3))

	e1.Add(sampleLogEvent(0.1, infoLevel, "a"))
	e1.Add(sampleLogEvent(0.15, infoLevel, "b"))
	e1.Add(sampleLogEvent(0.25, infoLevel, "c"))

	e2.Add(sampleLogEvent(0.06, infoLevel, "d"))
	e2.Add(sampleLogEvent(0.12, infoLevel, "e"))
	e2.Add(sampleLogEvent(0.18, infoLevel, "f"))
	e2.Add(sampleLogEvent(0.24, infoLevel, "g"))

	e1.Merge(e2)

	buf := e1.DataBuffer()
	err := e1.CollectorJSON(buf, agentRunID)
	json := buf.Bytes()

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

	e1.Add(sampleLogEvent(0.1, infoLevel, "a"))
	e1.Add(sampleLogEvent(0.15, infoLevel, "b"))
	e1.Add(sampleLogEvent(0.25, infoLevel, "c"))

	e2.Add(sampleLogEvent(0.06, infoLevel, "d"))
	e2.Add(sampleLogEvent(0.12, infoLevel, "e"))
	e2.Add(sampleLogEvent(0.18, infoLevel, "f"))
	e2.Add(sampleLogEvent(0.24, infoLevel, "g"))

	e1.mergeFailed(e2)

	buf := e1.DataBuffer()
	err := e1.CollectorJSON(buf, agentRunID)
	json := buf.Bytes()

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

	e1.Add(sampleLogEvent(0.1, infoLevel, "a"))
	e1.Add(sampleLogEvent(0.15, infoLevel, "b"))
	e1.Add(sampleLogEvent(0.25, infoLevel, "c"))

	e2.Add(sampleLogEvent(0.06, infoLevel, "d"))
	e2.Add(sampleLogEvent(0.12, infoLevel, "e"))
	e2.Add(sampleLogEvent(0.18, infoLevel, "f"))
	e2.Add(sampleLogEvent(0.24, infoLevel, "g"))

	e2.failedHarvests = failedEventsAttemptsLimit

	e1.mergeFailed(e2)

	buf := e1.DataBuffer()
	err := e1.CollectorJSON(buf, agentRunID)
	json := buf.Bytes()

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
		events.Add(sampleLogEvent(priority, "INFO", fmt.Sprint(priority)))
	}
	// Test that the capacity cannot exceed the max.
	if events.capacity() != 10 {
		t.Error(events.capacity())
	}
	e1, e2 := events.split()

	buf := e1.DataBuffer()
	err1 := e1.CollectorJSON(buf, agentRunID)
	j1 := buf.Bytes()

	buf = e2.DataBuffer()
	err2 := e2.CollectorJSON(buf, agentRunID)
	j2 := buf.Bytes()

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
		events.Add(sampleLogEvent(priority, "INFO", fmt.Sprint(priority)))
	}
	e1, e2 := events.split()
	buf := e1.DataBuffer()
	err1 := e1.CollectorJSON(buf, agentRunID)
	j1 := buf.Bytes()

	buf = e2.DataBuffer()
	err2 := e2.CollectorJSON(buf, agentRunID)
	j2 := buf.Bytes()

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
		events.Add(sampleLogEvent(priority, "INFO", fmt.Sprint(priority)))
	}
	e1, e2 := events.split()
	buf := e1.DataBuffer()
	err1 := e1.CollectorJSON(buf, agentRunID)
	j1 := buf.Bytes()

	buf = e2.DataBuffer()
	err2 := e2.CollectorJSON(buf, agentRunID)
	j2 := buf.Bytes()

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
	events.Add(sampleLogEvent(0.5, "INFO", "TEST"))
	if 1 != events.NumSeen() || 0 != events.NumSaved() || 0 != events.capacity() {
		t.Error(events.NumSeen(), events.NumSaved(), events.capacity())
	}
	buf := events.DataBuffer()
	err := events.CollectorJSON(buf, agentRunID)
	if err != nil || buf != nil {
		t.Error(err, string(buf.Bytes()))
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
	events.Add(sampleLogEvent(0.5, "INFO", "TEST"))
	if 1 != events.NumSeen() || 1 != len(events.severityCount) || 0 != events.NumSaved() || 5 != events.capacity() {
		t.Error(events.NumSeen(), len(events.severityCount), events.NumSaved(), events.capacity())
	}

	buf := events.DataBuffer()
	err := events.CollectorJSON(buf, agentRunID)
	if err != nil || buf != nil {
		t.Error(err, string(buf.Bytes()))
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

// Benchmark the cost of harvesting a full log event collector
func BenchmarkLogEventsCollectorJSON(b *testing.B) {
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(internal.MaxLogEvents))

	for i := 0; i < internal.MaxLogEvents; i++ {
		event := &logEvent{
			priority:  newPriority(),
			timestamp: 123456,
			severity:  "INFO",
			message:   "test message",
			spanID:    "Ad300dra7re89",
			traceID:   "2234iIhfLlejrJ0",
		}
		events.Add(event)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := events.DataBuffer()
		err := events.CollectorJSON(buf, agentRunID)
		if nil != err {
			b.Fatal(err)
		}
	}
}
