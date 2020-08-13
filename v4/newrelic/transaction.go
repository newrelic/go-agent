// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
)

// Transaction instruments one logical unit of work: either an inbound web
// request or background task.  Start a new Transaction with the
// Application.StartTransaction method.
//
// All methods on Transaction are nil safe. Therefore, a nil Transaction
// pointer can be safely used as a mock.
type Transaction struct {
	app      *Application
	rootSpan *span
	thread   *thread
	ended    bool
	name     string
}

type thread struct {
	sync.Mutex
	currentSpan *span
	// isMultiSpan is set to true of the associated transaction has more
	// than one span or crosses multiple goroutines.
	isMultiSpan bool
}

// End finishes the Transaction.  After that, subsequent calls to End or
// other Transaction methods have no effect.  All segments and
// instrumentation must be completed before End is called.
func (txn *Transaction) End() {
	if txn == nil {
		return
	}
	if txn.thread == nil {
		return
	}
	if txn.isEnded() {
		return
	}
	txn.rootSpan.end()
	txn.thread.Lock()
	txn.ended = true
	txn.thread.Unlock()
}

func (txn *Transaction) isEnded() bool {
	txn.thread.Lock()
	defer txn.thread.Unlock()
	return txn.ended
}

// Ignore prevents this transaction's data from being recorded.
func (txn *Transaction) Ignore() {}

// SetName names the transaction.  Use a limited set of unique names to
// ensure that Transactions are grouped usefully.
func (txn *Transaction) SetName(name string) {}

// NoticeError records an error.  The Transaction saves the first five
// errors.  For more control over the recorded error fields, see the
// newrelic.Error type.
//
// In certain situations, using this method may result in an error being
// recorded twice.  Errors are automatically recorded when
// Transaction.WriteHeader receives a status code at or above 400 or strictly
// below 100 that is not in the IgnoreStatusCodes configuration list.  This
// method is unaffected by the IgnoreStatusCodes configuration list.
//
// NoticeError examines whether the error implements the following optional
// methods:
//
//   // StackTrace records a stack trace
//   StackTrace() []uintptr
//
//   // ErrorClass sets the error's class
//   ErrorClass() string
//
//   // ErrorAttributes sets the errors attributes
//   ErrorAttributes() map[string]interface{}
//
// The newrelic.Error type, which implements these methods, is the recommended
// way to directly control the recorded error's message, class, stacktrace,
// and attributes.
func (txn *Transaction) NoticeError(err error) {}

// AddAttribute adds a key value pair to the transaction event, errors,
// and traces.
//
// The key must contain fewer than than 255 bytes.  The value must be a
// number, string, or boolean.
//
// For more information, see:
// https://docs.newrelic.com/docs/agents/manage-apm-agents/agent-metrics/collect-custom-attributes
func (txn *Transaction) AddAttribute(key string, value interface{}) {
	if txn == nil {
		return
	}
	if txn.rootSpan == nil {
		return
	}
	txn.rootSpan.Span.SetAttribute(key, value)
}

// SetWebRequestHTTP marks the transaction as a web transaction.  If
// the request is non-nil, SetWebRequestHTTP will additionally collect
// details on request attributes, url, and method.  If headers are
// present, the agent will look for distributed tracing headers using
// Transaction.AcceptDistributedTraceHeaders.
func (txn *Transaction) SetWebRequestHTTP(r *http.Request) {
	if r == nil {
		txn.SetWebRequest(WebRequest{})
		return
	}
	wr := WebRequest{
		Header:    r.Header,
		URL:       r.URL,
		Method:    r.Method,
		Transport: transport(r),
		Host:      r.Host,
	}
	txn.SetWebRequest(wr)
}

