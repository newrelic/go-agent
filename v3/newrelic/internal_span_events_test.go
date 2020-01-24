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
		reply.SampleEverything()
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
				"name":          "OtherTransaction/Go/hello",
				"sampled":       true,
				"category":      "generic",
				"priority":      internal.MatchAnything,
				"guid":          "4259d74b863e2fba",
				"transactionId": "1ae969564b34a33e",
				"nr.entryPoint": true,
				"traceId":       "1ae969564b34a33ecd1af05fe6923d6d",
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
				"guid":          "e71870997d57214c",
				"transactionId": "1ae969564b34a33e",
				"traceId":       "1ae969564b34a33ecd1af05fe6923d6d",
				"parentId":      "4259d74b863e2fba",
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
		reply.SampleEverything()
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
		reply.SampleEverything()
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
		reply.SampleEverything()
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
		reply.SampleEverything()
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
	})
}

func TestSpanEventAttributesDisabled(t *testing.T) {
	// Test that SpanEvents.Attributes.Enabled correctly disables span
	// attributes.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SampleEverything()
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
				"name":      "External/example.com/http/GET",
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
		reply.SampleEverything()
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
				"name":      "External/example.com/http/GET",
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
		reply.SampleEverything()
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
				"name":      "External/example.com/http/GET",
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
		reply.SampleEverything()
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
	})
}

func TestAddAgentSpanAttribute(t *testing.T) {
	// Test that AddAgentSpanAttribute successfully adds attributes to
	// spans.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	s := txn.StartSegment("hi")
	internal.AddAgentSpanAttribute(txn.Private, internal.SpanAttributeAWSRegion, "west")
	internal.AddAgentSpanAttribute(txn.Private, internal.SpanAttributeAWSRequestID, "123")
	internal.AddAgentSpanAttribute(txn.Private, internal.SpanAttributeAWSOperation, "secret")
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
	})
}

func TestAddAgentSpanAttributeExcluded(t *testing.T) {
	// Test that span attributes added by AddAgentSpanAttribute are subject
	// to span attribute configuration.
	replyfn := func(reply *internal.ConnectReply) {
		reply.SampleEverything()
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
	internal.AddAgentSpanAttribute(txn.Private, internal.SpanAttributeAWSRegion, "west")
	internal.AddAgentSpanAttribute(txn.Private, internal.SpanAttributeAWSRequestID, "123")
	internal.AddAgentSpanAttribute(txn.Private, internal.SpanAttributeAWSOperation, "secret")
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
				"name":     "Custom/hi",
				"sampled":  true,
				"category": "generic",
				"parentId": internal.MatchAnything,
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
		reply.SampleEverything()
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello")
	// Do not panic if there are no active spans!
	internal.AddAgentSpanAttribute(txn.Private, internal.SpanAttributeAWSRegion, "west")
	internal.AddAgentSpanAttribute(txn.Private, internal.SpanAttributeAWSRequestID, "123")
	internal.AddAgentSpanAttribute(txn.Private, internal.SpanAttributeAWSOperation, "secret")
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
	})
}

func TestAddSpanAttributeNilTransaction(t *testing.T) {
	// Test that AddAgentSpanAttribute does not panic if the transaction is
	// nil.
	internal.AddAgentSpanAttribute(nil, internal.SpanAttributeAWSRegion, "west")
	internal.AddAgentSpanAttribute(nil, internal.SpanAttributeAWSRequestID, "123")
	internal.AddAgentSpanAttribute(nil, internal.SpanAttributeAWSOperation, "secret")
}
