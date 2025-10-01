// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
)

func testSpanEventJSON(t *testing.T, e *spanEvent, expect string) {
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
	sampleSpanEvent = spanEvent{
		TraceID:       "trace-id",
		GUID:          "guid",
		TransactionID: "txn-id",
		Sampled:       true,
		Priority:      0.5,
		Timestamp:     timeFromUnixMilliseconds(1488393111000),
		Duration:      2 * time.Second,
		Name:          "myName",
		Category:      spanCategoryGeneric,
		IsEntrypoint:  true,
	}
)

func TestSpanEventGenericRootMarshal(t *testing.T) {
	e := sampleSpanEvent
	testSpanEventJSON(t, &e, `[
	{
		"type":"Span",
		"traceId":"trace-id",
		"guid":"guid",
		"transactionId":"txn-id",
		"sampled":true,
		"priority":0.500000,
		"timestamp":1488393111000,
		"duration":2,
		"name":"myName",
		"category":"generic",
		"nr.entryPoint":true
	},
	{},
	{}]`)
}

func TestSpanEventDatastoreMarshal(t *testing.T) {
	e := sampleSpanEvent

	// Alter sample span event for this test case
	e.IsEntrypoint = false
	e.ParentID = "parent-id"
	e.Category = spanCategoryDatastore
	e.Kind = "client"
	e.Component = "mySql"
	e.AgentAttributes.addString(SpanAttributeDBStatement, "SELECT * from foo")
	e.AgentAttributes.addString(SpanAttributeDBInstance, "123")
	e.AgentAttributes.addString(SpanAttributePeerAddress, "{host}:{portPathOrId}")
	e.AgentAttributes.addString(SpanAttributePeerHostname, "host")

	expectEvent(t, &e, internal.WantEvent{
		Intrinsics: map[string]interface{}{
			"type":          "Span",
			"traceId":       "trace-id",
			"guid":          "guid",
			"parentId":      "parent-id",
			"transactionId": "txn-id",
			"sampled":       true,
			"priority":      0.500000,
			"timestamp":     1.488393111e+12,
			"duration":      2,
			"name":          "myName",
			"category":      "datastore",
			"component":     "mySql",
			"span.kind":     "client",
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"db.statement":  "SELECT * from foo",
			"db.instance":   "123",
			"peer.address":  "{host}:{portPathOrId}",
			"peer.hostname": "host",
		},
	})
}

func TestSpanEventDatastoreWithoutHostMarshal(t *testing.T) {
	e := sampleSpanEvent

	// Alter sample span event for this test case
	e.IsEntrypoint = false
	e.ParentID = "parent-id"
	e.Category = spanCategoryDatastore
	e.Kind = "client"
	e.Component = "mySql"
	e.AgentAttributes.addString(SpanAttributeDBStatement, "SELECT * from foo")
	e.AgentAttributes.addString(SpanAttributeDBInstance, "123")

	// According to CHANGELOG.md, as of version 1.5, if `Host` and
	// `PortPathOrID` are not provided in a Datastore segment, they
	// do not appear as `"unknown"` in transaction traces and slow
	// query traces.  To maintain parity with the other offerings of
	// the Go Agent, neither do Span Events.
	expectEvent(t, &e, internal.WantEvent{
		Intrinsics: map[string]interface{}{
			"type":          "Span",
			"traceId":       "trace-id",
			"guid":          "guid",
			"parentId":      "parent-id",
			"transactionId": "txn-id",
			"sampled":       true,
			"priority":      0.500000,
			"timestamp":     1.488393111e+12,
			"duration":      2,
			"name":          "myName",
			"category":      "datastore",
			"component":     "mySql",
			"span.kind":     "client",
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"db.statement": "SELECT * from foo",
			"db.instance":  "123",
		},
	})
}

func TestSpanEventExternalMarshal(t *testing.T) {
	e := sampleSpanEvent

	// Alter sample span event for this test case
	e.ParentID = "parent-id"
	e.IsEntrypoint = false
	e.Category = spanCategoryHTTP
	e.Kind = "client"
	e.Component = "http"
	e.AgentAttributes.addString(SpanAttributeHTTPURL, "http://url.com")
	e.AgentAttributes.addString(SpanAttributeHTTPMethod, "GET")

	expectEvent(t, &e, internal.WantEvent{
		Intrinsics: map[string]interface{}{
			"type":          "Span",
			"traceId":       "trace-id",
			"guid":          "guid",
			"parentId":      "parent-id",
			"transactionId": "txn-id",
			"sampled":       true,
			"priority":      0.500000,
			"timestamp":     1.488393111e+12,
			"duration":      2,
			"name":          "myName",
			"category":      "http",
			"component":     "http",
			"span.kind":     "client",
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"http.url":    "http://url.com",
			"http.method": "GET",
		},
	})
}

func TestSpanEventsEndpointMethod(t *testing.T) {
	events := &spanEvents{}
	m := events.EndpointMethod()
	if m != cmdSpanEvents {
		t.Error(m)
	}
}

func TestSpanEventsMerge(t *testing.T) {

	events := []*spanEvent{
		{
			GUID:          "span-1-id",
			ParentID:      "root-span-id",
			Timestamp:     time.Now(),
			Duration:      3 * time.Millisecond,
			Name:          "span1",
			Category:      spanCategoryGeneric,
			IsEntrypoint:  false,
			Sampled:       true,
			Priority:      0.7,
			TransactionID: "txn-id",
			TraceID:       "inbound-trace-id",
		},
		{
			GUID:          "span-2-id",
			ParentID:      "span-1-id",
			Timestamp:     time.Now(),
			Duration:      3 * time.Millisecond,
			Name:          "span2",
			Category:      spanCategoryGeneric,
			IsEntrypoint:  false,
			Sampled:       true,
			Priority:      0.7,
			TransactionID: "txn-id",
			TraceID:       "inbound-trace-id",
		},
	}

	spanEvents := newSpanEvents(10)
	spanEvents.MergeSpanEvents(events)

	expectSpanEvents(t, spanEvents, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "span1",
				"sampled":       true,
				"priority":      0.7,
				"category":      spanCategoryGeneric,
				"parentId":      "root-span-id",
				"guid":          "span-1-id",
				"transactionId": "txn-id",
				"traceId":       "inbound-trace-id",
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":          "span2",
				"sampled":       true,
				"priority":      0.7,
				"category":      spanCategoryGeneric,
				"parentId":      "span-1-id",
				"guid":          "span-2-id",
				"transactionId": "txn-id",
				"traceId":       "inbound-trace-id",
			},
		},
	})
}
