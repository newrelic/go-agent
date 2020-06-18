// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal"
)

func TestTraceSegments(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.SegmentThreshold = 0
		cfg.TransactionTracer.StackTraceThreshold = 0
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0

		// Disable span event attributes to ensure they are separate.
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Attributes.Enabled = false

	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	basicSegment := StartSegment(txn, "basic")
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSRegion, "west")
	basicSegment.End()
	datastoreSegment := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "mycollection",
		Operation:          "myoperation",
		ParameterizedQuery: "myquery",
		Host:               "myhost",
		PortPathOrID:       "myport",
		DatabaseName:       "dbname",
		QueryParameters:    map[string]interface{}{"zap": "zip"},
	}
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSRequestID, "123")
	datastoreSegment.End()
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	externalSegment := StartExternalSegment(txn, req)
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSOperation, "secret")
	externalSegment.End()
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/hello",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/hello",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "Custom/basic",
						Attributes: map[string]interface{}{
							"backtrace":  internal.MatchAnything,
							"aws.region": "west",
						},
					},
					{
						SegmentName: "Datastore/statement/MySQL/mycollection/myoperation",
						Attributes: map[string]interface{}{
							"backtrace":        internal.MatchAnything,
							"query_parameters": "map[zap:zip]",
							"peer.address":     "myhost:myport",
							"peer.hostname":    "myhost",
							"db.statement":     "myquery",
							"db.instance":      "dbname",
							"aws.requestId":    123,
						},
					},
					{
						SegmentName: "External/example.com/http/GET",
						Attributes: map[string]interface{}{
							"backtrace":     internal.MatchAnything,
							"http.url":      "http://example.com",
							"aws.operation": "secret",
						},
					},
				},
			}},
		},
	}})
}

func TestTraceSegmentsNoBacktrace(t *testing.T) {
	// Test that backtrace will only appear if the segment's duration
	// exceeds TransactionTracer.StackTraceThreshold.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.SegmentThreshold = 0
		cfg.TransactionTracer.StackTraceThreshold = 1 * time.Hour
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0

		// Disable span event attributes to ensure they are separate.
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Attributes.Enabled = false

	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	basicSegment := StartSegment(txn, "basic")
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSRegion, "west")
	basicSegment.End()
	datastoreSegment := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "mycollection",
		Operation:          "myoperation",
		ParameterizedQuery: "myquery",
		Host:               "myhost",
		PortPathOrID:       "myport",
		DatabaseName:       "dbname",
		QueryParameters:    map[string]interface{}{"zap": "zip"},
	}
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSRequestID, "123")
	datastoreSegment.End()
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	externalSegment := StartExternalSegment(txn, req)
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSOperation, "secret")
	externalSegment.End()
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/hello",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/hello",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "Custom/basic",
						Attributes: map[string]interface{}{
							"aws.region": "west",
						},
					},
					{
						SegmentName: "Datastore/statement/MySQL/mycollection/myoperation",
						Attributes: map[string]interface{}{
							"query_parameters": "map[zap:zip]",
							"peer.address":     "myhost:myport",
							"peer.hostname":    "myhost",
							"db.statement":     "myquery",
							"db.instance":      "dbname",
							"aws.requestId":    123,
						},
					},
					{
						SegmentName: "External/example.com/http/GET",
						Attributes: map[string]interface{}{
							"http.url":      "http://example.com",
							"aws.operation": "secret",
						},
					},
				},
			}},
		},
	}})
}

func TestTraceStacktraceServerSideConfig(t *testing.T) {
	// Test that the server-side-config stack trace threshold is observed.
	replyfn := func(reply *internal.ConnectReply) {
		json.Unmarshal([]byte(`{"agent_config":{"transaction_tracer.stack_trace_threshold":0}}`), reply)
	}
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.SegmentThreshold = 0
		cfg.TransactionTracer.StackTraceThreshold = 1 * time.Hour
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	basicSegment := StartSegment(txn, "basic")
	basicSegment.End()
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/hello",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/hello",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "Custom/basic",
						Attributes: map[string]interface{}{
							"backtrace": internal.MatchAnything,
						},
					},
				},
			}},
		},
	}})
}

func TestTraceSegmentAttributesExcluded(t *testing.T) {
	// Test that segment attributes can be excluded by Attributes.Exclude.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.SegmentThreshold = 0
		cfg.TransactionTracer.StackTraceThreshold = 1 * time.Hour
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
		cfg.Attributes.Exclude = []string{
			SpanAttributeDBStatement,
			SpanAttributeDBInstance,
			SpanAttributeDBCollection,
			SpanAttributePeerAddress,
			SpanAttributePeerHostname,
			SpanAttributeHTTPURL,
			SpanAttributeHTTPMethod,
			SpanAttributeAWSOperation,
			SpanAttributeAWSRequestID,
			SpanAttributeAWSRegion,
			"query_parameters",
		}

		// Disable span event attributes to ensure they are separate.
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Attributes.Enabled = false

	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	basicSegment := StartSegment(txn, "basic")
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSRegion, "west")
	basicSegment.End()
	datastoreSegment := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "mycollection",
		Operation:          "myoperation",
		ParameterizedQuery: "myquery",
		Host:               "myhost",
		PortPathOrID:       "myport",
		DatabaseName:       "dbname",
		QueryParameters:    map[string]interface{}{"zap": "zip"},
	}
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSRequestID, "123")
	datastoreSegment.End()
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	externalSegment := StartExternalSegment(txn, req)
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSOperation, "secret")
	externalSegment.End()
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/hello",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/hello",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "Custom/basic",
						Attributes:  map[string]interface{}{},
					},
					{
						SegmentName: "Datastore/statement/MySQL/mycollection/myoperation",
						Attributes:  map[string]interface{}{},
					},
					{
						SegmentName: "External/example.com/http/GET",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}

