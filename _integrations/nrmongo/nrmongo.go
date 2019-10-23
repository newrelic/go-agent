package nrmongo

import (
	"context"
	"regexp"
	"sync"

	newrelic "github.com/newrelic/go-agent"
	"go.mongodb.org/mongo-driver/event"
)

// mongoMonitor TODO: doc
// This does not currently expire anything from the maps - if we miss CommandMonitor succeeded/failed events
// the maps could start to get big. May need to change this to store a segment and a time, and periodically
// remove old segments.
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

// NewCommandMonitor TODO: doc
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
		StartTime:    newrelic.StartSegmentNow(txn),
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
