// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// +build go1.8

package awssupport

import (
	"context"
	"net/http"
	"reflect"

	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
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
	if input.ServiceName == "dynamodb" || input.ServiceName == "DynamoDB" || input.ServiceName == "dax" {
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
		// Do NOT set any distributed trace headers.
		// Doing so can cause the AWS SDK's request signature to be invalid on retries.
		segment = &newrelic.ExternalSegment{
			Request:   input.HTTPRequest,
			StartTime: txn.StartSegmentNow(),
		}
	}

	integrationsupport.AddAgentSpanAttribute(txn, newrelic.SpanAttributeAWSOperation, input.Operation)
	integrationsupport.AddAgentSpanAttribute(txn, newrelic.SpanAttributeAWSRegion, input.Region)

	ctx := context.WithValue(httpCtx, segmentContextKey, segment)
	return input.HTTPRequest.WithContext(ctx)
}

// EndSegment will end any segment found in the given context.
func EndSegment(ctx context.Context, resp *http.Response) {
	if segment, ok := ctx.Value(segmentContextKey).(endable); ok {
		if resp != nil {
			if extSegment, ok := segment.(*newrelic.ExternalSegment); ok {
				extSegment.Response = resp
			}
			if requestID := GetRequestID(resp.Header); requestID != "" {
				txn := newrelic.FromContext(ctx)
				integrationsupport.AddAgentSpanAttribute(txn, newrelic.SpanAttributeAWSRequestID, requestID)
			}
		}
		segment.End()
	}
}
