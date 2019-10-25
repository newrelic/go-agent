package newrelic

import (
	"net/http"
	"net/url"
	"strings"
)

// Transaction instruments one logical unit of work: either an inbound web
// request or background task.  Start a new Transaction with the
// Application.StartTransaction() method.
type Transaction struct {
	Private interface{}
	thread  *thread
}

// End finishes the Transaction.  After that, subsequent calls to End or
// other Transaction methods have no effect.  All segments and
// instrumentation must be completed before End is called.
func (txn *Transaction) End() error {
	if nil == txn {
		return nil
	}
	if nil == txn.thread {
		return nil
	}

	// recover must be called in the function directly being deferred,
	// not any nested call!
	r := recover()
	return txn.thread.End(r)
}

// Ignore prevents this transaction's data from being recorded.
func (txn *Transaction) Ignore() error {
	if nil == txn {
		return nil
	}
	if nil == txn.thread {
		return nil
	}
	return txn.thread.Ignore()
}

// SetName names the transaction.  Use a limited set of unique names to
// ensure that Transactions are grouped usefully.
func (txn *Transaction) SetName(name string) error {
	if nil == txn {
		return nil
	}
	if nil == txn.thread {
		return nil
	}
	return txn.thread.SetName(name)
}

// NoticeError records an error.  The Transaction saves the first five
// errors.  For more control over the recorded error fields, see the
// newrelic.Error type.  In certain situations, using this method may
// result in an error being recorded twice:  Errors are automatically
// recorded when Transaction.WriteHeader receives a status code above
// 400 or below 100 that is not in the IgnoreStatusCodes configuration
// list.  This method is unaffected by the IgnoreStatusCodes
// configuration list.
func (txn *Transaction) NoticeError(err error) error {
	if nil == txn {
		return nil
	}
	if nil == txn.thread {
		return nil
	}
	return txn.thread.NoticeError(err)
}

// AddAttribute adds a key value pair to the transaction event, errors,
// and traces.
//
// The key must contain fewer than than 255 bytes.  The value must be a
// number, string, or boolean.
//
// For more information, see:
// https://docs.newrelic.com/docs/agents/manage-apm-agents/agent-metrics/collect-custom-attributes
func (txn *Transaction) AddAttribute(key string, value interface{}) error {
	if nil == txn {
		return nil
	}
	if nil == txn.thread {
		return nil
	}
	return txn.thread.AddAttribute(key, value)
}

// SetWebRequestHTTP marks the transaction as a web transaction.  If
// the request is non-nil, SetWebRequestHTTP will additionally collect
// details on request attributes, url, and method.  If headers are
// present, the agent will look for a distributed tracing header.
func (txn *Transaction) SetWebRequestHTTP(r *http.Request) error {
	if nil == r {
		return txn.SetWebRequest(nil)
	}
	wr := &WebRequest{
		Header:    r.Header,
		URL:       r.URL,
		Method:    r.Method,
		Transport: transport(r),
	}
	return txn.SetWebRequest(wr)
}

func transport(r *http.Request) TransportType {
	if strings.HasPrefix(r.Proto, "HTTP") {
		if r.TLS != nil {
			return TransportHTTPS
		}
		return TransportHTTP
	}
	return TransportUnknown
}

// SetWebRequest marks the transaction as a web transaction.  If
// WebRequest is non-nil, SetWebRequest will additionally collect
// details on request attributes, url, and method.  If headers are
// present, the agent will look for a distributed tracing header.  Use
// SetWebRequestHTTP if you have a *http.Request.
func (txn *Transaction) SetWebRequest(r *WebRequest) error {
	if nil == txn {
		return nil
	}
	if nil == txn.thread {
		return nil
	}
	return txn.thread.SetWebRequest(r)
}

// SetWebResponse sets transaction's http.ResponseWriter.  After calling
// this method, the transaction may be used in place of the
// ResponseWriter to intercept the response code.  This method is useful
// when the ResponseWriter is not available at the beginning of the
// transaction (if so, it can be given as a parameter to
// Application.StartTransaction).  This method will return a reference
// to the transaction which implements the combination of
// http.CloseNotifier, http.Flusher, http.Hijacker, and io.ReaderFrom
// implemented by the ResponseWriter.
func (txn *Transaction) SetWebResponse(w http.ResponseWriter) http.ResponseWriter {
	if nil == txn {
		return w
	}
	if nil == txn.thread {
		return w
	}
	return txn.thread.SetWebResponse(w)
}

// StartSegmentNow starts timing a segment.  The SegmentStartTime
// returned can be used as the StartTime field in Segment,
// DatastoreSegment, or ExternalSegment.  We recommend using the
// StartSegmentNow function instead of this method since it checks if
// the Transaction is nil.
func (txn *Transaction) StartSegmentNow() SegmentStartTime {
	if nil == txn {
		return SegmentStartTime{}
	}
	if nil == txn.thread {
		return SegmentStartTime{}
	}
	return txn.thread.StartSegmentNow()
}

// CreateDistributedTracePayload creates a payload used to link
// transactions.  CreateDistributedTracePayload should be called every
// time an outbound call is made since the payload contains a timestamp.
//
// StartExternalSegment calls CreateDistributedTracePayload, so you
// don't need to use it for outbound HTTP calls: Just use
// StartExternalSegment!
//
// This method never returns nil.  If the application is disabled or not
// yet connected then this method returns a shim implementation whose
// methods return empty strings.
func (txn *Transaction) CreateDistributedTracePayload() DistributedTracePayload {
	if nil == txn {
		return nil
	}
	if nil == txn.thread {
		return nil
	}
	return txn.thread.CreateDistributedTracePayload()
}

