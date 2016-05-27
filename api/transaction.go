package api

import "net/http"

type Transaction interface {
	// If StartTransaction is called with a non-nil http.ResponseWriter, the
	// Transaction itself may be used in its place.  Doing so will allow
	// future instrumentation of the response code and response headers.
	// These methods must not be called if the http.ResponseWriter parameter
	// to StartTransaction was nil.
	http.ResponseWriter

	// End finishes the current transaction, stopping all further
	// instrumentation.  Subsequent calls to End will have no effect. If End
	// is not called, the transaction is effectively ignored, and its data
	// will not be reported.
	End() error

	// SetName names the transaction.  Care should be taken to use a small
	// number of names:  If too many names are used, transactions will not
	// be grouped usefully.  This method will only work if called before
	// End, otherwise an error will be returned.
	SetName(name string) error

	// NoticeError records an error an associates it with the Transaction. A
	// stack trace is created for the error at the point at which this
	// method is called.  If NoticeError is called multiple times in the
	// same transaction, the first five errors are recorded (this behavior
	// is subject to potential change in the future).  This method will only
	// work if called before End, otherwise an error will be returned.
	NoticeError(err error) error

	// AddAttribute adds a key value pair tag to the current transaction.
	// This is information is attached to errors, transaction events, and
	// error events.  The key must be contain fewer than than 255 bytes.
	// The value must be a number, string, or boolean.  Attribute
	// configuration is applied (see config.go).
	//
	// For more information, see:
	// https://docs.newrelic.com/docs/agents/manage-apm-agents/agent-metrics/collect-custom-attributes
	AddAttribute(key string, value interface{}) error
}
