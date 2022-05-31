// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"fmt"
	"sync"
	"testing"

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

	infoLevel    = "INFO"
	unknownLevel = "UNKNOWN"
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
	if 2 != events.numSeen {
		t.Error(events.numSeen)
	}
	if 2 != events.NumSaved() {
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
		events.Add(sampleLogEvent(priority, "INFO", fmt.Sprint(priority)))
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
		events.Add(sampleLogEvent(priority, "INFO", fmt.Sprint(priority)))
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
	events.Add(sampleLogEvent(0.5, "INFO", "TEST"))
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
	events.Add(sampleLogEvent(0.5, "INFO", "TEST"))
	if 1 != events.NumSeen() || 1 != len(events.severityCount) || 0 != events.NumSaved() || 5 != events.capacity() {
		t.Error(events.NumSeen(), len(events.severityCount), events.NumSaved(), events.capacity())
	}
	js, err := events.CollectorJSON("agentRunID")
	if err != nil || js != nil {
		t.Error(err, string(js))
	}
}

func TestAsyncAddLogEvent(t *testing.T) {
	numThreads := 8
	capacity := numThreads - 1

	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(capacity))
	group := new(sync.WaitGroup)
	group.Add(numThreads)

	// Add a bunch of log events aynchronously
	for n := 0; n < numThreads/2; n++ {
		p := priority(float32(n) / 10.0)
		event := &logEvent{
			priority:  p,
			timestamp: 123456,
			severity:  "INFO",
			message:   fmt.Sprintf("info message %.2f", p),
		}
		go func(event *logEvent) {
			events.Add(event)
			group.Done()
		}(event)
	}

	for n := 0; n < numThreads/2; n++ {
		p := priority(float32(n+numThreads/2) / 10.0)
		event := &logEvent{
			priority:  p,
			timestamp: 123456,
			severity:  "WARN",
			message:   fmt.Sprintf("warn message %.2f", p),
		}
		go func(event *logEvent) {
			events.Add(event)
			group.Done()
		}(event)
	}

	group.Wait()

	expectMap := map[string]int{
		"INFO": numThreads / 2,
		"WARN": numThreads / 2,
	}

	metricErrors := events.assertMetrics(8, capacity, expectMap)
	if metricErrors != nil {
		t.Error(metricErrors)
	}

	// Test Heap Data
	// Assumes that heap implementation is correct when executed synchronously
	expectEvents := newLogEvents(testCommonAttributes, loggingConfigEnabled(capacity))
	for n := 0; n < numThreads/2; n++ {
		p := priority(float32(n) / 10.0)
		event := &logEvent{
			priority:  p,
			timestamp: 123456,
			severity:  "INFO",
			message:   fmt.Sprintf("info message %.2f", p),
		}
		expectEvents.Add(event)
	}

	for n := 0; n < numThreads/2; n++ {
		p := priority(float32(n+numThreads/2) / 10.0)
		event := &logEvent{
			priority:  p,
			timestamp: 123456,
			severity:  "WARN",
			message:   fmt.Sprintf("warn message %.2f", p),
		}
		expectEvents.Add(event)
	}

	heapError := events.assertHeapContains(expectEvents)
	if heapError != nil {
		t.Error(heapError)
	}
}

// verifies that each log events heap contains the same elements
// heaps must be composed of unique messages
func (events *logEvents) assertHeapContains(expect *logEvents) error {
	expectLogs := make(map[string]bool, len(expect.logs))

	for _, event := range expect.logs {
		expectLogs[event.message] = false
	}

	for _, event := range events.logs {
		expectLogs[event.message] = true
	}

	missing := []string{}
	for msg, contains := range expectLogs {
		if !contains {
			missing = append(missing, msg)
		}
	}

	if len(missing) != 0 {
		return fmt.Errorf("expected logs were missing from the event heap: %v", missing)
	}

	return nil
}

