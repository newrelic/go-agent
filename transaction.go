package newrelic

import (
	"net/http"
	"net/url"
)

// Transaction represents a request or a background task.
// Each Transaction should only be used in a single goroutine.
type Transaction interface {
	// The transaction's http.ResponseWriter methods will delegate to the
	// http.ResponseWriter provided as a parameter to
	// Application.StartTransaction or Transaction.SetWebResponse. This
	// allows instrumentation of the response code and response headers.
	// These methods may still be called without panic if the transaction
	// does not have a http.ResponseWriter.
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

	// SetWebRequest marks the transaction as a web transaction.  If
	// WebRequest is non-nil, SetWebRequest will additionally collect
	// details on request attributes, url, and method.  If headers are
	// present, the agent will look for a distributed tracing header.  Use
	// NewWebRequest to transform a *http.Request into a WebRequest.
	SetWebRequest(WebRequest) error

	// SetWebResponse sets transaction's http.ResponseWriter.  After calling
	// this method, the transaction may be used in place of the
	// ResponseWriter to intercept the response code.  This method is useful
	// when the ResponseWriter is not available at the beginning of the
	// transaction (if so, it can be given as a parameter to
	// Application.StartTransaction).  This method will return a reference
	// to the transaction which implements the combination of
	// http.CloseNotifier, http.Flusher, http.Hijacker, and io.ReaderFrom
	// implemented by the ResponseWriter.
	SetWebResponse(http.ResponseWriter) Transaction

	// StartSegmentNow allows the timing of functions, external calls, and
	// datastore calls.  The segments of each transaction MUST be used in a
	// single goroutine.  Consumers are encouraged to use the
	// `StartSegmentNow` functions which checks if the Transaction is nil.
	// See segments.go
	StartSegmentNow() SegmentStartTime

	// CreateDistributedTracePayload creates a payload to link the calls
	// between transactions. This method never returns nil. Instead, it may
	// return a shim implementation whose methods return empty strings.
	// CreateDistributedTracePayload should be called every time an outbound
	// call is made since the payload contains a timestamp.
	//
	// StartExternalSegment calls CreateDistributedTracePayload, so you
	// should not need to use this method for typical outbound HTTP calls.
	// Just use StartExternalSegment!
	CreateDistributedTracePayload() DistributedTracePayload

	// AcceptDistributedTracePayload is used at the beginning of a
	// transaction to identify the caller.
	//
	// Application.StartTransaction calls this method automatically if a
	// payload is present in the request headers (under the key
	// DistributedTracePayloadHeader).  Therefore, this method does not need
	// to be used for typical HTTP transactions.
	//
	// AcceptDistributedTracePayload should be used as early in the
	// transaction as possible. It may not be called after a call to
	// CreateDistributedTracePayload.
	//
	// The payload parameter may be a DistributedTracePayload or a string.
	AcceptDistributedTracePayload(t TransportType, payload interface{}) error

	// Application returns the Application which started the transaction.
	Application() Application

	// BrowserTimingHeader generates the JavaScript required to enable
	// support for New Relic's Browser product. This should be placed as
	// high in the generated HTML as possible to generate the best timing
	// information: we suggest including it immediately after the opening
	// <head> tag and any <meta charset> tags.
	//
	// Note that calling this function has the side effect of freezing the
	// transaction name: any calls to SetName() after BrowserTimingHeader()
	// will be ignored.
	//
	// The *BrowserTimingHeader return value will be nil if browser
	// monitoring is disabled, the application is not connected, or an error
	// occurred.  It is safe to call the pointer's methods if it is nil.
	//
	// There is not a corresponding BrowserTimingFooter() function, as New
	// Relic's Browser support no longer requires a separate footer. The
	// naming is for consistency with other New Relic language agents.
	BrowserTimingHeader() (*BrowserTimingHeader, error)

	// NewGoroutine allows you to create segments in multiple goroutines.
	//
	// NewGoroutine returns a new reference to the Transaction.  This must
	// be called any time you are passing the Transaction to another
	// goroutine which makes segments.  Each segment-creating goroutine must
	// have its own Transaction reference.  It does not matter if you call
	// this before or after the other goroutine has started.
	//
	// Each Transaction reference has its own segment stack which assumes
	// synchronous behavior when creating metrics and traces.
	//
	// All Transaction methods can be used in any Transaction reference.
	// The Transaction will end when End() is called in any goroutine.
	//
	// Example passing a new Transaction reference directly to another
	// goroutine:
	//
	//	go func(txn newrelic.Transaction) {
	//		defer newrelic.StartSegment(txn, "async").End()
	//		time.Sleep(100 * time.Millisecond)
	//	}(txn.NewGoroutine())
	//
	// Example passing a new Transaction reference on a channel to another
	// goroutine:
	//
	//	ch := make(chan newrelic.Transaction)
	//	go func() {
	//		txn := <-ch
	//		defer newrelic.StartSegment(txn, "async").End()
	//		time.Sleep(100 * time.Millisecond)
	//	}()
	//	ch <- txn.NewGoroutine()
	//
	NewGoroutine() Transaction
}

// DistributedTracePayload is used to instrument connections between
// transactions and applications.
type DistributedTracePayload interface {
	// HTTPSafe serializes the payload into a string containing http safe
	// characters.
	HTTPSafe() string
	// Text serializes the payload into a string.  The format is slightly
	// more compact than HTTPSafe.
	Text() string
}

const (
	// DistributedTracePayloadHeader is the header used by New Relic agents
	// for automatic trace payload instrumentation.
	DistributedTracePayloadHeader = "Newrelic"
)

// TransportType represents the type of connection that the trace payload was
// transported over.
type TransportType struct{ name string }

// TransportType names used across New Relic agents:
var (
	TransportUnknown = TransportType{name: "Unknown"}
	TransportHTTP    = TransportType{name: "HTTP"}
	TransportHTTPS   = TransportType{name: "HTTPS"}
	TransportKafka   = TransportType{name: "Kafka"}
	TransportJMS     = TransportType{name: "JMS"}
	TransportIronMQ  = TransportType{name: "IronMQ"}
	TransportAMQP    = TransportType{name: "AMQP"}
	TransportQueue   = TransportType{name: "Queue"}
	TransportOther   = TransportType{name: "Other"}
)

// WebRequest may be implemented to provide request information to
// Transaction.SetWebRequest.
type WebRequest interface {
	// Header may return nil if you don't have any headers or don't want to
	// transform them to http.Header format.
	Header() http.Header
	// URL may return nil if you don't have a URL or don't want to transform
	// it to *url.URL.
	URL() *url.URL
	Method() string
	// If a distributed tracing header is found in the headers returned by
	// Header(), this TransportType will be used in the distributed tracing
	// metrics.
	Transport() TransportType
}

// NewWebRequest turns a *http.Request into a WebRequest for input into
// Transaction.SetWebRequest.
func NewWebRequest(request *http.Request) WebRequest {
	if nil == request {
		return nil
	}
	return requestWrap{request: request}
}
