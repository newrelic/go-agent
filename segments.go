package newrelic

import (
	"net/http"

	"github.com/newrelic/go-agent/datastore"
)

// SegmentStartTime is created by Transaction.StartSegment and marks the
// beginning of a segment.  A segment with a zero-valued SegmentStart may safely
// be ended.
type SegmentStartTime struct{ segment }

// Segment represents a function, method, or any block of code.
type Segment struct {
	Name      string
	StartTime SegmentStartTime
}

// DatastoreSegment represents a call to a database or object store.
type DatastoreSegment struct {
	// Product is the datastore type.  See the constants in
	// datastore/datastore.go.
	Product datastore.Product
	// Collection is the table or group.
	Collection string
	// Operation is the relevant action, e.g. "SELECT" or "GET".
	Operation string
	StartTime SegmentStartTime
}

// ExternalSegment represents an external call.
type ExternalSegment struct {
	Request  *http.Request
	Response *http.Response
	// URL should be populated if Request is not populated.  Populating
	// Request is recommended.
	URL       string
	StartTime SegmentStartTime
}

// End finishes the segment.
func (s Segment) End() { endSegment(s) }

// End finishes the datastore segment.
func (s DatastoreSegment) End() { endDatastore(s) }

// End finishes the external segment.
func (s ExternalSegment) End() { endExternal(s) }

// StartSegmentNow helps avoid Transaction nil checks.
func StartSegmentNow(txn Transaction) SegmentStartTime {
	if nil != txn {
		return txn.StartSegmentNow()
	}
	return SegmentStartTime{}
}

// StartSegment makes it easier to instrument basic segments.
//
//    defer newrelic.StartSegment(txn, "foo").End()
//
func StartSegment(txn Transaction, name string) Segment {
	return Segment{
		StartTime: StartSegmentNow(txn),
		Name:      name,
	}
}

// StartExternalSegment makes it easier to instrument external calls.
//
//    segment := newrelic.StartExternalSegment(txn, request)
//    defer segment.End()
//    resp, err := client.Do(request)
//    segment.Response = resp
//
func StartExternalSegment(txn Transaction, request *http.Request) ExternalSegment {
	return ExternalSegment{
		StartTime: StartSegmentNow(txn),
		Request:   request,
	}
}
