// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build go1.8
// +build go1.8

package awssupport

import (
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
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

func TestAWSAccountIdFromAWSAccessKey(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		creds      aws.Credentials
		want       string
		wantErr    bool
		wantErrStr string // error message returned
	}{
		{
			name: "first test",
			creds: aws.Credentials{
				AccountID:   "",
				AccessKeyID: "AKIASAWSR23456AWS357",
			},
			want:    "138954266361",
			wantErr: false,
		},
		{
			name: "AccountID already exists and access key exists. Should return AccountID immediately",
			creds: aws.Credentials{
				AccountID:   "123451234512",
				AccessKeyID: "ASKDHA123457AKJFHAKS",
			},
			want:    "123451234512",
			wantErr: false,
		},
		{
			name: "AccountID already exists and access key exists with too short of length. Should return AccountID immediately",
			creds: aws.Credentials{
				AccountID:   "123451234512",
				AccessKeyID: "a",
			},
			want:    "123451234512",
			wantErr: false,
		},
		{
			name: "AccountID already exists and access key exists with improper format. Should return AccountID immediately",
			creds: aws.Credentials{
				AccountID:   "123451234512",
				AccessKeyID: "a a a.                      ",
			},
			want:    "123451234512",
			wantErr: false,
		},
		{
			name: "AccountID already exists and access key does not exist. Should return AccountID immediately",
			creds: aws.Credentials{
				AccountID: "123451234512",
			},
			want:    "123451234512",
			wantErr: false,
		},
		{
			name:       "AccountID does not exist and access key does not exist.  Should return an error",
			creds:      aws.Credentials{},
			want:       "",
			wantErr:    true,
			wantErrStr: "no access key id found",
		},
		{
			name: "AccountID does not exist and access key is in an improper format. Should return an error",
			creds: aws.Credentials{
				AccessKeyID: "123asdfas",
			},
			want:       "",
			wantErr:    true,
			wantErrStr: "improper access key id format",
		},
		{
			name: "AccountID does not exist and access key is in an improper format with only one character. Should return an error",
			creds: aws.Credentials{
				AccessKeyID: "a",
			},
			want:       "",
			wantErr:    true,
			wantErrStr: "improper access key id format",
		},
		{
			name: "AccountID does not exist and access key is in an improper format for decoding.",
			creds: aws.Credentials{
				AccessKeyID: "a a a.                      ",
			},
			want:       "",
			wantErr:    true,
			wantErrStr: "error decoding access keys",
		},
		{
			name: "AccountID does not exist and access key contains non base32 characters",
			creds: aws.Credentials{
				AccessKeyID: "AKIA1234567899876541",
			},
			want:       "",
			wantErr:    true,
			wantErrStr: "error decoding access keys",
		},
		{
			name: "AccountID does not exist and access key contains non base32 characters and is too short in length",
			creds: aws.Credentials{
				AccessKeyID: "AKIA1818",
			},
			want:       "",
			wantErr:    true,
			wantErrStr: "improper access key id format",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := AWSAccountIdFromAWSAccessKey(tt.creds)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("AWSAccountIdFromAWSAccessKey() failed: %v", gotErr)
				} else {
					if tt.wantErrStr != gotErr.Error() {
						t.Errorf("AWSAccountIdFromAWSAccessKey() error = %v, want %v", gotErr.Error(), tt.wantErrStr)
					}
				}
				return
			}
			if tt.wantErr {
				t.Fatal("AWSAccountIdFromAWSAccessKey() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if tt.want != got {
				t.Errorf("AWSAccountIdFromAWSAccessKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