// AcceptDistributedTracePayload links transactions by accepting a
// distributed trace payload from another transaction.
//
// Application.StartTransaction calls this method automatically if a
// payload is present in the request headers.  Therefore, this method
// does not need to be used for typical HTTP transactions.
//
// AcceptDistributedTracePayload should be used as early in the
// transaction as possible.  It may not be called after a call to
// CreateDistributedTracePayload.
//
// The payload parameter may be a DistributedTracePayload, a string, or
// a []byte.
func (txn *Transaction) AcceptDistributedTracePayload(t TransportType, payload interface{}) error {
	if nil == txn {
		return nil
	}
	if nil == txn.thread {
		return nil
	}
	return txn.thread.AcceptDistributedTracePayload(t, payload)
}

// Application returns the Application which started the transaction.
func (txn *Transaction) Application() *Application {
	if nil == txn {
		return nil
	}
	if nil == txn.thread {
		return nil
	}
	return txn.thread.Application()
}

// BrowserTimingHeader generates the JavaScript required to enable New
// Relic's Browser product.  This code should be placed into your pages
// as close to the top of the <head> element as possible, but after any
// position-sensitive <meta> tags (for example, X-UA-Compatible or
// charset information).
//
// This function freezes the transaction name: any calls to SetName()
// after BrowserTimingHeader() will be ignored.
//
// The *BrowserTimingHeader return value will be nil if browser
// monitoring is disabled, the application is not connected, or an error
// occurred.  It is safe to call the pointer's methods if it is nil.
func (txn *Transaction) BrowserTimingHeader() (*BrowserTimingHeader, error) {
	if nil == txn {
		return nil, nil
	}
	if nil == txn.thread {
		return nil, nil
	}
	return txn.thread.BrowserTimingHeader()
}

// NewGoroutine allows you to use the Transaction in multiple
// goroutines.
//
// Each goroutine must have its own Transaction reference returned by
// NewGoroutine.  You must call NewGoroutine to get a new Transaction
// reference every time you wish to pass the Transaction to another
// goroutine. It does not matter if you call this before or after the
// other goroutine has started.
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
func (txn *Transaction) NewGoroutine() *Transaction {
	if nil == txn {
		return nil
	}
	if nil == txn.thread {
		return nil
	}
	return txn.thread.NewGoroutine()
}

// GetTraceMetadata returns distributed tracing identifiers.  Empty
// string identifiers are returned if the transaction has finished.
func (txn *Transaction) GetTraceMetadata() TraceMetadata {
	if nil == txn {
		return TraceMetadata{}
	}
	if nil == txn.thread {
		return TraceMetadata{}
	}
	return txn.thread.GetTraceMetadata()
}

// GetLinkingMetadata returns the fields needed to link data to a trace or
// entity.
func (txn *Transaction) GetLinkingMetadata() LinkingMetadata {
	if nil == txn {
		return LinkingMetadata{}
	}
	if nil == txn.thread {
		return LinkingMetadata{}
	}
	return txn.thread.GetLinkingMetadata()
}

// IsSampled indicates if the Transaction is sampled.  A sampled
// Transaction records a span event for each segment.  Distributed tracing
// must be enabled for transactions to be sampled.  False is returned if
// the transaction has finished.
func (txn *Transaction) IsSampled() bool {
	if nil == txn {
		return false
	}
	if nil == txn.thread {
		return false
	}
	return txn.thread.IsSampled()
}

// DistributedTracePayload traces requests between applications or processes.
// DistributedTracePayloads are automatically added to HTTP requests by
// StartExternalSegment, so you only need to use this if you are tracing through
// a message queue or another non-HTTP communication library.  The
// DistributedTracePayload may be marshalled in one of two formats: HTTPSafe or
// Text.  All New Relic agents can accept payloads in either format.
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

// TransportType is used in Transaction.AcceptDistributedTracePayload() to
// represent the type of connection that the trace payload was transported over.
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

// WebRequest is used to provide request information to Transaction.SetWebRequest.
type WebRequest struct {
	// Header may be nil if you don't have any headers or don't want to
	// transform them to http.Header format.
	Header http.Header
	// URL may be nil if you don't have a URL or don't want to transform
	// it to *url.URL.
	URL    *url.URL
	Method string
	// If a distributed tracing header is found in the WebRequest.Header,
	// this TransportType will be used in the distributed tracing metrics.
	Transport TransportType
}

// LinkingMetadata is returned by Transaction.GetLinkingMetadata.  It contains
// identifiers needed link data to a trace or entity.
type LinkingMetadata struct {
	// TraceID identifies the entire distributed trace.  This field is empty
	// if distributed tracing is disabled.
	TraceID string
	// SpanID identifies the currently active segment.  This field is empty
	// if distributed tracing is disabled or the transaction is not sampled.
	SpanID string
	// EntityName is the Application name as set on the newrelic.Config.  If
	// multiple application names are specified, only the first is returned.
	EntityName string
	// EntityType is the type of this entity and is always the string
	// "SERVICE".
	EntityType string
	// EntityGUID is the unique identifier for this entity.
	EntityGUID string
	// Hostname is the hostname this entity is running on.
	Hostname string
}

// TraceMetadata is returned by Transaction.GetTraceMetadata.  It contains
// distributed tracing identifiers.
type TraceMetadata struct {
	// TraceID identifies the entire distributed trace.  This field is empty
	// if distributed tracing is disabled.
	TraceID string
	// SpanID identifies the currently active segment.  This field is empty
	// if distributed tracing is disabled or the transaction is not sampled.
	SpanID string
}
