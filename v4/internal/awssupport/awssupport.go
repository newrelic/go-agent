// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// +build go1.8

package awssupport

import (
	"context"
	"net/http"
	"reflect"

	"github.com/newrelic/go-agent/v4/internal/integrationsupport"
	"github.com/newrelic/go-agent/v4/newrelic"
)

type contextKeyType struct{}

var segmentContextKey = contextKeyType(struct{}{})

type endable interface{ End() }

func getTableName(params interface{}) string {
	var tableName string

	v := reflect.ValueOf(params)
	if v.IsValid() && v.Kind() == reflect.Ptr {
		e := v.Elem()
		if e.Kind() == reflect.Struct {
			n := e.FieldByName("TableName")
			if n.IsValid() {
				if name, ok := n.Interface().(*string); ok {
					if nil != name {
						tableName = *name
					}
				}
			}
		}
	}

	return tableName
}

// GetRequestID looks for the AWS request ID header.
func GetRequestID(hdr http.Header) string {
	id := hdr.Get("X-Amzn-Requestid")
	if id == "" {
		// Alternative version of request id in the header
		id = hdr.Get("X-Amz-Request-Id")
	}
	return id
}

// StartSegmentInputs is used as the input to StartSegment.
type StartSegmentInputs struct {
	HTTPRequest *http.Request
	ServiceName string
	Operation   string
	Region      string
	Params      interface{}
}

// StartSegment starts a segment of either type DatastoreSegment or
// ExternalSegment given the serviceName provided. The segment is then added to
// the request context.
func StartSegment(input StartSegmentInputs) *http.Request {

	httpCtx := input.HTTPRequest.Context()
	txn := newrelic.FromContext(httpCtx)

	var segment endable
	// Service name capitalization is different for v1 and v2.
	if input.ServiceName == "dynamodb" || input.ServiceName == "DynamoDB" {
		segment = &newrelic.DatastoreSegment{
			Product:            newrelic.DatastoreDynamoDB,
			Collection:         getTableName(input.Params),
			Operation:          input.Operation,
			ParameterizedQuery: "",
			QueryParameters:    nil,
			Host:               input.HTTPRequest.URL.Host,
			PortPathOrID:       input.HTTPRequest.URL.Port(),
			DatabaseName:       "",
			StartTime:          txn.StartSegmentNow(),
		}
	} else {
		segment = newrelic.StartExternalSegment(txn, input.HTTPRequest)
	}

	integrationsupport.AddAgentSpanAttribute(txn, newrelic.SpanAttributeAWSOperation, input.Operation)
	integrationsupport.AddAgentSpanAttribute(txn, newrelic.SpanAttributeAWSRegion, input.Region)

	ctx := context.WithValue(httpCtx, segmentContextKey, segment)
	return input.HTTPRequest.WithContext(ctx)
}

// EndSegment will end any segment found in the given context.
func EndSegment(ctx context.Context, hdr http.Header) {
	if segment, ok := ctx.Value(segmentContextKey).(endable); ok {
		if id := GetRequestID(hdr); "" != id {
			txn := newrelic.FromContext(ctx)
			integrationsupport.AddAgentSpanAttribute(txn, newrelic.SpanAttributeAWSRequestID, id)
		}
		segment.End()
	}
}
