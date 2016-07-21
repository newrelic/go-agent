package newrelic

import (
	"net/http"

	"github.com/newrelic/go-agent/datastore"
)

// SegmentTracer times blocks of code.  It is embedded into Transaction.
// It must be used in a single goroutine.
type SegmentTracer interface {
	// StartSegment begins timing a segment and returns an identification
	// token.  Pass this token to an end method to finish timing the
	// segment.
	StartSegment() Token

	// EndSegment finishes timing a basic segment.  The segment can be a
	// method, function, or any arbitrary block of code.  Typically, this
	// method is used on function entry with a defer:
	//
	//  defer txn.EndSegment(txn.StartSegment(), "myFunction")
	//
	// You may avoid the cost of defer if performance is critical.  When
	// using this pattern, note that a segment will not be recorded if a
	// panic occurs between StartSegment and EndSegment.
	//
	//  token := txn.StartSegment()
	//  // do the work
	//  txn.EndSegment()
	EndSegment(token Token, name string)

	// EndExternal should be used in place of EndSegment when the segment
	// represents an external request.  The host will be parsed out of the
	// url and used in the metrics created.  The host will appear as
	// "unknown" if the url is an empty string or cannot be parsed.
	//
	//  defer txn.EndExternal(txn.StartSegment(), "http://example.com")
	EndExternal(token Token, url string)

	// EndDatastore should be used in place of EndSegment when the segment
	// represents a call to a database or object store.  See the datastore
	// subpackage for documentation of the segment parameters.
	//
	//  defer txn.EndDatastore(txn.StartSegment(), datastore.Segment{
	//  	Product:    datastore.MySQL,
	//  	Collection: "my_table",
	//  	Operation:  "SELECT",
	//  })
	EndDatastore(Token, datastore.Segment)

	// PrepareRequest should be used before an external request is sent.
	// This will be used to add context headers in the future.
	PrepareRequest(token Token, request *http.Request)

	// EndRequest is recommended in place of EndExternal when a request and
	// response are available.
	EndRequest(token Token, request *http.Request, response *http.Response)
}

// Token is used to track segment tracing.  It must not be modified.
type Token uint64
