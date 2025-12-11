// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package nroci

import (
	"testing"

	"github.com/oracle/nosql-go-sdk/nosqldb"
)

func Test_extractRequestFields(t *testing.T) {
	// as we move into other types of requests, I will refactor this test to take in those types as well
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		req   any
		want  string
		want2 string
	}{
		{
			name:  "Default case should return 3 empty strings with non-used type",
			req:   &nosqldb.SystemRequest{},
			want:  "",
			want2: "",
		},
		{
			name: "Should populate all 3 strings with *nosqldb.TableRequest",
			req: &nosqldb.TableRequest{
				TableName: "table1",
				Statement: `SELECT * FROM table1 WHERE param="value"`,
				Namespace: "oci_test",
			},
			want:  "table1",
			want2: `SELECT * FROM table1 WHERE param="value"`,
		},
		{
			name:  "Should return empty string with *nosqldb.TableRequest",
			req:   &nosqldb.TableRequest{},
			want:  "",
			want2: "",
		},
		{
			name: "Should populate all 3 strings with *nosqldb.QueryRequest",
			req: &nosqldb.QueryRequest{
				TableName: "qrtable1",
				Statement: `SELECT * FROM qrtable1 WHERE param="value"`,
			},
			want:  "qrtable1",
			want2: `SELECT * FROM qrtable1 WHERE param="value"`,
		},
		{
			name:  "Should return empty string with *nosqldb.QueryRequest",
			req:   &nosqldb.QueryRequest{},
			want:  "",
			want2: "",
		},
		{
			name: "Should populate all 3 strings with *nosqldb.PutRequest",
			req: &nosqldb.PutRequest{
				TableName: "ptable",
			},
			want:  "ptable",
			want2: "",
		},
		{
			name:  "Should return empty string with *nosqldb.PutRequest",
			req:   &nosqldb.PutRequest{},
			want:  "",
			want2: "",
		},
		{
			name: "Should populate all 3 strings with *nosqldb.WriteMultiple",
			req: &nosqldb.WriteMultipleRequest{
				TableName: "wrtable",
			},
			want:  "wrtable",
			want2: "",
		},
		{
			name:  "Should return empty string with *nosqldb.WriteMultiple",
			req:   &nosqldb.WriteMultipleRequest{},
			want:  "",
			want2: "",
		},
		{
			name: "Should populate all 3 strings with *nosqldb.DeleteRequest",
			req: &nosqldb.DeleteRequest{
				TableName: "dtable",
				Namespace: "oci_test_delete",
			},
			want:  "dtable",
			want2: "",
		},
		{
			name:  "Should return empty string with *nosqldb.DeleteRequest",
			req:   &nosqldb.DeleteRequest{},
			want:  "",
			want2: "",
		},
		{
			name: "Should populate all 3 strings with *nosqldb.MultiDeleteRequest",
			req: &nosqldb.MultiDeleteRequest{
				TableName: "mdtable",
				Namespace: "oci_test_mdelete",
			},
			want:  "mdtable",
			want2: "",
		},
		{
			name:  "Should return empty string with *nosqldb.MultiDeleteRequest",
			req:   &nosqldb.MultiDeleteRequest{},
			want:  "",
			want2: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got2 := extractRequestFields(tt.req)
			if got != tt.want {
				t.Errorf("extractRequestFields() = %v, want %v", got, tt.want)
			}
			if got2 != tt.want2 {
				t.Errorf("extractRequestFields() = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func Test_extractHostPort(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		endpoint string
		want     string
		want2    string
	}{
		{
			name:     "Endpoint is empty should return 2 empty strings",
			endpoint: "",
			want:     "",
			want2:    "",
		},
		{
			name:     "url.Parse returns an error.  Should return 2 empty strings",
			endpoint: "not:an.en#d()point",
			want:     "",
			want2:    "",
		},
		{
			name:     "Host and port both exist in a valid url with https scheme",
			endpoint: "https://ocitest:8080",
			want:     "ocitest",
			want2:    "8080",
		},
		{
			name:     "Host and port both exist in a valid url with http scheme",
			endpoint: "http://ocitest:8080",
			want:     "ocitest",
			want2:    "8080",
		},
		{
			name:     "Host exists but port does not in a valid url with https scheme",
			endpoint: "https://ocitest",
			want:     "ocitest",
			want2:    "443",
		},
		{
			name:     "Host exists but port does not in a valid url with http scheme",
			endpoint: "http://ocitest",
			want:     "ocitest",
			want2:    "80",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got2 := extractHostPort(tt.endpoint)
			if got != tt.want {
				t.Errorf("extractHostPort() -> host = %v, want %v", got, tt.want)
			}
			if got2 != tt.want2 {
				t.Errorf("extractHostPort() -> port = %v, want %v", got2, tt.want2)
			}
		})
	}
}
