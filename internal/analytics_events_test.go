// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"strconv"
	"testing"
	"time"
)

var (
	agentRunID = `12345`
)

type priorityWriter Priority

func (x priorityWriter) WriteJSON(buf *bytes.Buffer) {
	buf.WriteString(strconv.FormatFloat(float64(x), 'f', -1, 32))
}

func sampleAnalyticsEvent(priority Priority) analyticsEvent {
	return analyticsEvent{
		priority,
		priorityWriter(priority),
	}
}

func TestBasic(t *testing.T) {
	events := newAnalyticsEvents(10)
	events.addEvent(sampleAnalyticsEvent(0.5))
	events.addEvent(sampleAnalyticsEvent(0.5))
	events.addEvent(sampleAnalyticsEvent(0.5))

	json, err := events.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}

	expected := `["12345",{"reservoir_size":10,"events_seen":3},[0.5,0.5,0.5]]`

	if string(json) != expected {
		t.Error(string(json), expected)
	}
	if 3 != events.numSeen {
		t.Error(events.numSeen)
	}
	if 3 != events.NumSaved() {
		t.Error(events.NumSaved())
	}
}

func TestEmpty(t *testing.T) {
	events := newAnalyticsEvents(10)
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

func TestSampling(t *testing.T) {
	events := newAnalyticsEvents(3)
	events.addEvent(sampleAnalyticsEvent(0.999999))
	events.addEvent(sampleAnalyticsEvent(0.1))
	events.addEvent(sampleAnalyticsEvent(0.9))
	events.addEvent(sampleAnalyticsEvent(0.2))
	events.addEvent(sampleAnalyticsEvent(0.8))
	events.addEvent(sampleAnalyticsEvent(0.3))

	json, err := events.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}
	if string(json) != `["12345",{"reservoir_size":3,"events_seen":6},[0.8,0.999999,0.9]]` {
		t.Error(string(json))
	}
	if 6 != events.numSeen {
		t.Error(events.numSeen)
	}
	if 3 != events.NumSaved() {
		t.Error(events.NumSaved())
	}
}

func TestMergeEmpty(t *testing.T) {
	e1 := newAnalyticsEvents(10)
	e2 := newAnalyticsEvents(10)
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

func TestMergeFull(t *testing.T) {
	e1 := newAnalyticsEvents(2)
	e2 := newAnalyticsEvents(3)

	e1.addEvent(sampleAnalyticsEvent(0.1))
	e1.addEvent(sampleAnalyticsEvent(0.15))
	e1.addEvent(sampleAnalyticsEvent(0.25))

	e2.addEvent(sampleAnalyticsEvent(0.06))
	e2.addEvent(sampleAnalyticsEvent(0.12))
	e2.addEvent(sampleAnalyticsEvent(0.18))
	e2.addEvent(sampleAnalyticsEvent(0.24))

	e1.Merge(e2)
	json, err := e1.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}
	if string(json) != `["12345",{"reservoir_size":2,"events_seen":7},[0.24,0.25]]` {
		t.Error(string(json))
	}
	if 7 != e1.numSeen {
		t.Error(e1.numSeen)
	}
	if 2 != e1.NumSaved() {
		t.Error(e1.NumSaved())
	}
}

func TestAnalyticsEventMergeFailedSuccess(t *testing.T) {
	e1 := newAnalyticsEvents(2)
	e2 := newAnalyticsEvents(3)

	e1.addEvent(sampleAnalyticsEvent(0.1))
	e1.addEvent(sampleAnalyticsEvent(0.15))
	e1.addEvent(sampleAnalyticsEvent(0.25))

	e2.addEvent(sampleAnalyticsEvent(0.06))
	e2.addEvent(sampleAnalyticsEvent(0.12))
	e2.addEvent(sampleAnalyticsEvent(0.18))
	e2.addEvent(sampleAnalyticsEvent(0.24))

	e1.mergeFailed(e2)

	json, err := e1.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}
	if string(json) != `["12345",{"reservoir_size":2,"events_seen":7},[0.24,0.25]]` {
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

func TestAnalyticsEventMergeFailedLimitReached(t *testing.T) {
	e1 := newAnalyticsEvents(2)
	e2 := newAnalyticsEvents(3)

	e1.addEvent(sampleAnalyticsEvent(0.1))
	e1.addEvent(sampleAnalyticsEvent(0.15))
	e1.addEvent(sampleAnalyticsEvent(0.25))

	e2.addEvent(sampleAnalyticsEvent(0.06))
	e2.addEvent(sampleAnalyticsEvent(0.12))
	e2.addEvent(sampleAnalyticsEvent(0.18))
	e2.addEvent(sampleAnalyticsEvent(0.24))

	e2.failedHarvests = failedEventsAttemptsLimit

	e1.mergeFailed(e2)

	json, err := e1.CollectorJSON(agentRunID)
	if nil != err {
		t.Fatal(err)
	}
	if string(json) != `["12345",{"reservoir_size":2,"events_seen":3},[0.15,0.25]]` {
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

func analyticsEventBenchmarkHelper(b *testing.B, w jsonWriter) {
	events := newAnalyticsEvents(MaxTxnEvents)
	event := analyticsEvent{0, w}
	for n := 0; n < MaxTxnEvents; n++ {
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
	event := &TxnEvent{
		FinalName: "WebTransaction/Go/zip/zap",
		Start:     time.Now(),
		Duration:  2 * time.Second,
		Queuing:   1 * time.Second,
		Zone:      ApdexSatisfying,
		Attrs:     nil,
	}
	analyticsEventBenchmarkHelper(b, event)
}

func BenchmarkCustomEventsCollectorJSON(b *testing.B) {
	now := time.Now()
	ce, err := CreateCustomEvent("myEventType", map[string]interface{}{
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
	e := TxnErrorFromResponseCode(time.Now(), 503)
	e.Stack = GetStackTrace()

	txnName := "WebTransaction/Go/zip/zap"
	event := &ErrorEvent{
		ErrorData: e,
		TxnEvent: TxnEvent{
			FinalName: txnName,
			Duration:  3 * time.Second,
			Attrs:     nil,
		},
	}
	analyticsEventBenchmarkHelper(b, event)
}

func TestSplitFull(t *testing.T) {
	events := newAnalyticsEvents(10)
	for i := 0; i < 15; i++ {
		events.addEvent(sampleAnalyticsEvent(Priority(float32(i) / 10.0)))
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
	events := newAnalyticsEvents(10)
	for i := 0; i < 7; i++ {
		events.addEvent(sampleAnalyticsEvent(Priority(float32(i) / 10.0)))
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
	events := newAnalyticsEvents(10)
	for i := 0; i < 8; i++ {
		events.addEvent(sampleAnalyticsEvent(Priority(float32(i) / 10.0)))
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

func TestAnalyticsEventsZeroCapacity(t *testing.T) {
	// Analytics events methods should be safe when configurable harvest
	// settings have an event limit of zero.
	events := newAnalyticsEvents(0)
	if 0 != events.NumSeen() || 0 != events.NumSaved() || 0 != events.capacity() {
		t.Error(events.NumSeen(), events.NumSaved(), events.capacity())
	}
	events.addEvent(sampleAnalyticsEvent(0.5))
	if 1 != events.NumSeen() || 0 != events.NumSaved() || 0 != events.capacity() {
		t.Error(events.NumSeen(), events.NumSaved(), events.capacity())
	}
	js, err := events.CollectorJSON("agentRunID")
	if err != nil || js != nil {
		t.Error(err, string(js))
	}
}
