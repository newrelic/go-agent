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
		ParentID:      "parent-id",
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
		Address:   "url.com",
		Hostname:  "host",
	}

	sampleSpanExternalExtras = spanExternalExtras{
		URL:       "http://url.com",
		Method:    "GET",
		Component: "http",
	}
)

// TODO: Is nr.entrypoint the correct payload field for indicating that a span is a root span?
func TestSpanEventGenericMarshal(t *testing.T) {
	e := sampleSpanEvent
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
		"peer.address":"url.com",
		"peer.hostname":"host",
		"span.kind":"client"
	},
	{},
	{}]`)
}

func TestSpanEventExternalMarshal(t *testing.T) {
	e := sampleSpanEvent

	// Alter sample span event for this test case
	e.Category = spanCategoryHTTP
	e.IsEntrypoint = false
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
