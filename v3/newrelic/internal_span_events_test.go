// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
)

func TestSpanEventSuccess(t *testing.T) {
	// Test that a basic segment creates a span event, and that a
	// transaction has a root span event.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
		reply.TraceIDGenerator = internal.NewTraceIDGenerator(12345)
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	segment := txn.StartSegment("mySegment")
	segment.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "Custom/mySegment",
				"sampled":       true,
				"category":      "generic",
				"priority":      internal.MatchAnything,
				"guid":          "e71870997d57214c",
				"transactionId": "1ae969564b34a33e",
				"traceId":       "1ae969564b34a33ecd1af05fe6923d6d",
				"parentId":      "4259d74b863e2fba",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"priority":         internal.MatchAnything,
				"guid":             "4259d74b863e2fba",
				"transactionId":    "1ae969564b34a33e",
				"nr.entryPoint":    true,
				"traceId":          "1ae969564b34a33ecd1af05fe6923d6d",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestSpanEventsLocallyDisabled(t *testing.T) {
	// Test that span events do not get created if Config.SpanEvents.Enabled
	// is false.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	segment := txn.StartSegment("mySegment")
	segment.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{})
}

func TestSpanEventsRemotelyDisabled(t *testing.T) {
	// Test that span events do not get created if the connect reply
	// disables span events.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
		reply.CollectSpanEvents = false
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	segment := txn.StartSegment("mySegment")
	segment.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{})
}

func TestSpanEventsDisabledWithoutDistributedTracing(t *testing.T) {
	// Test that span events do not get created distributed tracing is not
	// enabled.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	segment := txn.StartSegment("mySegment")
	segment.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{})
}

func TestSpanEventDatastoreExternal(t *testing.T) {
	// Test that a datastore and external segments creates the correct span
	// events.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	segment := DatastoreSegment{
		StartTime:          txn.StartSegmentNow(),
		Product:            DatastoreMySQL,
		Collection:         "mycollection",
		Operation:          "myoperation",
		ParameterizedQuery: "myquery",
		Host:               "myhost",
		PortPathOrID:       "myport",
		DatabaseName:       "dbname",
	}
	segment.End()
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	s := StartExternalSegment(txn, req)
	s.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"sampled":   true,
				"name":      "Datastore/statement/MySQL/mycollection/myoperation",
				"category":  "datastore",
				"component": "MySQL",
				"span.kind": "client",
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"db.statement":  "myquery",
				"db.instance":   "dbname",
				"db.collection": "mycollection",
				"peer.address":  "myhost:myport",
				"peer.hostname": "myhost",
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"name":      "External/example.com/http/GET",
				"category":  "http",
				"component": "http",
				"span.kind": "client",
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"http.url":    "http://example.com",
				"http.method": "GET",
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestSpanEventAttributesDisabled(t *testing.T) {
	// Test that SpanEvents.Attributes.Enabled correctly disables span
	// attributes.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Attributes.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	segment := DatastoreSegment{
		StartTime:          txn.StartSegmentNow(),
		Product:            DatastoreMySQL,
		Collection:         "mycollection",
		Operation:          "myoperation",
		ParameterizedQuery: "myquery",
		Host:               "myhost",
		PortPathOrID:       "myport",
		DatabaseName:       "dbname",
	}
	segment.End()
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	s := StartExternalSegment(txn, req)
	s.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"sampled":   true,
				"name":      "Datastore/statement/MySQL/mycollection/myoperation",
				"category":  "datastore",
				"component": "MySQL",
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"name":      "External/example.com/http/GET",
				"category":  "http",
				"component": "http",
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestSpanEventAttributesSpecificallyExcluded(t *testing.T) {
	// Test that SpanEvents.Attributes.Exclude excludes span attributes.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Attributes.Exclude = []string{
			SpanAttributeDBStatement,
			SpanAttributeDBInstance,
			SpanAttributeDBCollection,
			SpanAttributePeerAddress,
			SpanAttributePeerHostname,
			SpanAttributeHTTPURL,
			SpanAttributeHTTPMethod,
		}
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	segment := DatastoreSegment{
		StartTime:          txn.StartSegmentNow(),
		Product:            DatastoreMySQL,
		Collection:         "mycollection",
		Operation:          "myoperation",
		ParameterizedQuery: "myquery",
		Host:               "myhost",
		PortPathOrID:       "myport",
		DatabaseName:       "dbname",
	}
	segment.End()
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	s := StartExternalSegment(txn, req)
	s.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"sampled":   true,
				"name":      "Datastore/statement/MySQL/mycollection/myoperation",
				"category":  "datastore",
				"component": "MySQL",
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"name":      "External/example.com/http/GET",
				"category":  "http",
				"component": "http",
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestSpanEventAttributesExcluded(t *testing.T) {
	// Test that Attributes.Exclude excludes span attributes.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.Attributes.Exclude = []string{
			SpanAttributeDBStatement,
			SpanAttributeDBInstance,
			SpanAttributeDBCollection,
			SpanAttributePeerAddress,
			SpanAttributePeerHostname,
			SpanAttributeHTTPURL,
			SpanAttributeHTTPMethod,
		}
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	segment := DatastoreSegment{
		StartTime:          txn.StartSegmentNow(),
		Product:            DatastoreMySQL,
		Collection:         "mycollection",
		Operation:          "myoperation",
		ParameterizedQuery: "myquery",
		Host:               "myhost",
		PortPathOrID:       "myport",
		DatabaseName:       "dbname",
	}
	segment.End()
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	s := StartExternalSegment(txn, req)
	s.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"sampled":   true,
				"name":      "Datastore/statement/MySQL/mycollection/myoperation",
				"category":  "datastore",
				"component": "MySQL",
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"name":      "External/example.com/http/GET",
				"category":  "http",
				"component": "http",
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestSpanAttributesFromTxnExcludedOnTxn(t *testing.T) {
	// Test that attributes on the root span that come from transactions do not
	// get excluded when excluded with TransactionEvents.Attributes.Exclude.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.TransactionEvents.Attributes.Exclude = []string{
			AttributeRequestMethod,
			AttributeRequestURI,
			AttributeRequestHost,
		}
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	txn.SetWebRequestHTTP(req)
	txn.End()
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":             "WebTransaction/Go/hello",
				"sampled":          true,
				"nr.apdexPerfZone": "S",
				"guid":             internal.MatchAnything,
				"traceId":          internal.MatchAnything,
				"priority":         internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":             "WebTransaction/Go/hello",
				"transaction.name": "WebTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"request.method":       "GET",
				"request.uri":          "http://example.com",
				"request.headers.host": "example.com",
			},
		},
	})
}

