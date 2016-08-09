package newrelic

import "net/http"

// SegmentStartTime is created by Transaction.StartSegmentNow and marks the
// beginning of a segment.  A segment with a zero-valued SegmentStartTime may
// safely be ended.
type SegmentStartTime struct{ segment }

// Segment is used to instrument functions, methods, and blocks of code.  The
// easiest way use Segment is the StartSegment function.
type Segment struct {
	StartTime SegmentStartTime
	Name      string
}

// DatastoreSegment is used to instrument calls to databases and object stores.
// Here is an example:
//
// 	defer newrelic.DatastoreSegment{
// 		StartTime:  newrelic.StartSegmentNow(txn),
// 		Product:    newrelic.DatastoreMySQL,
// 		Collection: "my_table",
// 		Operation:  "SELECT",
// 	}.End()
//
type DatastoreSegment struct {
	StartTime SegmentStartTime
	// Product is the datastore type.  See the constants in datastore.go.
	Product DatastoreProduct
	// Collection is the table or group.
	Collection string
	// Operation is the relevant action, e.g. "SELECT" or "GET".
	Operation string
}

// ExternalSegment is used to instrument external calls.  StartExternalSegment
// is recommended when you have access to an http.Request.
type ExternalSegment struct {
	StartTime SegmentStartTime
	Request   *http.Request
	Response  *http.Response
	// If you do not have access to the request, this URL field should be
	// used to indicate the endpoint.
	URL string
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

// StartSegment makes it easy to instrument segments.  To time a function, do
// the following:
//
//	func timeMe(txn newrelic.Transaction) {
//		defer newrelic.StartSegment(txn, "timeMe").End()
//		// ... function code here ...
//	}
//
// To time a block of code, do the following:
//
//	segment := StartSegment(txn, "myBlock")
//	// ... code you want to time here ...
//	segment.End()
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
//    resp, err := client.Do(request)
//    segment.Response = resp
//    segment.End()
//
func StartExternalSegment(txn Transaction, request *http.Request) ExternalSegment {
	return ExternalSegment{
		StartTime: StartSegmentNow(txn),
		Request:   request,
	}
}