func (events *logEvents) assertMetrics(expectSeen, expectSaved int, expectSeverity map[string]int) error {
	err := assertInt(expectSeen, int(events.NumSeen()))
	if err != nil {
		return fmt.Errorf("incorrect number of events seen: %v", err)
	}

	err = assertInt(expectSaved, int(events.NumSaved()))
	if err != nil {
		return fmt.Errorf("incorrect number of events saved: %v", err)
	}

	if len(expectSeverity) != len(events.severityCount) {
		return fmt.Errorf("incorrect number of severities seen: expect %d, actual %d", len(expectSeverity), len(events.severityCount))
	}

	for k, v := range expectSeverity {
		val, ok := events.severityCount[k]
		if !ok {
			return fmt.Errorf("expected severity %s is missing from actual severity count", k)
		}

		err := assertInt(v, val)
		if err != nil {
			return fmt.Errorf("incorrect severity count for %s: expect %d, actual %d", k, v, val)
		}
	}

	return nil
}

func assertInt(expect int, actual int) error {
	if expect != actual {
		return fmt.Errorf("expected %d, actual %d", expect, actual)
	}
	return nil
}

func BenchmarkAddMaximumLogEvent(b *testing.B) {
	eventList := make([]*logEvent, internal.MaxLogEvents)
	for n := 0; n < internal.MaxTxnEvents; n++ {
		eventList[n] = &logEvent{
			priority:  newPriority(),
			timestamp: 123456,
			severity:  "INFO",
			message:   "test message",
			spanID:    "Ad300dra7re89",
			traceID:   "2234iIhfLlejrJ0",
		}
	}
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(internal.MaxLogEvents))

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < internal.MaxTxnEvents; n++ {
		events.Add(eventList[n])
	}
}

func BenchmarkWriteMaximumLogEventJSON(b *testing.B) {
	eventList := make([]*logEvent, internal.MaxLogEvents)
	for n := 0; n < internal.MaxTxnEvents; n++ {
		eventList[n] = &logEvent{
			priority:  newPriority(),
			timestamp: 123456,
			severity:  "INFO",
			message:   "test message",
			spanID:    "Ad300dra7re89",
			traceID:   "2234iIhfLlejrJ0",
		}
	}
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(internal.MaxLogEvents))

	for n := 0; n < internal.MaxTxnEvents; n++ {
		events.Add(eventList[n])
	}

	b.ReportAllocs()
	b.ResetTimer()

	js, err := events.CollectorJSON(agentRunID)
	if nil != err {
		b.Fatal(err, js)
	}
}

func BenchmarkAddAndWriteLogEvent(b *testing.B) {
	b.ReportAllocs()

	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(internal.MaxLogEvents))
	event := &logEvent{
		priority:  newPriority(),
		timestamp: 123456,
		severity:  "INFO",
		message:   "test message",
		spanID:    "Ad300dra7re89",
		traceID:   "2234iIhfLlejrJ0",
	}

	events.Add(event)
	js, err := events.CollectorJSON(agentRunID)
	if nil != err {
		b.Fatal(err, js)
	}
}

func BenchmarkAddAndWriteMaximumLogEvents(b *testing.B) {

	eventList := make([]*logEvent, internal.MaxLogEvents)
	events := newLogEvents(testCommonAttributes, loggingConfigEnabled(internal.MaxLogEvents))
	for n := 0; n < internal.MaxTxnEvents; n++ {
		eventList[n] = &logEvent{
			priority:  newPriority(),
			timestamp: 123456,
			severity:  "INFO",
			message:   "test message",
			spanID:    "Ad300dra7re89",
			traceID:   "2234iIhfLlejrJ0",
		}
	}

	b.ResetTimer()

	for n := 0; n < internal.MaxTxnEvents; n++ {
		events.Add(eventList[n])
	}

	js, err := events.CollectorJSON(agentRunID)
	if nil != err {
		b.Fatal(err, js)
	}
}
