// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/standard"
	"go.opentelemetry.io/otel/api/trace"
	"google.golang.org/grpc/codes"
)

type span struct {
	sync.Mutex
	Span   trace.Span
	ctx    context.Context
	parent *span
	ended  bool
	thread *thread
}

// SegmentStartTime is created by Transaction.StartSegmentNow and marks the
// beginning of a segment.  A segment with a zero-valued SegmentStartTime may
// safely be ended.
type SegmentStartTime struct {
	*span
}

// Segment is used to instrument functions, methods, and blocks of code.  The
// easiest way use Segment is the Transaction.StartSegment method.
type Segment struct {
	StartTime SegmentStartTime
	Name      string
}

// DatastoreSegment is used to instrument calls to databases and object stores.
type DatastoreSegment struct {
	// StartTime should be assigned using Transaction.StartSegmentNow before
	// each datastore call is made.
	StartTime SegmentStartTime

	// Product, Collection, and Operation are highly recommended as they are
	// used for aggregate metrics:
	//
	// Product is the datastore type.  See the constants in
	// https://github.com/newrelic/go-agent/blob/master/datastore.go.  Product
	// is one of the fields primarily responsible for the grouping of Datastore
	// metrics.
	Product DatastoreProduct
	// Collection is the table or group being operated upon in the datastore,
	// e.g. "users_table".  This becomes the db.collection attribute on Span
	// events and Transaction Trace segments.  Collection is one of the fields
	// primarily responsible for the grouping of Datastore metrics.
	Collection string
	// Operation is the relevant action, e.g. "SELECT" or "GET".  Operation is
	// one of the fields primarily responsible for the grouping of Datastore
	// metrics.
	Operation string

	// The following fields are used for extra metrics and added to instance
	// data:
	//
	// ParameterizedQuery may be set to the query being performed.  It must
	// not contain any raw parameters, only placeholders.
	ParameterizedQuery string
	// QueryParameters may be used to provide query parameters.  Care should
	// be taken to only provide parameters which are not sensitive.
	// QueryParameters are ignored in high security mode. The keys must contain
	// fewer than than 255 bytes.  The values must be numbers, strings, or
	// booleans.
	QueryParameters map[string]interface{}
	// Host is the name of the server hosting the datastore.
	Host string
	// PortPathOrID can represent either the port, path, or id of the
	// datastore being connected to.
	PortPathOrID string
	// DatabaseName is name of database instance where the current query is
	// being executed.  This becomes the db.instance attribute on Span events
	// and Transaction Trace segments.
	DatabaseName string
}

// ExternalSegment instruments external calls.  StartExternalSegment is the
// recommended way to create ExternalSegments.
type ExternalSegment struct {
	StartTime SegmentStartTime
	Request   *http.Request
	Response  *http.Response

	// URL is an optional field which can be populated in lieu of Request if
	// you don't have an http.Request.  Either URL or Request must be
	// populated.  If both are populated then Request information takes
	// priority.  URL is parsed using url.Parse so it must include the
	// protocol scheme (eg. "http://").
	URL string
	// Host is an optional field that is automatically populated from the
	// Request or URL.  It is used for external metrics, transaction trace
	// segment names, and span event names.  Use this field to override the
	// host in the URL or Request.  This field does not override the host in
	// the "http.url" attribute.
	Host string
	// Procedure is an optional field that can be set to the remote
	// procedure being called.  If set, this value will be used in metrics,
	// transaction trace segment names, and span event names.  If unset, the
	// request's http method is used.
	Procedure string
	// Library is an optional field that defaults to "http".  It is used for
	// external metrics and the "component" span attribute.  It should be
	// the framework making the external call.
	Library string

	// statusCode is the status code for the response.  This value takes
	// precedence over the status code set on the Response.
	statusCode *int
}

// MessageProducerSegment instruments calls to add messages to a queueing system.
type MessageProducerSegment struct {
	StartTime SegmentStartTime

	// Library is the name of the library instrumented.  eg. "RabbitMQ",
	// "JMS"
	Library string

	// DestinationType is the destination type.
	DestinationType MessageDestinationType

	// DestinationName is the name of your queue or topic.  eg. "UsersQueue".
	DestinationName string

	// DestinationTemporary must be set to true if destination is temporary
	// to improve metric grouping.
	DestinationTemporary bool
}

// MessageDestinationType is used for the MessageSegment.DestinationType field.
type MessageDestinationType string

// These message destination type constants are used in for the
// MessageSegment.DestinationType field.
const (
	MessageQueue    MessageDestinationType = "Queue"
	MessageTopic    MessageDestinationType = "Topic"
	MessageExchange MessageDestinationType = "Exchange"
)

