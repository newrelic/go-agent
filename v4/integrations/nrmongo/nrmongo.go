// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrmongo instruments https://github.com/mongodb/mongo-go-driver
//
// Use this package to instrument your MongoDB calls without having to manually
// create DatastoreSegments.  To do so, first set the monitor in the connect
// options using `SetMonitor`
// (https://godoc.org/go.mongodb.org/mongo-driver/mongo/options#ClientOptions.SetMonitor):
//
//	nrMon := nrmongo.NewCommandMonitor(nil)
//	client, err := mongo.Connect(ctx, options.Client().SetMonitor(nrMon))
//
// Note that it is important that this `nrmongo` monitor is the last monitor
// set, otherwise it will be overwritten.  If needing to use more than one
// `event.CommandMonitor`, pass the original monitor to the
// `nrmongo.NewCommandMonitor` function:
//
//	origMon := &event.CommandMonitor{
//		Started:   origStarted,
//		Succeeded: origSucceeded,
//		Failed:    origFailed,
//	}
//	nrMon := nrmongo.NewCommandMonitor(origMon)
//	client, err := mongo.Connect(ctx, options.Client().SetMonitor(nrMon))
//
// Then add the current transaction to the context used in any MongoDB call:
//
//	ctx = newrelic.NewContext(context.Background(), txn)
//	resp, err := collection.InsertOne(ctx, bson.M{"name": "pi", "value": 3.14159})
package nrmongo

import (
	"context"
	"regexp"
	"sync"

	"github.com/newrelic/go-agent/v4/internal"
	"github.com/newrelic/go-agent/v4/newrelic"
	"go.mongodb.org/mongo-driver/event"
)

func init() { internal.TrackUsage("integration", "datastore", "mongo") }

type mongoMonitor struct {
	segmentMap  map[int64]*newrelic.DatastoreSegment
	origCommMon *event.CommandMonitor
	sync.Mutex
}

// The Mongo connection ID is constructed as: `fmt.Sprintf("%s[-%d]", addr, nextConnectionID())`,
// where addr is of the form `host:port` (or `a.sock` for unix sockets)
// See https://github.com/mongodb/mongo-go-driver/blob/b39cd78ce7021252efee2fb44aa6e492d67680ef/x/mongo/driver/topology/connection.go#L68
// and https://github.com/mongodb/mongo-go-driver/blob/b39cd78ce7021252efee2fb44aa6e492d67680ef/x/mongo/driver/address/addr.go
var connIDPattern = regexp.MustCompile(`([^:\[]+)(?::(\d+))?\[-\d+]`)

// NewCommandMonitor returns a new `*event.CommandMonitor`
// (https://godoc.org/go.mongodb.org/mongo-driver/event#CommandMonitor).  If
// provided, the original `*event.CommandMonitor` will be called as well.  The
// returned `*event.CommandMonitor` creates `newrelic.DatastoreSegment`s
// (https://godoc.org/github.com/newrelic/go-agent#DatastoreSegment) for each
// database call.
//
//	// Use `SetMonitor` to register the CommandMonitor.
//	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017").SetMonitor(nrmongo.NewCommandMonitor(nil)))
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Add transaction to the context.  This step is required.
//	ctx = newrelic.NewContext(ctx, txn)
//
//	collection := client.Database("testing").Collection("numbers")
//	resp, err := collection.InsertOne(ctx, bson.M{"name": "pi", "value": 3.14159})
//	if err != nil {
//		log.Fatal(err)
//	}
func NewCommandMonitor(original *event.CommandMonitor) *event.CommandMonitor {
	m := mongoMonitor{
		segmentMap:  make(map[int64]*newrelic.DatastoreSegment),
		origCommMon: original,
	}
	return &event.CommandMonitor{
		Started:   m.started,
		Succeeded: m.succeeded,
		Failed:    m.failed,
	}
}

func (m *mongoMonitor) started(ctx context.Context, e *event.CommandStartedEvent) {
	if m.origCommMon != nil && m.origCommMon.Started != nil {
		m.origCommMon.Started(ctx, e)
	}
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return
	}
	host, port := calcHostAndPort(e.ConnectionID)
	sgmt := newrelic.DatastoreSegment{
		StartTime:    txn.StartSegmentNow(),
		Product:      newrelic.DatastoreMongoDB,
		Collection:   collName(e),
		Operation:    e.CommandName,
		Host:         host,
		PortPathOrID: port,
		DatabaseName: e.DatabaseName,
	}
	m.addSgmt(e, &sgmt)
}

func collName(e *event.CommandStartedEvent) string {
	coll := e.Command.Lookup(e.CommandName)
	collName, _ := coll.StringValueOK()
	return collName
}

func (m *mongoMonitor) addSgmt(e *event.CommandStartedEvent, sgmt *newrelic.DatastoreSegment) {
	m.Lock()
	defer m.Unlock()
	m.segmentMap[e.RequestID] = sgmt
}

func (m *mongoMonitor) succeeded(ctx context.Context, e *event.CommandSucceededEvent) {
	m.endSgmtIfExists(e.RequestID)
	if m.origCommMon != nil && m.origCommMon.Succeeded != nil {
		m.origCommMon.Succeeded(ctx, e)
	}
}

func (m *mongoMonitor) failed(ctx context.Context, e *event.CommandFailedEvent) {
	m.endSgmtIfExists(e.RequestID)
	if m.origCommMon != nil && m.origCommMon.Failed != nil {
		m.origCommMon.Failed(ctx, e)
	}
}

func (m *mongoMonitor) endSgmtIfExists(id int64) {
	m.getAndRemoveSgmt(id).End()
}

func (m *mongoMonitor) getAndRemoveSgmt(id int64) *newrelic.DatastoreSegment {
	m.Lock()
	defer m.Unlock()
	sgmt := m.segmentMap[id]
	if sgmt != nil {
		delete(m.segmentMap, id)
	}
	return sgmt
}

func calcHostAndPort(connID string) (host string, port string) {
	// FindStringSubmatch either returns nil or an array of the size # of submatches + 1 (in this case 3)
	addressParts := connIDPattern.FindStringSubmatch(connID)
	if len(addressParts) == 3 {
		host = addressParts[1]
		port = addressParts[2]
	}
	return
}
