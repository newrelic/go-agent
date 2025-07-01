// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build go1.8
// +build go1.8

package awssupport

import (
	"github.com/aws/aws-sdk-go/aws/client/metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"net/http"
	"strings"
	"testing"
)

func TestGetTableName(t *testing.T) {
	str := "this is a string"
	var emptyStr string
	strPtr := &str
	emptyStrPtr := &emptyStr

	testcases := []struct {
		params   interface{}
		expected string
	}{
		{params: nil, expected: ""},
		{params: str, expected: ""},
		{params: strPtr, expected: ""},
		{params: struct{ other string }{other: str}, expected: ""},
		{params: &struct{ other string }{other: str}, expected: ""},
		{params: struct{ TableName bool }{TableName: true}, expected: ""},
		{params: &struct{ TableName bool }{TableName: true}, expected: ""},
		{params: struct{ TableName string }{TableName: str}, expected: ""},
		{params: &struct{ TableName string }{TableName: str}, expected: ""},
		{params: struct{ TableName *string }{TableName: nil}, expected: ""},
		{params: &struct{ TableName *string }{TableName: nil}, expected: ""},
		{params: struct{ TableName *string }{TableName: emptyStrPtr}, expected: ""},
		{params: &struct{ TableName *string }{TableName: emptyStrPtr}, expected: ""},
		{params: struct{ TableName *string }{TableName: strPtr}, expected: ""},
		{params: &struct{ TableName *string }{TableName: strPtr}, expected: str},
	}

	for i, test := range testcases {
		if out := getTableName(test.params); test.expected != out {
			t.Error(i, out, test.params, test.expected)
		}
	}
}

func TestGetRequestID(t *testing.T) {
	primary := "X-Amzn-Requestid"
	secondary := "X-Amz-Request-Id"

	testcases := []struct {
		hdr      http.Header
		expected string
	}{
		{hdr: http.Header{
			"hello": []string{"world"},
		}, expected: ""},

		{hdr: http.Header{
			strings.ToUpper(primary): []string{"hello"},
		}, expected: ""},

		{hdr: http.Header{
			primary: []string{"hello"},
		}, expected: "hello"},

		{hdr: http.Header{
			secondary: []string{"hello"},
		}, expected: "hello"},

		{hdr: http.Header{
			primary:   []string{"hello"},
			secondary: []string{"world"},
		}, expected: "hello"},

		{hdr: http.Header{}, expected: ""},
	}

	for i, test := range testcases {
		if out := GetRequestID(test.hdr); test.expected != out {
			t.Error(i, out, test.hdr, test.expected)
		}
	}
}

func TestStartAndEndSegment(t *testing.T) {
	req := request.Request{
		ClientInfo: metadata.ClientInfo{ServiceName: "awssupport-test", SigningRegion: "us-east-1"},
		Operation:  &request.Operation{HTTPMethod: "GET", HTTPPath: "/"},
		HTTPRequest: &http.Request{
			Method: "GET",
		},
		Params: nil,
	}

	input := StartSegmentInputs{
		HTTPRequest: req.HTTPRequest,
		ServiceName: req.ClientInfo.ServiceName,
		Operation:   req.Operation.Name,
		Region:      req.ClientInfo.SigningRegion,
		Params:      req.Params,
	}

	ctx := req.HTTPRequest.Context()
	v := ctx.Value(segmentContextKey)
	if v != nil {
		t.Errorf("Context segmentContextKey value is not nil %v", v)
	}

	req.HTTPRequest = StartSegment(input)

	ctx = req.HTTPRequest.Context()
	v = ctx.Value(segmentContextKey)
	if v == nil {
		t.Error("Context segmentContextKey value is nil")
	}

	EndSegment(ctx, req.HTTPResponse)
	v = req.HTTPRequest.Context().Value(segmentContextKey)
	t.Log("Done")
}

func TestStartAndEndDynamoDbSegment(t *testing.T) {
	req := request.Request{
		ClientInfo: metadata.ClientInfo{ServiceName: "dynamodb", SigningRegion: "us-east-1"},
		Operation:  &request.Operation{HTTPMethod: "GET", HTTPPath: "/"},
		HTTPRequest: &http.Request{
			Method: "GET",
		},
		Params: map[string]string{"TableName": "myTable"},
	}

	input := StartSegmentInputs{
		HTTPRequest: req.HTTPRequest,
		ServiceName: req.ClientInfo.ServiceName,
		Operation:   req.Operation.Name,
		Region:      req.ClientInfo.SigningRegion,
		Params:      req.Params,
	}

	ctx := req.HTTPRequest.Context()
	v := ctx.Value(segmentContextKey)
	if v != nil {
		t.Errorf("Context segmentContextKey value is not nil %v", v)
	}

	req.HTTPRequest = StartSegment(input)

	ctx = req.HTTPRequest.Context()
	v = ctx.Value(segmentContextKey)
	if v == nil {
		t.Error("Context segmentContextKey value is nil")
	}

	EndSegment(ctx, req.HTTPResponse)
	v = req.HTTPRequest.Context().Value(segmentContextKey)
	t.Log("Done")
}
