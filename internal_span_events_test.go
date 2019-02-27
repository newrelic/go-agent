package newrelic

import (
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/internal"
)

func TestSpanEventSuccess(t *testing.T) {
	// Test that a basic segment creates a span event, and that a
	// transaction has a root span event.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	segment := StartSegment(txn, "mySegment")
	segment.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "OtherTransaction/Go/hello",
				"sampled":       true,
				"category":      "generic",
				"priority":      internal.MatchAnything,
				"guid":          internal.MatchAnything,
				"transactionId": internal.MatchAnything,
				"nr.entryPoint": true,
				"traceId":       internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":          "Custom/mySegment",
				"sampled":       true,
				"category":      "generic",
				"priority":      internal.MatchAnything,
				"guid":          internal.MatchAnything,
				"transactionId": internal.MatchAnything,
				"traceId":       internal.MatchAnything,
				"parentId":      internal.MatchAnything,
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
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
		cfg.SpanEvents.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	segment := StartSegment(txn, "mySegment")
	segment.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{})
}

func TestSpanEventsRemotelyDisabled(t *testing.T) {
	// Test that span events do not get created if the connect reply
	// disables span events.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
		reply.CollectSpanEvents = false
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	segment := StartSegment(txn, "mySegment")
	segment.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{})
}

func TestSpanEventsDisabledWithoutDistributedTracing(t *testing.T) {
	// Test that span events do not get created distributed tracing is not
	// enabled.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = false
		cfg.CrossApplicationTracer.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	segment := StartSegment(txn, "mySegment")
	segment.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{})
}

func TestSpanEventDatastoreExternal(t *testing.T) {
	// Test that a datastore and external segments creates the correct span
	// events.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	segment := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
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
				"name":          "OtherTransaction/Go/hello",
				"sampled":       true,
				"category":      "generic",
				"nr.entryPoint": true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
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
				"peer.address":  "myhost:myport",
				"peer.hostname": "myhost",
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"name":      "External/example.com/all",
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
	})
}

func TestSpanEventAttributesDisabled(t *testing.T) {
	// Test that SpanEvents.Attributes.Enabled correctly disables span
	// attributes.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
		cfg.SpanEvents.Attributes.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	segment := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
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
				"name":          "OtherTransaction/Go/hello",
				"sampled":       true,
				"category":      "generic",
				"nr.entryPoint": true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
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
				"name":      "External/example.com/all",
				"category":  "http",
				"component": "http",
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestSpanEventAttributesSpecificallyExcluded(t *testing.T) {
	// Test that SpanEvents.Attributes.Exclude excludes span attributes.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
		cfg.SpanEvents.Attributes.Exclude = []string{
			SpanAttributeDBStatement,
			SpanAttributeDBInstance,
			SpanAttributePeerAddress,
			SpanAttributePeerHostname,
			SpanAttributeHTTPURL,
			SpanAttributeHTTPMethod,
		}
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	segment := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
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
				"name":          "OtherTransaction/Go/hello",
				"sampled":       true,
				"category":      "generic",
				"nr.entryPoint": true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
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
				"name":      "External/example.com/all",
				"category":  "http",
				"component": "http",
				"span.kind": "client",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestSpanEventAttributesExcluded(t *testing.T) {
	// Test that Attributes.Exclude excludes span attributes.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
		cfg.Attributes.Exclude = []string{
			SpanAttributeDBStatement,
			SpanAttributeDBInstance,
			SpanAttributePeerAddress,
			SpanAttributePeerHostname,
			SpanAttributeHTTPURL,
			SpanAttributeHTTPMethod,
		}
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	segment := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
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
				"name":          "OtherTransaction/Go/hello",
				"sampled":       true,
				"category":      "generic",
				"nr.entryPoint": true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
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
				"name":      "External/example.com/all",
				"category":  "http",
				"component": "http",
				"span.kind": "client",
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
		reply.AdaptiveSampler = internal.SampleEverything{}
		reply.SecurityPolicies.RecordSQL.SetEnabled(false)
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	segment := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
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
				"name":          "OtherTransaction/Go/hello",
				"sampled":       true,
				"category":      "generic",
				"nr.entryPoint": true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
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
				"peer.address":  "myhost:myport",
				"peer.hostname": "myhost",
				"db.statement":  "'myoperation' on 'mycollection' using 'MySQL'",
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"parentId":  internal.MatchAnything,
				"name":      "External/example.com/all",
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
	})
}
