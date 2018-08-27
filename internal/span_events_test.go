package internal

import (
	"encoding/json"
	"testing"
	"time"
)

func testSpanEventJSON(t *testing.T, e *SpanEvent, expect string) {
	js, err := json.Marshal(e)
	if nil != err {
		t.Error(err)
		return
	}
	expect = CompactJSONString(expect)
	if string(js) != expect {
		t.Errorf("\nexpect=%s\nactual=%s\n", expect, string(js))
	}
}

var (
	sampleSpanEvent = SpanEvent{
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

	sampleSpanDatastoreExtras = spanDatastoreExtras{
		Component: "mySql",
		Statement: "SELECT * from foo",
		Instance:  "123",
		Address:   "{host}:{portPathOrId}",
		Hostname:  "host",
	}

	sampleSpanExternalExtras = spanExternalExtras{
		URL:       "http://url.com",
		Method:    "GET",
		Component: "http",
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
	e.DatastoreExtras = &sampleSpanDatastoreExtras

	testSpanEventJSON(t, &e, `[
	{
		"type":"Span",
		"traceId":"trace-id",
		"guid":"guid",
		"parentId":"parent-id",
		"transactionId":"txn-id",
		"sampled":true,
		"priority":0.500000,
		"timestamp":1488393111000,
		"duration":2,
		"name":"myName",
		"category":"datastore",
		"component":"mySql",
		"db.statement":"SELECT * from foo",
		"db.instance":"123",
		"peer.address":"{host}:{portPathOrId}",
		"peer.hostname":"host",
		"span.kind":"client"
	},
	{},
	{}]`)
}

func TestSpanEventDatastoreWithoutHostMarshal(t *testing.T) {
	e := sampleSpanEvent

	// Alter sample span event for this test case
	e.IsEntrypoint = false
	e.ParentID = "parent-id"
	e.Category = spanCategoryDatastore
	e.DatastoreExtras = &sampleSpanDatastoreExtras
	e.DatastoreExtras.Hostname = ""
	e.DatastoreExtras.Address = ""

	// According to CHANGELOG.md, as of version 1.5, if `Host` and
	// `PortPathOrID` are not provided in a Datastore segment, they
	// do not appear as `"unknown"` in transaction traces and slow
	// query traces.  To maintain parity with the other offerings of
	// the Go Agent, neither do Span Events.
	testSpanEventJSON(t, &e, `[
	{
		"type":"Span",
		"traceId":"trace-id",
		"guid":"guid",
		"parentId":"parent-id",
		"transactionId":"txn-id",
		"sampled":true,
		"priority":0.500000,
		"timestamp":1488393111000,
		"duration":2,
		"name":"myName",
		"category":"datastore",
		"component":"mySql",
		"db.statement":"SELECT * from foo",
		"db.instance":"123",
		"span.kind":"client"
	},
	{},
	{}]`)
}

func TestSpanEventExternalMarshal(t *testing.T) {
	e := sampleSpanEvent

	// Alter sample span event for this test case
	e.ParentID = "parent-id"
	e.IsEntrypoint = false
	e.Category = spanCategoryHTTP
	e.ExternalExtras = &sampleSpanExternalExtras

	testSpanEventJSON(t, &e, `[
	{
		"type":"Span",
		"traceId":"trace-id",
		"guid":"guid",
		"parentId":"parent-id",
		"transactionId":"txn-id",
		"sampled":true,
		"priority":0.500000,
		"timestamp":1488393111000,
		"duration":2,
		"name":"myName",
		"category":"http",
		"http.url":"http://url.com",
		"http.method":"GET",
		"span.kind":"client",
		"component":"http"
	},
	{},
	{}]`)
}

func TestSpanEventsEndpointMethod(t *testing.T) {
	events := &spanEvents{}
	m := events.EndpointMethod()
	if m != cmdSpanEvents {
		t.Error(m)
	}
}