func TestTraceSegmentAttributesSpecificallyExcluded(t *testing.T) {
	// Test that segment attributes can be excluded by
	// TransactionTracer.Segments.Attributes.Exclude.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.SegmentThreshold = 0
		cfg.TransactionTracer.StackTraceThreshold = 1 * time.Hour
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
		cfg.TransactionTracer.Segments.Attributes.Exclude = []string{
			SpanAttributeDBStatement,
			SpanAttributeDBInstance,
			SpanAttributeDBCollection,
			SpanAttributePeerAddress,
			SpanAttributePeerHostname,
			SpanAttributeHTTPURL,
			SpanAttributeHTTPMethod,
			SpanAttributeAWSOperation,
			SpanAttributeAWSRequestID,
			SpanAttributeAWSRegion,
			"query_parameters",
		}

		// Disable span event attributes to ensure they are separate.
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Attributes.Enabled = false

	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	basicSegment := StartSegment(txn, "basic")
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSRegion, "west")
	basicSegment.End()
	datastoreSegment := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "mycollection",
		Operation:          "myoperation",
		ParameterizedQuery: "myquery",
		Host:               "myhost",
		PortPathOrID:       "myport",
		DatabaseName:       "dbname",
		QueryParameters:    map[string]interface{}{"zap": "zip"},
	}
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSRequestID, "123")
	datastoreSegment.End()
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	externalSegment := StartExternalSegment(txn, req)
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSOperation, "secret")
	externalSegment.End()
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/hello",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/hello",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "Custom/basic",
						Attributes:  map[string]interface{}{},
					},
					{
						SegmentName: "Datastore/statement/MySQL/mycollection/myoperation",
						Attributes:  map[string]interface{}{},
					},
					{
						SegmentName: "External/example.com/http/GET",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}

func TestTraceSegmentAttributesDisabled(t *testing.T) {
	// Test that segment attributes can be disabled by Attributes.Enabled
	// but backtrace and transaction_guid still appear.
	cfgfn := func(cfg *Config) {
		cfg.Attributes.Enabled = false
		cfg.TransactionTracer.SegmentThreshold = 0
		cfg.TransactionTracer.StackTraceThreshold = 0
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
	}
	app := testApp(crossProcessReplyFn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	basicSegment := StartSegment(txn, "basic")
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSRegion, "west")
	basicSegment.End()
	datastoreSegment := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "mycollection",
		Operation:          "myoperation",
		ParameterizedQuery: "myquery",
		Host:               "myhost",
		PortPathOrID:       "myport",
		DatabaseName:       "dbname",
		QueryParameters:    map[string]interface{}{"zap": "zip"},
	}
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSRequestID, "123")
	datastoreSegment.End()
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	externalSegment := StartExternalSegment(txn, req)
	externalSegment.Response = &http.Response{
		Header: outboundCrossProcessResponse(),
	}
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSOperation, "secret")
	externalSegment.End()
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/hello",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/hello",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "Custom/basic",
						Attributes: map[string]interface{}{
							"backtrace": internal.MatchAnything,
						},
					},
					{
						SegmentName: "Datastore/statement/MySQL/mycollection/myoperation",
						Attributes: map[string]interface{}{
							"backtrace": internal.MatchAnything,
						},
					},
					{
						SegmentName: "ExternalTransaction/example.com/12345#67890/WebTransaction/Go/txn",
						Attributes: map[string]interface{}{
							"backtrace":        internal.MatchAnything,
							"transaction_guid": internal.MatchAnything,
						},
					},
				},
			}},
		},
	}})
}

func TestTraceSegmentAttributesSpecificallyDisabled(t *testing.T) {
	// Test that segment attributes can be disabled by
	// TransactionTracer.Segments.Attributes.Enabled but backtrace and
	// transaction_guid still appear.
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.Segments.Attributes.Enabled = false
		cfg.TransactionTracer.SegmentThreshold = 0
		cfg.TransactionTracer.StackTraceThreshold = 0
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
	}
	app := testApp(crossProcessReplyFn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	basicSegment := StartSegment(txn, "basic")
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSRegion, "west")
	basicSegment.End()
	datastoreSegment := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "mycollection",
		Operation:          "myoperation",
		ParameterizedQuery: "myquery",
		Host:               "myhost",
		PortPathOrID:       "myport",
		DatabaseName:       "dbname",
		QueryParameters:    map[string]interface{}{"zap": "zip"},
	}
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSRequestID, "123")
	datastoreSegment.End()
	req, _ := http.NewRequest("GET", "http://example.com?ignore=me", nil)
	externalSegment := StartExternalSegment(txn, req)
	externalSegment.Response = &http.Response{
		Header: outboundCrossProcessResponse(),
	}
	internal.AddAgentSpanAttribute(txn, internal.SpanAttributeAWSOperation, "secret")
	externalSegment.End()
	txn.End()
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/hello",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/hello",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "Custom/basic",
						Attributes: map[string]interface{}{
							"backtrace": internal.MatchAnything,
						},
					},
					{
						SegmentName: "Datastore/statement/MySQL/mycollection/myoperation",
						Attributes: map[string]interface{}{
							"backtrace": internal.MatchAnything,
						},
					},
					{
						SegmentName: "ExternalTransaction/example.com/12345#67890/WebTransaction/Go/txn",
						Attributes: map[string]interface{}{
							"backtrace":        internal.MatchAnything,
							"transaction_guid": internal.MatchAnything,
						},
					},
				},
			}},
		},
	}})
}
