// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrmongo

import (
	"context"
	"testing"

	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/integrationsupport"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
)

var (
	connID       = "localhost:27017[-1]"
	reqID  int64 = 10
	raw, _       = bson.Marshal(bson.D{primitive.E{Key: "commName", Value: "collName"}, {Key: "$db", Value: "testing"}})
	ste          = &event.CommandStartedEvent{
		Command:      raw,
		DatabaseName: "testdb",
		CommandName:  "commName",
		RequestID:    reqID,
		ConnectionID: connID,
	}
	finishedEvent = event.CommandFinishedEvent{
		DurationNanos: 5,
		CommandName:   "name",
		RequestID:     reqID,
		ConnectionID:  connID,
	}
	se = &event.CommandSucceededEvent{
		CommandFinishedEvent: finishedEvent,
		Reply:                nil,
	}
	fe = &event.CommandFailedEvent{
		CommandFinishedEvent: finishedEvent,
		Failure:              "failureCause",
	}
)

func TestOrigMonitorsAreCalled(t *testing.T) {
	var started, succeeded, failed bool
	origMonitor := &event.CommandMonitor{
		Started:   func(ctx context.Context, e *event.CommandStartedEvent) { started = true },
		Succeeded: func(ctx context.Context, e *event.CommandSucceededEvent) { succeeded = true },
		Failed:    func(ctx context.Context, e *event.CommandFailedEvent) { failed = true },
	}
	ctx := context.Background()
	nrMonitor := NewCommandMonitor(origMonitor)

	nrMonitor.Started(ctx, ste)
	if !started {
		t.Error("started not called")
	}
	nrMonitor.Succeeded(ctx, se)
	if !succeeded {
		t.Error("succeeded not called")
	}
	nrMonitor.Failed(ctx, fe)
	if !failed {
		t.Error("failed not called")
	}
}

func TestClientOptsWithNullFunctions(t *testing.T) {
	origMonitor := &event.CommandMonitor{} // the monitor isn't nil, but its functions are.
	ctx := context.Background()
	nrMonitor := NewCommandMonitor(origMonitor)

	// Verifying no nil pointer dereferences
	nrMonitor.Started(ctx, ste)
	nrMonitor.Succeeded(ctx, se)
	nrMonitor.Failed(ctx, fe)
}

func TestHostAndPort(t *testing.T) {
	type hostAndPort struct {
		host string
		port string
	}
	testCases := map[string]hostAndPort{
		"localhost:8080[-1]":                     {host: "localhost", port: "8080"},
		"something.com:987[-789]":                {host: "something.com", port: "987"},
		"thisformatiswrong":                      {host: "", port: ""},
		"somethingunix.sock[-876]":               {host: "somethingunix.sock", port: ""},
		"/var/dir/path/somethingunix.sock[-876]": {host: "/var/dir/path/somethingunix.sock", port: ""},
	}
	for test, expected := range testCases {
		h, p := calcHostAndPort(test)
		if expected.host != h {
			t.Errorf("unexpected host - expected %s, got %s", expected.host, h)
		}
		if expected.port != p {
			t.Errorf("unexpected port - expected %s, got %s", expected.port, p)
		}
	}
}