func TestSpanAttributesFromTxnExcludedByDefault(t *testing.T) {
	// Test that the user-agent attribute on span events is excluded by
	// default but can be included with SpanEvents.Attributes.Include.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	req.Header.Add("User-Agent", "sample user agent")
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	txn.SetWebRequestHTTP(req)
	txn.End()
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":             "WebTransaction/Go/hello",
				"sampled":          true,
				"nr.apdexPerfZone": "S",
				"guid":             internal.MatchAnything,
				"traceId":          internal.MatchAnything,
				"priority":         internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"request.method":       "GET",
				"request.uri":          "http://example.com",
				"request.headers.host": "example.com",
			},
		},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":             "WebTransaction/Go/hello",
				"transaction.name": "WebTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"request.method":       "GET",
				"request.uri":          "http://example.com",
				"request.headers.host": "example.com",
			},
		},
	})

	cfgfn = func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Attributes.Include = []string{
			AttributeRequestUserAgent,
		}
	}
	app = testApp(replyfn, cfgfn, t)
	txn = app.StartTransaction("hello")
	txn.SetWebRequestHTTP(req)
	txn.End()
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":             "WebTransaction/Go/hello",
				"sampled":          true,
				"nr.apdexPerfZone": "S",
				"guid":             internal.MatchAnything,
				"traceId":          internal.MatchAnything,
				"priority":         internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"request.method":       "GET",
				"request.uri":          "http://example.com",
				"request.headers.host": "example.com",
			},
		},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":             "WebTransaction/Go/hello",
				"transaction.name": "WebTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"request.method":            "GET",
				"request.uri":               "http://example.com",
				"request.headers.userAgent": "sample user agent",
				"request.headers.host":      "example.com",
			},
		},
	})
}

