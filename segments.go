package newrelic

import (
	"net/http"
	"time"

	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/cat"
)

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
	// ParameterizedQuery may be set to the query being performed.  It must
	// not contain any raw parameters, only placeholders.
	ParameterizedQuery string
	// QueryParameters may be used to provide query parameters.  Care should
	// be taken to only provide parameters which are not sensitive.
	// QueryParameters are ignored in high security mode.
	QueryParameters map[string]interface{}
	// Host is the name of the server hosting the datastore.
	Host string
	// PortPathOrID can represent either the port, path, or id of the
	// datastore being connected to.
	PortPathOrID string
	// DatabaseName is name of database where the current query is being
	// executed.
	DatabaseName string
}

// ExternalSegment is used to instrument external calls.  StartExternalSegment
// is recommended when you have access to an http.Request.
type ExternalSegment struct {
	StartTime SegmentStartTime
	Request   *http.Request
	Response  *http.Response
	// If you do not have access to the request, this URL field should be
	// used to indicate the endpoint.  NOTE: If non-empty, this field
	// is parsed using url.Parse and therefore it MUST include the protocol
	// (eg. "http://").
	URL string
}

type AdvancedExternalSegment struct {
	StartTime SegmentStartTime
	cat.AppDataHeader
	internal.CrossProcessMetadata
	startTime time.Time
	URL       string
}

// End finishes the segment.
func (s Segment) End() error { return endSegment(s) }

// End finishes the datastore segment.
func (s DatastoreSegment) End() error { return endDatastore(s) }

// End finishes the external segment.
func (s ExternalSegment) End() error { return endExternal(s) }

// End finishes the custom external segment.
func (s AdvancedExternalSegment) End() error {
	if s.AppDataHeader.ResponseTimeInSeconds == 0.0 {
		s.AppDataHeader.ResponseTimeInSeconds = float64(time.Since(s.startTime)) / float64(time.Second)
	}
	return endAdvancedExternal(s)
}

// RestartTiming restats the start time of the external request to improve accuracy
func (s *AdvancedExternalSegment) RestartTiming() {
	s.startTime = time.Now()
}

// Import will import the appdata normally read from X-Newrelic-App-Data
func (s *AdvancedExternalSegment) Import(appdata string) error {
	cat, err := s.StartTime.txn.TxnData.CrossProcess.ParseAppData(appdata)
	if err != nil {
		return err
	}
	s.AppDataHeader = *cat
	return nil
}

// OutboundHeaders returns the headers that should be attached to the external
// request.
func (s ExternalSegment) OutboundHeaders() http.Header {
	return outboundHeaders(s)
}

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
	s := ExternalSegment{
		StartTime: StartSegmentNow(txn),
		Request:   request,
	}

	for key, values := range s.OutboundHeaders() {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}

	return s
}

// StartAdvancedExternalSegment makes it possible to instrument external calls that aren't http based
//
//    segment := newrelic.StartAdvancedExternalSegment(txn)
//    // transmit segment.Metadata to the remote service
//    // if possible call segment.Import(appdata) for most accuracy
//    segment.End()
//
func StartAdvancedExternalSegment(ctxn AdvancedTransaction, name, url string) AdvancedExternalSegment {
	txn := ctxn.(wrap)
	metadata, err := txn.CrossProcess.CreateCrossProcessMetadata(txn.Name, txn.Config.AppName)
	if err != nil {
		txn.Config.Logger.Debug("error generating outbound headers", map[string]interface{}{
			"error": err,
		})
	}

	s := AdvancedExternalSegment{
		StartTime: StartSegmentNow(ctxn),
		AppDataHeader: cat.AppDataHeader{
			CrossProcessID:  string(txn.CrossProcess.CrossProcessID),
			TransactionName: name,
			TransactionGUID: txn.CrossProcess.GUID,
		},
		CrossProcessMetadata: metadata,
		startTime:            time.Now(),
		URL:                  url,
	}

	return s
}
