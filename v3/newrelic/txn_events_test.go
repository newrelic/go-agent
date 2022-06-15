// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal/cat"
)

func testTxnEventJSON(t testing.TB, e *txnEvent, expect string) {
	// Type assertion to support early Go versions.
	if h, ok := t.(interface {
		Helper()
	}); ok {
		h.Helper()
	}
	js, err := json.Marshal(e)
	if nil != err {
		t.Error(err)
		return
	}
	expect = compactJSONString(expect)
	if string(js) != expect {
		t.Errorf("\nexpect=%s\nactual=%s\n", expect, string(js))
	}
}

var (
	sampleTxnEvent = txnEvent{
		FinalName: "myName",
		BetterCAT: betterCAT{
			Enabled:  true,
			TxnID:    "txn-id",
			TraceID:  "trace-id",
			Priority: 0.5,
		},
		Start:     timeFromUnixMilliseconds(1488393111000),
		Duration:  2 * time.Second,
		TotalTime: 3 * time.Second,
		Zone:      apdexNone,
		Attrs:     nil,
	}

	sampleTxnEventWithOldCAT = txnEvent{
		FinalName: "myOldName",
		BetterCAT: betterCAT{
			Enabled: false,
		},
		Start:     timeFromUnixMilliseconds(1488393111000),
		Duration:  2 * time.Second,
		TotalTime: 3 * time.Second,
		Zone:      apdexNone,
		Attrs:     nil,
	}

	sampleTxnEventWithError = txnEvent{
		FinalName: "myName",
		BetterCAT: betterCAT{
			Enabled:  true,
			TxnID:    "txn-id",
			TraceID:  "trace-id",
			Priority: 0.5,
		},
		Start:     timeFromUnixMilliseconds(1488393111000),
		Duration:  2 * time.Second,
		TotalTime: 3 * time.Second,
		Zone:      apdexNone,
		Attrs:     nil,
		HasError:  true,
	}
)

func TestTxnEventMarshal(t *testing.T) {
	e := sampleTxnEvent
	testTxnEventJSON(t, &e, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1488393111000,
		"error":false,
		"duration":2,
		"totalTime":3,
		"guid":"txn-id",
		"traceId":"trace-id",
		"priority":0.500000,
		"sampled":false
	},
	{},
	{}]`)
}

func TestTxnEventMarshalWithApdex(t *testing.T) {
	e := sampleTxnEvent
	e.Zone = apdexFailing
	testTxnEventJSON(t, &e, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1488393111000,
		"nr.apdexPerfZone":"F",
		"error":false,
		"duration":2,
		"totalTime":3,
		"guid":"txn-id",
		"traceId":"trace-id",
		"priority":0.500000,
		"sampled":false
	},
	{},
	{}]`)
}

func TestTxnEventMarshalWithDatastoreExternal(t *testing.T) {
	e := sampleTxnEvent
	e.externalCallCount = 22
	e.externalDuration = 1122334 * time.Millisecond
	e.datastoreCallCount = 33
	e.datastoreDuration = 5566778 * time.Millisecond

	testTxnEventJSON(t, &e, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1488393111000,
		"error":false,
		"duration":2,
		"externalCallCount":22,
		"externalDuration":1122.334,
		"databaseCallCount":33,
		"databaseDuration":5566.778,
		"totalTime":3,
		"guid":"txn-id",
		"traceId":"trace-id",
		"priority":0.500000,
		"sampled":false
	},
	{},
	{}]`)
}

func TestTxnEventMarshalWithInboundCaller(t *testing.T) {
	e := sampleTxnEvent
	e.BetterCAT.Inbound = &payload{
		Type:                 "Browser",
		App:                  "caller-app",
		Account:              "caller-account",
		ID:                   "caller-id",
		TransactionID:        "caller-parent-id",
		TracedID:             "trip-id",
		TransportDuration:    2 * time.Second,
		HasNewRelicTraceInfo: true,
	}
	e.BetterCAT.TraceID = "trip-id"
	e.BetterCAT.TransportType = "HTTP"
	testTxnEventJSON(t, &e, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1488393111000,
		"error":false,
		"duration":2,
		"totalTime":3,
		"parent.type": "Browser",
		"parent.app": "caller-app",
		"parent.account": "caller-account",
		"parent.transportDuration": 2,
		"parent.transportType": "HTTP",
		"guid":"txn-id",
		"traceId":"trip-id",
		"priority":0.500000,
		"sampled":false,
		"parentId": "caller-parent-id",
		"parentSpanId": "caller-id"
	},
	{},
	{}]`)
}

func TestTxnEventMarshalWithInboundCallerOldCAT(t *testing.T) {
	e := sampleTxnEventWithOldCAT
	e.BetterCAT.Inbound = &payload{
		Type:              "Browser",
		App:               "caller-app",
		Account:           "caller-account",
		ID:                "caller-id",
		TransactionID:     "caller-parent-id",
		TracedID:          "trip-id",
		TransportDuration: 2 * time.Second,
	}
	testTxnEventJSON(t, &e, `[
	{
		"type":"Transaction",
		"name":"myOldName",
		"timestamp":1488393111000,
		"error":false,
		"duration":2,
		"totalTime":3
	},
	{},
	{}]`)
}