func (s *span) end() {
	s.Span.End()
	s.Lock()
	s.ended = true
	s.Unlock()
	parent := s.parent
	for parent != nil {
		if !parent.isEnded() {
			s.thread.setCurrentSpan(parent)
			return
		}
		parent = parent.parent
	}
}

func (s *span) isEnded() bool {
	if s == nil {
		return true
	}
	s.Lock()
	defer s.Unlock()
	return s.ended
}

// AddAttribute adds a key value pair to the current segment.
//
// The key must contain fewer than than 255 bytes.  The value must be a
// number, string, or boolean.
func (s *Segment) AddAttribute(key string, val interface{}) {
	if s == nil {
		return
	}
	if s.StartTime.span == nil {
		return
	}
	s.StartTime.Span.SetAttribute(key, val)
}

// End finishes the segment.
func (s *Segment) End() {
	if s == nil {
		return
	}
	if s.StartTime.isEnded() {
		return
	}
	s.StartTime.Span.SetName(s.Name)
	s.StartTime.end()
}

// AddAttribute adds a key value pair to the current DatastoreSegment.
//
// The key must contain fewer than than 255 bytes.  The value must be a
// number, string, or boolean.
func (s *DatastoreSegment) AddAttribute(key string, val interface{}) {
	if s == nil {
		return
	}
	if s.StartTime.span == nil {
		return
	}
	s.StartTime.Span.SetAttribute(key, val)
}

// End finishes the datastore segment.
func (s *DatastoreSegment) End() {
	if s == nil {
		return
	}
	if s.StartTime.isEnded() {
		return
	}

	s.addAttributes(s.StartTime.Span.SetAttribute)
	s.StartTime.Span.SetName(s.name())
	s.StartTime.end()
}

func (s *DatastoreSegment) addAttributes(setter func(string, interface{})) {
	setter("db.system", valOrUnknown(string(s.Product)))
	setter("db.statement", valOrUnknown(s.statement()))
	setter("db.operation", valOrUnknown(s.Operation))
	setter("db.collection", valOrUnknown(s.Collection))

	if net.ParseIP(s.Host) != nil {
		setter("net.peer.ip", s.Host)
	} else {
		setter("net.peer.name", valOrUnknown(s.Host))
	}
	if s.PortPathOrID != "" {
		if port, err := strconv.Atoi(s.PortPathOrID); err == nil {
			setter("net.peer.port", port)
		}
	}

	switch s.Product {
	case DatastoreCassandra:
		setter("db.cassandra.keyspace", valOrUnknown(s.DatabaseName))
	case DatastoreRedis:
		setter("db.redis.database_index", valOrUnknown(s.DatabaseName))
	case DatastoreMongoDB:
		setter("db.mongodb.collection", valOrUnknown(s.DatabaseName))
	default:
		setter("db.name", valOrUnknown(s.DatabaseName))
	}
}

func (s *DatastoreSegment) name() string {
	return s.statement()
}

func valOrUnknown(v string) string {
	if v == "" {
		return "unknown"
	}
	return v
}

func (s *DatastoreSegment) statement() string {
	pq := s.ParameterizedQuery
	if pq == "" {
		op := valOrUnknown(s.Operation)
		coll := valOrUnknown(s.Collection)
		prod := valOrUnknown(string(s.Product))
		pq = "'" + op + "' on '" + coll + "' using '" + prod + "'"
	}
	return pq
}

// AddAttribute adds a key value pair to the current ExternalSegment.
//
// The key must contain fewer than than 255 bytes.  The value must be a
// number, string, or boolean.
func (s *ExternalSegment) AddAttribute(key string, val interface{}) {
	if s == nil {
		return
	}
	if s.StartTime.span == nil {
		return
	}
	s.StartTime.Span.SetAttribute(key, val)
}

// End finishes the external segment.
func (s *ExternalSegment) End() {
	if s == nil {
		return
	}
	if s.StartTime.isEnded() {
		return
	}
	s.addAttributes(s.StartTime.Span.SetAttributes)
	s.StartTime.Span.SetName(s.name())
	s.setSpanStatus(s.StartTime.Span.SetStatus)
	s.StartTime.end()
}

func (s *ExternalSegment) setSpanStatus(setter func(codes.Code, string)) {
	var code int
	if s.statusCode != nil {
		code = *s.statusCode
	} else if s.Response != nil {
		code = s.Response.StatusCode
	}
	if code < 17 {
		// Assume the code is already a grpc status code
		c := codes.Code(code)
		setter(c, c.String())
		return
	}
	setter(standard.SpanStatusFromHTTPStatusCode(code))
}

