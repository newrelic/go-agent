// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package nroci

import (
	"strings"
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

// Test_parseConfigAtLine tests parseConfigAtLine(start int, splitContent []string)
// It will check for keys in the oci config file and assign them accordingly to the
// *nrConfig.  Currently, the only key we pull in is tenancyOCID so that is the only
// field we are checking for in this test
func Test_parseConfigAtLine(t *testing.T) {
	splitContent := strings.Split(`
[DEFAULT]
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy
region=us-ashburn-1

[ADMIN_USER]
user=ocid1.user.admin
fingerprint=fingerprint
key_file=keys/admin_key.pem
tenancy=ocid1.tenancy
pass_phrase=funwords

[NO_TENANCY]
user=ocid1.user.admin
fingerprint=fingerprint
key_file=keys/admin_key.pem
pass_phrase=funwords

[NO_EQUALS_SIGN]
user=ocid1.user.admin
fingerprint=fingerprint
key_file=keys/admin_key.pem
tenancy:ocid1.tenancy
pass_phrase=funwords`, "\n")
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		start        int
		splitContent []string
		want         *nrConfig
		wantErr      bool
	}{
		{
			name:         "Parse DEFAULT USER (line 2) with TenancyOCID",
			start:        2,
			splitContent: splitContent,
			want: &nrConfig{
				tenancyOCID: "ocid1.tenancy",
			},
			wantErr: false,
		},
		{
			name:         "Parse ADMIN_USER (line 9) with TenancyOCID",
			start:        9,
			splitContent: splitContent,
			want: &nrConfig{
				tenancyOCID: "ocid1.tenancy",
			},
			wantErr: false,
		},
		{
			name:         "Parse NO_TENANCY USER (line 15) with TenancyOCID",
			start:        15,
			splitContent: splitContent,
			want: &nrConfig{
				tenancyOCID: "",
			},
			wantErr: false,
		},
		{
			name:         "Parse [DEFAULT] (line 1) and immediately break",
			start:        1,
			splitContent: splitContent,
			want: &nrConfig{
				tenancyOCID: "",
			},
			wantErr: false,
		},
		{
			name:         "Parse [ADMIN_USER] (line 8) and immediately break",
			start:        8,
			splitContent: splitContent,
			want: &nrConfig{
				tenancyOCID: "",
			},
			wantErr: false,
		},
		{
			name:         "Parse [NO_TENANCY] (line 14) and immediately break",
			start:        14,
			splitContent: splitContent,
			want: &nrConfig{
				tenancyOCID: "",
			},
			wantErr: false,
		},
		{
			name:         "Parse NO_EQUALS USER (line 14) and return empty string",
			start:        14,
			splitContent: splitContent,
			want: &nrConfig{
				tenancyOCID: "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := parseConfigAtLine(tt.start, tt.splitContent)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("parseConfigAtLine() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("parseConfigAtLine() succeeded unexpectedly")
			}
			if tt.want.tenancyOCID != got.tenancyOCID {
				t.Errorf("parseConfigAtLine().tenancyOCID = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseOCIConfigFile(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		data    []byte
		profile string
		want    *nrConfig
		wantErr bool
	}{
		{
			name:    "No data is passed in",
			data:    []byte{},
			profile: "",
			want:    &nrConfig{},
			wantErr: true,
		},
		{
			name: "Data is unable to be parsed data is passed in with profile",
			data: []byte(`
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy
region=us-ashburn-1`),
			profile: "",
			want:    &nrConfig{},
			wantErr: true,
		},
		{
			name: "Data is unable to be parsed data is passed in with no matching profile",
			data: []byte(`
[DEFAULT]
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy
region=us-ashburn-1`),
			profile: "NOT_DEFAULT",
			want:    &nrConfig{},
			wantErr: true,
		},
		{
			name: "Data is unable to be parsed data is passed in with incorrect profile format",
			data: []byte(`
DEFAULT
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy
region=us-ashburn-1`),
			profile: "DEFAULT",
			want:    &nrConfig{},
			wantErr: true,
		},
		{
			name: "Data is unable to be parsed data is passed in with incorrect spacing",
			data: []byte(`
\t[DEFAULT]
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy
region=us-ashburn-1`),
			profile: "DEFAULT",
			want:    &nrConfig{},
			wantErr: true,
		},
		{
			name: "DEFAULT profile is parsed correctly",
			data: []byte(`
[DEFAULT]
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy
region=us-ashburn-1`),
			profile: "DEFAULT",
			want: &nrConfig{
				profile:     "DEFAULT",
				tenancyOCID: "ocid1.tenancy",
			},
			wantErr: false,
		},
		{
			name: "DEFAULT profile is parsed correctly with no tenancy",
			data: []byte(`
[DEFAULT]
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
region=us-ashburn-1`),
			profile: "DEFAULT",
			want: &nrConfig{
				profile:     "DEFAULT",
				tenancyOCID: "",
			},
			wantErr: false,
		},
		{
			name: "NOT_DEFAULT profile is parsed correctly",
			data: []byte(`
[DEFAULT]
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy
region=us-ashburn-1

[NOT_DEFAULT]
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy
region=us-ashburn-1`),
			profile: "NOT_DEFAULT",
			want: &nrConfig{
				profile:     "NOT_DEFAULT",
				tenancyOCID: "ocid1.tenancy",
			},
			wantErr: false,
		},
		{
			name: "NOT_DEFAULT profile is parsed correctly and DEFAULT has no tenancyOCID",
			data: []byte(`
[DEFAULT]
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
region=us-ashburn-1

[NOT_DEFAULT]
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy
region=us-ashburn-1`),
			profile: "NOT_DEFAULT",
			want: &nrConfig{
				profile:     "NOT_DEFAULT",
				tenancyOCID: "ocid1.tenancy",
			},
			wantErr: false,
		},
		{
			name: "NOT_DEFAULT profile is parsed correctly but contains no tenancyOCID",
			data: []byte(`
[DEFAULT]
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy
region=us-ashburn-1

[NOT_DEFAULT]
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
region=us-ashburn-1`),
			profile: "NOT_DEFAULT",
			want: &nrConfig{
				profile:     "NOT_DEFAULT",
				tenancyOCID: "",
			},
			wantErr: false,
		},
		{
			name: "NOT_DEFAULT profile is parsed correctly and no other profile exists",
			data: []byte(`
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy
region=us-ashburn-1

[NOT_DEFAULT]
user=ocid1.user
fingerprint=fingerprint
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy
region=us-ashburn-1`),
			profile: "NOT_DEFAULT",
			want: &nrConfig{
				profile:     "NOT_DEFAULT",
				tenancyOCID: "ocid1.tenancy",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := parseOCIConfigFile(tt.data, tt.profile)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("parseOCIConfigFile() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("parseOCIConfigFile() succeeded unexpectedly")
			}
			if tt.want.profile != got.profile {
				t.Errorf("parseOCIConfigFile() = %v, want %v", got, tt.want)
			}
			if tt.want.tenancyOCID != got.tenancyOCID {
				t.Errorf("parseOCIConfigFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