func TestTxnEventMarshalWithAttributes(t *testing.T) {
	aci := config{Config: defaultConfig()}
	aci.TransactionEvents.Attributes.Exclude = append(aci.TransactionEvents.Attributes.Exclude, "zap")
	aci.TransactionEvents.Attributes.Exclude = append(aci.TransactionEvents.Attributes.Exclude, AttributeHostDisplayName)
	cfg := createAttributeConfig(aci, true)
	attr := newAttributes(cfg)
	attr.Agent.Add(AttributeHostDisplayName, "exclude me", nil)
	attr.Agent.Add(AttributeRequestMethod, "GET", nil)
	addUserAttribute(attr, "zap", 123, destAll)
	addUserAttribute(attr, "zip", 456, destAll)
	e := sampleTxnEvent
	e.Attrs = attr
	testTxnEventJSON(t, &e, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1488393111000,
		"error":false,
		"duration":2,
		"totalTime":3,
		"guid":"txn-id",
		"traceId":"trace-id",
		"priority":0.500000,
		"sampled":false
	},
	{
		"zip":456
	},
	{
		"request.method":"GET"
	}]`)
}

func TestTxnEventsPayloadsEmpty(t *testing.T) {
	events := newTxnEvents(10)
	ps := events.payloads(5)
	if len(ps) != 1 {
		t.Error(ps)
	}
	if data, err := ps[0].Data("agentRunID", time.Now()); data != nil || err != nil {
		t.Error(data, err)
	}
}

func TestTxnEventsPayloadsUnderLimit(t *testing.T) {
	events := newTxnEvents(10)
	for i := 0; i < 4; i++ {
		events.AddTxnEvent(&txnEvent{}, priority(float32(i)/10.0))
	}
	ps := events.payloads(5)
	if len(ps) != 1 {
		t.Error(ps)
	}
	if data, err := ps[0].Data("agentRunID", time.Now()); data == nil || err != nil {
		t.Error(data, err)
	}
}

func TestTxnEventsPayloadsOverLimit(t *testing.T) {
	events := newTxnEvents(10)
	for i := 0; i < 6; i++ {
		events.AddTxnEvent(&txnEvent{}, priority(float32(i)/10.0))
	}
	ps := events.payloads(5)
	if len(ps) != 2 {
		t.Error(ps)
	}
	if data, err := ps[0].Data("agentRunID", time.Now()); data == nil || err != nil {
		t.Error(data, err)
	}
	if data, err := ps[1].Data("agentRunID", time.Now()); data == nil || err != nil {
		t.Error(data, err)
	}
}

func TestTxnEventsSynthetics(t *testing.T) {
	events := newTxnEvents(1)

	regular := &txnEvent{
		FinalName: "Regular",
		Start:     time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
		Duration:  2 * time.Second,
		Zone:      apdexNone,
		Attrs:     nil,
	}

	synthetics := &txnEvent{
		FinalName: "Synthetics",
		Start:     time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
		Duration:  2 * time.Second,
		Zone:      apdexNone,
		Attrs:     nil,
		CrossProcess: txnCrossProcess{
			Type: txnCrossProcessSynthetics,
			Synthetics: &cat.SyntheticsHeader{
				ResourceID: "resource",
				JobID:      "job",
				MonitorID:  "monitor",
			},
		},
	}

	events.AddTxnEvent(regular, 1.99999)

	// Check that the event was saved.
	if saved := events.analyticsEvents.events[0].jsonWriter; saved != regular {
		t.Errorf("unexpected saved event: expected=%v; got=%v", regular, saved)
	}

	// The priority sampling algorithm is implemented using isLowerPriority().  In
	// the case of an event pool with a single event, an incoming event with the
	// same priority would kick out the event already in the pool.  To really test
	// whether synthetics are given highest deference, add a synthetics event
	// with a really low priority and affirm it kicks out the event already in the
	// pool.
	events.AddTxnEvent(synthetics, 0.0)

	// Check that the event was saved and its priority was appropriately augmented.
	if saved := events.analyticsEvents.events[0].jsonWriter; saved != synthetics {
		t.Errorf("unexpected saved event: expected=%v; got=%v", synthetics, saved)
	}

	if priority := events.analyticsEvents.events[0].priority; priority != 2.0 {
		t.Errorf("synthetics event has unexpected priority: %f", priority)
	}
}

func TestTxnEventMarshalWithError(t *testing.T) {
	e := sampleTxnEventWithError
	testTxnEventJSON(t, &e, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1488393111000,
		"error":true,
		"duration":2,
		"totalTime":3,
		"guid":"txn-id",
		"traceId":"trace-id",
		"priority":0.500000,
		"sampled":false
	},
	{},
	{}]`)
}
