// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build go1.8
// +build go1.8

package awssupport

import (
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