func (s *ExternalSegment) addAttributes(setter func(...kv.KeyValue)) {
	req := s.Request
	if s.Response != nil && s.Response.Request != nil {
		req = s.Response.Request
	}
	if req != nil {
		setter(standard.EndUserAttributesFromHTTPRequest(req)...)
		if req.URL != nil {
			setter(standard.HTTPClientAttributesFromHTTPRequest(req)...)
		}
	}

	if s.Procedure != "" {
		setter(standard.HTTPMethodKey.String(s.Procedure))
	}
	setter(standard.HTTPUrlKey.String(s.cleanURL()))

	lib := s.Library
	if lib == "" {
		lib = "http"
	}
	setter(kv.Key("http.component").String(lib))

	var code int
	if s.statusCode != nil {
		code = *s.statusCode
	} else if s.Response != nil {
		code = s.Response.StatusCode
	}
	setter(standard.HTTPAttributesFromHTTPStatusCode(code)...)
}

func (s *ExternalSegment) cleanURL() string {
	url := s.URL
	if url == "" {
		r := s.Request
		if nil != s.Response && nil != s.Response.Request {
			r = s.Response.Request
		}
		if r != nil && r.URL != nil && r.URL.Scheme != "" {
			url = r.URL.Scheme + "://" + r.URL.Host + r.URL.Path
		}
	}
	return valOrUnknown(url)
}

func (s *ExternalSegment) name() string {
	return s.library() + " " + s.method() + " " + s.host()
}

func (s *ExternalSegment) library() string {
	if s.Library == "" {
		return "http"
	}
	return s.Library
}

func (s *ExternalSegment) url() (*url.URL, error) {
	if "" != s.URL {
		return url.Parse(s.URL)
	}
	r := s.Request
	if nil != s.Response && nil != s.Response.Request {
		r = s.Response.Request
	}
	if r != nil {
		return r.URL, nil
	}
	return nil, nil
}

func (s *ExternalSegment) host() string {
	host := s.Host
	if host == "" {
		if url, _ := s.url(); url != nil {
			host = url.Host
		}
	}
	host = valOrUnknown(host)
	return host
}

func (s *ExternalSegment) method() string {
	if "" != s.Procedure {
		return s.Procedure
	}
	r := s.Request
	if nil != s.Response && nil != s.Response.Request {
		r = s.Response.Request
	}

	if nil != r {
		if "" != r.Method {
			return r.Method
		}
		// Golang's http package states that when a client's Request has
		// an empty string for Method, the method is GET.
		return "GET"
	}

	return "unknown"
}

// AddAttribute adds a key value pair to the current MessageProducerSegment.
//
// The key must contain fewer than than 255 bytes.  The value must be a
// number, string, or boolean.
func (s *MessageProducerSegment) AddAttribute(key string, val interface{}) {
	if s == nil {
		return
	}
	if s.StartTime.span == nil {
		return
	}
	s.StartTime.Span.SetAttribute(key, val)
}

// End finishes the message segment.
func (s *MessageProducerSegment) End() {
	if s == nil {
		return
	}
	if s.StartTime.isEnded() {
		return
	}
	s.StartTime.Span.SetName(s.name())
	s.StartTime.end()
}

func (s *MessageProducerSegment) name() string {
	dest := s.DestinationName
	if s.DestinationTemporary {
		dest = "(temporary)"
	}
	dest = valOrUnknown(dest)
	return dest + " send"
}

// SetStatusCode sets the status code for the response of this ExternalSegment.
// This status code will be included as an attribute on Span Events.  If status
// code is not set using this method, then the status code found on the
// ExternalSegment.Response will be used.
//
// Use this method when you are creating ExternalSegment manually using either
// StartExternalSegment or the ExternalSegment struct directly.  Status code is
// set automatically when using NewRoundTripper.
func (s *ExternalSegment) SetStatusCode(code int) {
	s.StartTime.Lock()
	s.statusCode = &code
	s.StartTime.Unlock()
}

// StartExternalSegment starts the instrumentation of an external call and adds
// distributed tracing headers to the request.  If the Transaction parameter is
// nil then StartExternalSegment will look for a Transaction in the request's
// context using FromContext.
//
// Using the same http.Client for all of your external requests?  Check out
// NewRoundTripper: You may not need to use StartExternalSegment at all!
//
func StartExternalSegment(txn *Transaction, request *http.Request) *ExternalSegment {
	if nil == txn && nil != request {
		txn = FromContext(request.Context())
	}
	s := &ExternalSegment{
		StartTime: txn.StartSegmentNow(),
		Request:   request,
	}

	if nil != request && nil != request.Header {
		txn.InsertDistributedTraceHeaders(request.Header)
	}

	return s
}