func TestSpanAttributesFromTxnExcludedOnSpan(t *testing.T) {
	// Test that attributes on transaction that are shared with the root span
	// do not get excluded when excluded with SpanEvents.Attributes.Exclude.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Attributes.Exclude = []string{
			AttributeRequestMethod,
			AttributeRequestURI,
			AttributeRequestHost,
		}
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	txn.SetWebRequestHTTP(req)
	txn.End()
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":             "WebTransaction/Go/hello",
				"sampled":          true,
				"nr.apdexPerfZone": "S",
				"guid":             internal.MatchAnything,
				"traceId":          internal.MatchAnything,
				"priority":         internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"request.method":       "GET",
				"request.uri":          "http://example.com",
				"request.headers.host": "example.com",
			},
		},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":             "WebTransaction/Go/hello",
				"transaction.name": "WebTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestSpanAttributesFromTxnExcludedGlobally(t *testing.T) {
	// Test that attributes on transaction that are shared with the root span
	// get excluded from both when excluded with Attributes.Exclude.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.Attributes.Exclude = []string{
			AttributeRequestMethod,
			AttributeRequestURI,
			AttributeRequestHost,
		}
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	txn.SetWebRequestHTTP(req)
	txn.End()
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":             "WebTransaction/Go/hello",
				"sampled":          true,
				"nr.apdexPerfZone": "S",
				"guid":             internal.MatchAnything,
				"traceId":          internal.MatchAnything,
				"priority":         internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":             "WebTransaction/Go/hello",
				"transaction.name": "WebTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestSpanEventAttributesLASP(t *testing.T) {
	// Test that security policies prevent the capture of the input query
	// statement.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
		reply.SecurityPolicies.RecordSQL.SetEnabled(false)
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	segment := DatastoreSegment{
		StartTime:          txn.StartSegmentNow(),
		Product:            DatastoreMySQL,
		Collection:         "mycollection",
		Operation:          "myoperation",
		ParameterizedQuery: "myquery",
		Host:               "myhost",
		PortPathOrID:       "myport",
		DatabaseName:       "dbname",
	}
	segment.End()
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	s := StartExternalSegment(txn, req)
	s.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"sampled":   true,
				"name":      "Datastore/statement/MySQL/mycollection/myoperation",
				"category":  "datastore",
				"component": "MySQL",
				"span.kind": "client",
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"db.instance":   "dbname",
				"db.collection": "mycollection",
				"peer.address":  "myhost:myport",
				"peer.hostname": "myhost",
				"db.statement":  "'myoperation' on 'mycollection' using 'MySQL'",
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"name":      "External/example.com/http/GET",
				"category":  "http",
				"component": "http",
				"span.kind": "client",
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"http.url":    "http://example.com",
				"http.method": "GET",
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestAddAgentSpanAttribute(t *testing.T) {
	// Test that AddAgentSpanAttribute successfully adds attributes to
	// spans.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	s := txn.StartSegment("hi")
	internal.AddAgentSpanAttribute(txn.Private, SpanAttributeAWSRegion, "west")
	internal.AddAgentSpanAttribute(txn.Private, SpanAttributeAWSRequestID, "123")
	internal.AddAgentSpanAttribute(txn.Private, SpanAttributeAWSOperation, "secret")
	s.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":     "Custom/hi",
				"sampled":  true,
				"category": "generic",
				"parentId": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"aws.operation": "secret",
				"aws.requestId": "123",
				"aws.region":    "west",
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestAddAgentSpanAttributeExcluded(t *testing.T) {
	// Test that span attributes added by AddAgentSpanAttribute are subject
	// to span attribute configuration.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Attributes.Exclude = []string{
			SpanAttributeAWSOperation,
			SpanAttributeAWSRequestID,
			SpanAttributeAWSRegion,
		}
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	s := txn.StartSegment("hi")
	internal.AddAgentSpanAttribute(txn.Private, SpanAttributeAWSRegion, "west")
	internal.AddAgentSpanAttribute(txn.Private, SpanAttributeAWSRequestID, "123")
	internal.AddAgentSpanAttribute(txn.Private, SpanAttributeAWSOperation, "secret")
	s.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":     "Custom/hi",
				"sampled":  true,
				"category": "generic",
				"parentId": internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestAddSpanAttributeNoActiveSpan(t *testing.T) {
	// Test that AddAgentSpanAttribute does not have problems if called when
	// there is no active span.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SetSampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	// Do not panic if there are no active spans!
	internal.AddAgentSpanAttribute(txn.Private, SpanAttributeAWSRegion, "west")
	internal.AddAgentSpanAttribute(txn.Private, SpanAttributeAWSRequestID, "123")
	internal.AddAgentSpanAttribute(txn.Private, SpanAttributeAWSOperation, "secret")
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestAddSpanAttributeNilTransaction(t *testing.T) {
	// Test that AddAgentSpanAttribute does not panic if the transaction is
	// nil.
	internal.AddAgentSpanAttribute(nil, SpanAttributeAWSRegion, "west")
	internal.AddAgentSpanAttribute(nil, SpanAttributeAWSRequestID, "123")
	internal.AddAgentSpanAttribute(nil, SpanAttributeAWSOperation, "secret")
}

func TestSpanEventHTTPStatusCode(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello")
	resp := &http.Response{
		StatusCode: 13,
	}
	s := ExternalSegment{
		StartTime: txn.StartSegmentNow(),
		Response:  resp,
	}
	s.SetStatusCode(0)
	s.End()
	app.expectNoLoggedErrors(t)
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"name":      "External/unknown/http",
				"category":  "http",
				"component": "http",
				"span.kind": "client",
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				// SetStatusCode takes precedence over Response.StatusCode
				"http.statusCode": 0,
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestSpanEvent_TxnCustomAttrsAreCopied(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello")
	s := txn.StartSegment("segment")
	s.End()
	key := "attr-key"
	value := "attr-value"
	txn.AddAttribute(key, value)
	txn.End()
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":     "OtherTransaction/Go/hello",
				"traceId":  "52fdfc072182654f163f5f0f9a621d72",
				"priority": internal.MatchAnything,
				"guid":     "52fdfc072182654f",
				"sampled":  true,
			},
			UserAttributes: map[string]interface{}{
				key: value,
			},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"parentId": internal.MatchAnything,
				"name":     "Custom/segment",
				"category": "generic",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			// Txn custom attrs should get copied to the root span
			UserAttributes: map[string]interface{}{
				key: value,
			},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestSpanEvent_TxnCustomAttrsAreExcluded_OnlyFromTxn(t *testing.T) {
	app := testApp(distributedTracingReplyFields, func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.TransactionEvents.Attributes.Exclude = []string{AttributeRequestMethod}
	}, t)
	txn := app.StartTransaction("hello")
	s := txn.StartSegment("segment")
	s.End()
	txn.AddAttribute(AttributeRequestMethod, "attr-value")
	txn.End()

	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":     "OtherTransaction/Go/hello",
				"traceId":  "52fdfc072182654f163f5f0f9a621d72",
				"priority": internal.MatchAnything,
				"guid":     "52fdfc072182654f",
				"sampled":  true,
			},
			// the custom attr should be filtered out
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"parentId": internal.MatchAnything,
				"name":     "Custom/segment",
				"category": "generic",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			UserAttributes: map[string]interface{}{
				AttributeRequestMethod: "attr-value",
			},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestSpanEvent_TxnCustomAttrsAreExcluded_OnlyFromSpans(t *testing.T) {
	app := testApp(distributedTracingReplyFields, func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Attributes.Exclude = []string{AttributeRequestMethod}
	}, t)
	txn := app.StartTransaction("hello")
	s := txn.StartSegment("segment")
	s.End()
	txn.AddAttribute(AttributeRequestMethod, "attr-value")
	txn.End()

	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":     "OtherTransaction/Go/hello",
				"traceId":  "52fdfc072182654f163f5f0f9a621d72",
				"priority": internal.MatchAnything,
				"guid":     "52fdfc072182654f",
				"sampled":  true,
			},
			UserAttributes: map[string]interface{}{
				AttributeRequestMethod: "attr-value",
			},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"parentId": internal.MatchAnything,
				"name":     "Custom/segment",
				"category": "generic",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			// the custom attr should be filtered out
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestSpanEventExcludeCustomAttrs(t *testing.T) {
	app := testApp(distributedTracingReplyFields, func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Attributes.Exclude = []string{"attribute"}
	}, t)
	txn := app.StartTransaction("hello")
	s := txn.StartSegment("segment")
	s.AddAttribute("attribute", "value")
	s.End()
	txn.End()

	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"parentId": internal.MatchAnything,
				"name":     "Custom/segment",
				"category": "generic",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			// the custom attr should be filtered out
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestAddSpanAttributeHighSecurity(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.HighSecurity = true
	}
	app := testApp(distributedTracingReplyFields, cfgfn, t)
	txn := app.StartTransaction("hello")
	seg := txn.StartSegment("segment")
	seg.AddAttribute("key", 1)
	app.expectSingleLoggedError(t, "unable to add segment attribute", map[string]interface{}{
		"reason": errHighSecurityEnabled.Error(),
	})
	seg.End()
	txn.End()

	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"parentId": internal.MatchAnything,
				"name":     "Custom/segment",
				"category": "generic",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			// the custom attr should not be added
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestAddSpanAttributeSecurityPolicyDisablesParameters(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.SecurityPolicies.CustomParameters.SetEnabled(false)
	}
	app := testApp(replyfn, enableBetterCAT, t)
	txn := app.StartTransaction("hello")
	seg := txn.StartSegment("segment")
	seg.AddAttribute("key", 1)
	app.expectSingleLoggedError(t, "unable to add segment attribute", map[string]interface{}{
		"reason": errSecurityPolicy.Error(),
	})
	seg.End()
	txn.End()

	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"parentId": internal.MatchAnything,
				"name":     "Custom/segment",
				"category": "generic",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"category":         "generic",
				"nr.entryPoint":    true,
			},
			// the custom attr should not be added
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}