// SetWebRequest marks the transaction as a web transaction.  SetWebRequest
// additionally collects details on request attributes, url, and method if
// these fields are set.  If headers are present, the agent will look for
// distributed tracing headers using Transaction.AcceptDistributedTraceHeaders.
// Use Transaction.SetWebRequestHTTP if you have a *http.Request.
func (txn *Transaction) SetWebRequest(r WebRequest) {
	txn.AcceptDistributedTraceHeaders(r.Transport, r.Header)
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

type dummyResponseWriter struct{}

func (rw dummyResponseWriter) Header() http.Header         { return nil }
func (rw dummyResponseWriter) Write(b []byte) (int, error) { return 0, nil }
func (rw dummyResponseWriter) WriteHeader(code int)        {}

// SetWebResponse allows the Transaction to instrument response code and
// response headers.  Use the return value of this method in place of the input
// parameter http.ResponseWriter in your instrumentation.
//
// The returned http.ResponseWriter is safe to use even if the Transaction
// receiver is nil or has already been ended.
//
// The returned http.ResponseWriter implements the combination of
// http.CloseNotifier, http.Flusher, http.Hijacker, and io.ReaderFrom
// implemented by the input http.ResponseWriter.
//
// This method is used by WrapHandle, WrapHandleFunc, and most integration
// package middlewares.  Therefore, you probably want to use this only if you
// are writing your own instrumentation middleware.
func (txn *Transaction) SetWebResponse(w http.ResponseWriter) http.ResponseWriter {
	if w == nil {
		// Accepting a nil parameter makes it easy for consumers to add
		// a response code to the transaction without a response
		// writer:
		//
		//    txn.SetWebResponse(nil).WriteHeader(500)
		//
		w = dummyResponseWriter{}
	}
	return w
}

// StartSegmentNow starts timing a segment.  The SegmentStartTime returned can
// be used as the StartTime field in Segment, DatastoreSegment, or
// ExternalSegment.  The returned SegmentStartTime is safe to use even  when the
// Transaction receiver is nil.  In this case, the segment will have no effect.
func (txn *Transaction) StartSegmentNow() SegmentStartTime {
	if txn == nil {
		return SegmentStartTime{}
	}
	if txn.thread == nil {
		return SegmentStartTime{}
	}
	if txn.isEnded() {
		return SegmentStartTime{}
	}
	parent := txn.thread.getCurrentSpan()
	ctx, sp := txn.app.tracer.Start(parent.ctx, "",
		trace.WithSpanKind(trace.SpanKindInternal))
	span := &span{
		Span:   sp,
		ctx:    ctx,
		parent: parent,
		thread: txn.thread,
	}
	txn.thread.setCurrentSpan(span)
	return SegmentStartTime{
		span: span,
	}
}

func (thd *thread) setCurrentSpan(s *span) {
	thd.Lock()
	thd.currentSpan = s
	thd.isMultiSpan = true
	thd.Unlock()
}

func (thd *thread) getCurrentSpan() *span {
	if thd == nil {
		return nil
	}
	thd.Lock()
	defer thd.Unlock()
	return thd.currentSpan
}

// StartSegment makes it easy to instrument segments.  To time a function, do
// the following:
//
//	func timeMe(txn newrelic.Transaction) {
//		defer txn.StartSegment("timeMe").End()
//		// ... function code here ...
//	}
//
// To time a block of code, do the following:
//
//	segment := txn.StartSegment("myBlock")
//	// ... code you want to time here ...
//	segment.End()
func (txn *Transaction) StartSegment(name string) *Segment {
	return &Segment{
		StartTime: txn.StartSegmentNow(),
		Name:      name,
	}
}

// InsertDistributedTraceHeaders adds the Distributed Trace headers used to
// link transactions.  InsertDistributedTraceHeaders should be called every
// time an outbound call is made since the payload contains a timestamp.
//
// When the Distributed Tracer is enabled, InsertDistributedTraceHeaders will
// always insert W3C trace context headers.  It also by default inserts the New Relic
// distributed tracing header, but can be configured based on the
// Config.DistributedTracer.ExcludeNewRelicHeader option.
//
// StartExternalSegment calls InsertDistributedTraceHeaders, so you don't need
// to use it for outbound HTTP calls: Just use StartExternalSegment!
func (txn *Transaction) InsertDistributedTraceHeaders(hdrs http.Header) {
	if nil == txn || nil == txn.thread || nil == txn.app {
		return
	}

	currentSpan := txn.thread.getCurrentSpan()

	if nil == currentSpan {
		return
	}

	propagation.InjectHTTP(currentSpan.ctx, txn.app.propagators, hdrs)
}

// AcceptDistributedTraceHeaders links transactions by accepting distributed
// trace headers from another transaction.
//
// Transaction.SetWebRequest and Transaction.SetWebRequestHTTP both call this
// method automatically with the request headers.  Therefore, this method does
// not need to be used for typical HTTP transactions.
//
// AcceptDistributedTraceHeaders should be used as early in the transaction as
// possible.  It may not be called after a call to
// Transaction.InsertDistributedTraceHeaders.
//
// AcceptDistributedTraceHeaders first looks for the presence of W3C trace
// context headers.  Only when those are not found will it look for the New
// Relic distributed tracing header.
func (txn *Transaction) AcceptDistributedTraceHeaders(t TransportType, hdrs http.Header) {
	if nil == txn || nil == txn.thread || nil == txn.thread.currentSpan {
		return
	}

	txn.thread.currentSpan.Lock()
	defer txn.thread.currentSpan.Unlock()

	// Here we create a OpenTelemetry context that is detached from the
	// current trace. All segments (spans) subsequently started with this
	// context will be detached from the transaction trace, but rather will
	// have the remote trace id as trace id and the remote span id as the
	// parent span id.
	//
	// If no more than the root segment were yet created for this
	// transaction, we discard the root segment and replace it with a new
	// root segment that has the proper remote parent and trace id.
	remoteCtx := propagation.ExtractHTTP(context.Background(), txn.app.propagators, hdrs)

	if !txn.thread.isMultiSpan {
		ctx, sp := txn.app.tracer.Start(remoteCtx, txn.name)
		txn.rootSpan = &span{
			Span: sp,
			ctx:  ctx,
		}
		txn.thread.currentSpan = txn.rootSpan
	} else {
		txn.thread.currentSpan.ctx = remoteCtx
	}
}

// Application returns the Application which started the transaction.
func (txn *Transaction) Application() *Application {
	return txn.app
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
func (txn *Transaction) BrowserTimingHeader() *BrowserTimingHeader {
	return nil
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
// Note that any segments that end after the transaction ends will not
// be reported.
func (txn *Transaction) NewGoroutine() *Transaction {
	txn.thread.Lock()
	defer txn.thread.Unlock()

	txn.thread.isMultiSpan = true

	newTxn := *txn
	newTxn.thread = &thread{
		currentSpan: txn.thread.currentSpan,
		isMultiSpan: true,
	}
	return &newTxn
}

// GetTraceMetadata returns distributed tracing identifiers.  Empty
// string identifiers are returned if the transaction has finished.
func (txn *Transaction) GetTraceMetadata() TraceMetadata {
	return TraceMetadata{}
}

// GetLinkingMetadata returns the fields needed to link data to a trace or
// entity.
func (txn *Transaction) GetLinkingMetadata() LinkingMetadata {
	return LinkingMetadata{}
}

// IsSampled indicates if the Transaction is sampled.  A sampled
// Transaction records a span event for each segment.  Distributed tracing
// must be enabled for transactions to be sampled.  False is returned if
// the Transaction has finished.
func (txn *Transaction) IsSampled() bool {
	return false
}

const (
	// DistributedTraceNewRelicHeader is the header used by New Relic agents
	// for automatic trace payload instrumentation.
	DistributedTraceNewRelicHeader = "Newrelic"
	// DistributedTraceW3CTraceStateHeader is one of two headers used by W3C
	// trace context
	DistributedTraceW3CTraceStateHeader = "Tracestate"
	// DistributedTraceW3CTraceParentHeader is one of two headers used by W3C
	// trace context
	DistributedTraceW3CTraceParentHeader = "Traceparent"
)

// TransportType is used in Transaction.AcceptDistributedTraceHeaders to
// represent the type of connection that the trace payload was transported
// over.
type TransportType string

// TransportType names used across New Relic agents:
const (
	TransportUnknown TransportType = "Unknown"
	TransportHTTP    TransportType = "HTTP"
	TransportHTTPS   TransportType = "HTTPS"
	TransportKafka   TransportType = "Kafka"
	TransportJMS     TransportType = "JMS"
	TransportIronMQ  TransportType = "IronMQ"
	TransportAMQP    TransportType = "AMQP"
	TransportQueue   TransportType = "Queue"
	TransportOther   TransportType = "Other"
)

// WebRequest is used to provide request information to Transaction.SetWebRequest.
type WebRequest struct {
	// Header may be nil if you don't have any headers or don't want to
	// transform them to http.Header format.
	Header http.Header
	// URL may be nil if you don't have a URL or don't want to transform
	// it to *url.URL.
	URL *url.URL
	// Method is the request's method.
	Method string
	// If a distributed tracing header is found in the WebRequest.Header,
	// this TransportType will be used in the distributed tracing metrics.
	Transport TransportType
	// This is the value of the `Host` header. Go does not add it to the
	// http.Header object and so must be passed separately.
	Host string
}

// LinkingMetadata is returned by Transaction.GetLinkingMetadata.  It contains
// identifiers needed to link data to a trace or entity.
type LinkingMetadata struct {
	// TraceID identifies the entire distributed trace.  This field is empty
	// if distributed tracing is disabled.
	TraceID string
	// SpanID identifies the currently active segment.  This field is empty
	// if distributed tracing is disabled or the transaction is not sampled.
	SpanID string
	// EntityName is the Application name as set on the Config.  If multiple
	// application names are specified in the Config, only the first is
	// returned.
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
