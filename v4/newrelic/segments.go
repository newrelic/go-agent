// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/api/trace"
)

type span struct {
	Span trace.Span
	ctx  context.Context
	// TODO: linked list and ballooning memory?
	parent *span
	ended  bool
	// TODO: reference cycles?
	txn *Transaction
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
	s.ended = true
	if s.txn != nil {
		parent := s.parent
		for parent != nil {
			if !parent.ended {
				s.txn.setCurrentSpan(s.parent)
				return
			}
			parent = parent.parent
		}
	}
}

// AddAttribute adds a key value pair to the current segment.
//
// The key must contain fewer than than 255 bytes.  The value must be a
// number, string, or boolean.
func (s *Segment) AddAttribute(key string, val interface{}) {}

// End finishes the segment.
func (s *Segment) End() {
	if s == nil {
		return
	}
	if s.StartTime.span == nil {
		return
	}
	s.StartTime.Span.SetName(s.Name)
	s.StartTime.end()
}

// AddAttribute adds a key value pair to the current DatastoreSegment.
//
// The key must contain fewer than than 255 bytes.  The value must be a
// number, string, or boolean.
func (s *DatastoreSegment) AddAttribute(key string, val interface{}) {}

// End finishes the datastore segment.
func (s *DatastoreSegment) End() {}

// AddAttribute adds a key value pair to the current ExternalSegment.
//
// The key must contain fewer than than 255 bytes.  The value must be a
// number, string, or boolean.
func (s *ExternalSegment) AddAttribute(key string, val interface{}) {}

// End finishes the external segment.
func (s *ExternalSegment) End() {}

// AddAttribute adds a key value pair to the current MessageProducerSegment.
//
// The key must contain fewer than than 255 bytes.  The value must be a
// number, string, or boolean.
func (s *MessageProducerSegment) AddAttribute(key string, val interface{}) {}

// End finishes the message segment.
func (s *MessageProducerSegment) End() {}

// SetStatusCode sets the status code for the response of this ExternalSegment.
// This status code will be included as an attribute on Span Events.  If status
// code is not set using this method, then the status code found on the
// ExternalSegment.Response will be used.
//
// Use this method when you are creating ExternalSegment manually using either
// StartExternalSegment or the ExternalSegment struct directly.  Status code is
// set automatically when using NewRoundTripper.
func (s *ExternalSegment) SetStatusCode(code int) {}

// StartExternalSegment starts the instrumentation of an external call and adds
// distributed tracing headers to the request.  If the Transaction parameter is
// nil then StartExternalSegment will look for a Transaction in the request's
// context using FromContext.
//
// Using the same http.Client for all of your external requests?  Check out
// NewRoundTripper: You may not need to use StartExternalSegment at all!
//
func StartExternalSegment(txn *Transaction, request *http.Request) *ExternalSegment {
	return nil
}
