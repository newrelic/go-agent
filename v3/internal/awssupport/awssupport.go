// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build go1.8
// +build go1.8

package awssupport

import (
	"context"
	"encoding/base32"
	"fmt"
	"net/http"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"

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

func AWSAccountIdFromAWSAccessKey(creds aws.Credentials) (string, error) {
	if creds.AccountID != "" {
		return creds.AccountID, nil
	}
	if creds.AccessKeyID == "" {
		return "", fmt.Errorf("no access key id found")
	}
	if len(creds.AccessKeyID) < 16 {
		return "", fmt.Errorf("improper access key id format")
	}
	trimmedAccessKey := creds.AccessKeyID[4:]
	decoded, err := base32.StdEncoding.DecodeString(trimmedAccessKey)
	if err != nil {
		return "", fmt.Errorf("error decoding access keys")
	}
	var bigEndian uint64
	for i := 0; i < 6; i++ {
		bigEndian = bigEndian << 8      // shift 8 bits left.  Most significant byte read in first (decoded[i])
		bigEndian |= uint64(decoded[i]) // apply OR for current byte
	}

	mask := uint64(0x7fffffffff80)

	num := (bigEndian & mask) >> 7 // apply mask and get rid of last 7 bytes from mask

	return fmt.Sprintf("%d", num), nil
}
