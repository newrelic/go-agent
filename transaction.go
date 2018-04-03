package newrelic

import (
	"net/http"
	"net/url"
)

// Transaction represents a request or a background task.
// Each Transaction should only be used in a single goroutine.
type Transaction interface {
	// If StartTransaction is called with a non-nil http.ResponseWriter then
	// the Transaction may be used in its place.  This allows
	// instrumentation of the response code and response headers.
	http.ResponseWriter

	// End finishes the current transaction, stopping all further
	// instrumentation.  Subsequent calls to End will have no effect.
	End() error

	// Ignore ensures that this transaction's data will not be recorded.
	Ignore() error

	// SetName names the transaction.  Transactions will not be grouped
	// usefully if too many unique names are used.
	SetName(name string) error

	// NoticeError records an error.  The first five errors per transaction
	// are recorded (this behavior is subject to potential change in the
	// future).
	NoticeError(err error) error

	// AddAttribute adds a key value pair to the current transaction.  This
	// information is attached to errors, transaction events, and error
	// events.  The key must contain fewer than than 255 bytes.  The value
	// must be a number, string, or boolean.  Attribute configuration is
	// applied (see config.go).
	//
	// For more information, see:
	// https://docs.newrelic.com/docs/agents/manage-apm-agents/agent-metrics/collect-custom-attributes
	AddAttribute(key string, value interface{}) error

	// StartSegmentNow allows the timing of functions, external calls, and
	// datastore calls.  The segments of each transaction MUST be used in a
	// single goroutine.  Consumers are encouraged to use the
	// `StartSegmentNow` functions which checks if the Transaction is nil.
	// See segments.go
	StartSegmentNow() SegmentStartTime
}

// AdvancedTransaction represents a request or a background task.
// The same rules apply to a AdvancedTransaction as a Transaction.
// This is provided as a method for more advanced usage beyond simple
// http requests
type AdvancedTransaction interface {
	Transaction

	// SetWeb will convert the transaction to a web transaction for a given URL
	// in the normal transaction this is set to true when given a *http.Request
	// with the *url.URL being pulled from that request
	SetWeb(web bool, url *url.URL)

	// SetCrossProcess will configure the required CrossProcess infromation
	// normally extracted from the *http.Request.Headers
	SetCrossProcess(id, txnData, synthetics string)

	// SetResponseCode permits recording of the http response code (eg: 200)
	// which would normally be collected automatically by the standard transaction
	SetResponseCode(code int)
}
