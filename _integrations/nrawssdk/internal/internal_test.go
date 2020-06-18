// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"net/http"
	"strings"
	"testing"

	requestv2 "github.com/aws/aws-sdk-go-v2/aws"
	restv2 "github.com/aws/aws-sdk-go-v2/private/protocol/rest"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	requestv1 "github.com/aws/aws-sdk-go/aws/request"
	restv1 "github.com/aws/aws-sdk-go/private/protocol/rest"
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
	}

	for i, test := range testcases {
		if out := getRequestID(test.hdr); test.expected != out {
			t.Error(i, out, test.hdr, test.expected)
		}
	}

	// Make sure our assumptions still hold against aws-sdk-go
	for _, test := range testcases {
		req := &requestv1.Request{
			HTTPResponse: &http.Response{
				Header: test.hdr,
			},
		}
		restv1.UnmarshalMeta(req)
		if out := getRequestID(test.hdr); req.RequestID != out {
			t.Error("requestId assumptions incorrect", out, req.RequestID,
				test.hdr, test.expected)
		}
	}

	// Make sure our assumptions still hold against aws-sdk-go-v2
	for _, test := range testcases {
		req := &requestv2.Request{
			HTTPResponse: &http.Response{
				Header: test.hdr,
			},
			Data: &lambda.InvokeOutput{},
		}
		restv2.UnmarshalMeta(req)
		if out := getRequestID(test.hdr); req.RequestID != out {
			t.Error("requestId assumptions incorrect", out, req.RequestID,
				test.hdr, test.expected)
		}
	}
}