func TestMonitor(t *testing.T) {
	var started, succeeded, failed bool
	origMonitor := &event.CommandMonitor{
		Started:   func(ctx context.Context, e *event.CommandStartedEvent) { started = true },
		Succeeded: func(ctx context.Context, e *event.CommandSucceededEvent) { succeeded = true },
		Failed:    func(ctx context.Context, e *event.CommandFailedEvent) { failed = true },
	}
	nrMonitor := mongoMonitor{
		segmentMap:  make(map[int64]*newrelic.DatastoreSegment),
		origCommMon: origMonitor,
	}
	app := createTestApp()
	txn := app.StartTransaction("txnName", nil, nil)
	ctx := newrelic.NewContext(context.Background(), txn)
	nrMonitor.started(ctx, ste)
	if !started {
		t.Error("Original monitor not started")
	}
	if len(nrMonitor.segmentMap) != 1 {
		t.Errorf("Wrong number of segments, expected 1 but got %d", len(nrMonitor.segmentMap))
	}
	nrMonitor.succeeded(ctx, se)
	if !succeeded {
		t.Error("Original monitor not succeeded")
	}
	if len(nrMonitor.segmentMap) != 0 {
		t.Errorf("Wrong number of segments, expected 0 but got %d", len(nrMonitor.segmentMap))
	}
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransactionTotalTime/Go/txnName", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/instance/MongoDB/" + internal.ThisHost + "/27017", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/operation/MongoDB/commName", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/txnName", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allOther", Scope: "", Forced: true, Data: []float64{1.0}},
		{Name: "Datastore/MongoDB/all", Scope: "", Forced: true, Data: []float64{1.0}},
		{Name: "Datastore/MongoDB/allOther", Scope: "", Forced: true, Data: []float64{1.0}},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MongoDB/collName/commName", Scope: "", Forced: false, Data: []float64{1.0}},
		{Name: "Datastore/statement/MongoDB/collName/commName", Scope: "OtherTransaction/Go/txnName", Forced: false, Data: []float64{1.0}},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "OtherTransaction/Go/txnName",
				"sampled":       true,
				"category":      "generic",
				"nr.entryPoint": true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":      "Datastore/statement/MongoDB/collName/commName",
				"sampled":   true,
				"category":  "datastore",
				"component": "MongoDB",
				"span.kind": "client",
				"parentId":  internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"peer.address":  internal.ThisHost + ":27017",
				"peer.hostname": internal.ThisHost,
				"db.statement":  "'commName' on 'collName' using 'MongoDB'",
				"db.instance":   "testdb",
				"db.collection": "collName",
			},
		},
	})

	txn = app.StartTransaction("txnName", nil, nil)
	ctx = newrelic.NewContext(context.Background(), txn)
	nrMonitor.started(ctx, ste)
	if len(nrMonitor.segmentMap) != 1 {
		t.Errorf("Wrong number of segments, expected 1 but got %d", len(nrMonitor.segmentMap))
	}
	nrMonitor.failed(ctx, fe)
	if !failed {
		t.Error("Original monitor not succeeded")
	}
	if len(nrMonitor.segmentMap) != 0 {
		t.Errorf("Wrong number of segments, expected 0 but got %d", len(nrMonitor.segmentMap))
	}
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransactionTotalTime/Go/txnName", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/instance/MongoDB/" + internal.ThisHost + "/27017", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/operation/MongoDB/commName", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/txnName", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allOther", Scope: "", Forced: true, Data: []float64{2.0}},
		{Name: "Datastore/MongoDB/all", Scope: "", Forced: true, Data: []float64{2.0}},
		{Name: "Datastore/MongoDB/allOther", Scope: "", Forced: true, Data: []float64{2.0}},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MongoDB/collName/commName", Scope: "", Forced: false, Data: []float64{2.0}},
		{Name: "Datastore/statement/MongoDB/collName/commName", Scope: "OtherTransaction/Go/txnName", Forced: false, Data: []float64{2.0}},
	})
}

func TestCollName(t *testing.T) {
	command := "find"
	ex1, _ := bson.Marshal(bson.D{{Key: command, Value: "numbers"}, {Key: "$db", Value: "testing"}})
	ex2, _ := bson.Marshal(bson.D{{Key: "filter", Value: ""}})
	testCases := map[string]bson.Raw{
		"numbers": ex1,
		"":        ex2,
	}
	for name, raw := range testCases {
		e := event.CommandStartedEvent{
			Command:     raw,
			CommandName: command,
		}
		result := collName(&e)
		if result != name {
			t.Errorf("Wrong collection name: %s", result)
		}
	}

}

func createTestApp() integrationsupport.ExpectApp {
	return integrationsupport.NewTestApp(replyFn, cfgFn)
}

var cfgFn = func(cfg *newrelic.Config) {
	cfg.Enabled = false
	cfg.DistributedTracer.Enabled = true
	cfg.TransactionTracer.SegmentThreshold = 0
	cfg.TransactionTracer.Threshold.IsApdexFailing = false
	cfg.TransactionTracer.Threshold.Duration = 0
}

var replyFn = func(reply *internal.ConnectReply) {
	reply.AdaptiveSampler = internal.SampleEverything{}
}
