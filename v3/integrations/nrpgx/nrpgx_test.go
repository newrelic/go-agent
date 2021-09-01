// Copyright 2020, 2021 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrpgx

import (
	"testing"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func TestParseDSN(t *testing.T) {
	testcases := []struct {
		dsn             string
		expHost         string
		expPortPathOrID string
		expDatabaseName string
		env             map[string]string
	}{
		// urls
		// #0
		{
			dsn:             "postgresql://",
			expHost:         "localhost",
			expPortPathOrID: "5432",
			expDatabaseName: "",
		},
		// #1
		{
			dsn:             "postgresql://localhost",
			expHost:         "localhost",
			expPortPathOrID: "5432",
			expDatabaseName: "",
		},
		// #2
		{
			dsn:             "postgresql://localhost:5433",
			expHost:         "localhost",
			expPortPathOrID: "5433",
			expDatabaseName: "",
		},
		// #3
		{
			dsn:             "postgresql://localhost/mydb",
			expHost:         "localhost",
			expPortPathOrID: "5432",
			expDatabaseName: "mydb",
		},
		// #4
		{
			dsn:             "postgresql://user@localhost",
			expHost:         "localhost",
			expPortPathOrID: "5432",
			expDatabaseName: "",
		},
		// #5
		{
			dsn:             "postgresql://other@localhost/otherdb?connect_timeout=10&application_name=myapp",
			expHost:         "localhost",
			expPortPathOrID: "5432",
			expDatabaseName: "otherdb",
		},
		// #6
		{
			dsn:             "postgresql:///mydb?host=myhost.com&port=5433",
			expHost:         "myhost.com",
			expPortPathOrID: "5433",
			expDatabaseName: "mydb",
		},
		// #7
		{
			dsn:             "postgresql://[2001:db8::1234]/database",
			expHost:         "2001:db8::1234",
			expPortPathOrID: "5432",
			expDatabaseName: "database",
		},
		// #8
		{
			dsn:             "postgresql://[2001:db8::1234]:7890/database",
			expHost:         "2001:db8::1234",
			expPortPathOrID: "7890",
			expDatabaseName: "database",
		},
		// #9
		{
			dsn:             "postgresql:///dbname?host=/var/lib/postgresql",
			expHost:         "localhost",
			expPortPathOrID: "/var/lib/postgresql/.s.PGSQL.5432",
			expDatabaseName: "dbname",
		},
		// #10
		{
			dsn:             "postgresql://%2Fvar%2Flib%2Fpostgresql/dbname",
			expHost:         "",
			expPortPathOrID: "",
			expDatabaseName: "",
		},

		// key,value pairs
		// #11
		{
			dsn:             "host=1.2.3.4 port=1234 dbname=mydb",
			expHost:         "1.2.3.4",
			expPortPathOrID: "1234",
			expDatabaseName: "mydb",
		},
		// #12
		{
			dsn:             "host =1.2.3.4 port= 1234 dbname = mydb",
			expHost:         "1.2.3.4",
			expPortPathOrID: "1234",
			expDatabaseName: "mydb",
		},
		// #13
		{
			dsn:             "host =        1.2.3.4 port=\t\t1234 dbname =\n\t\t\tmydb",
			expHost:         "1.2.3.4",
			expPortPathOrID: "1234",
			expDatabaseName: "mydb",
		},
		// #14
		{
			dsn:             "host ='1.2.3.4' port= '1234' dbname = 'mydb'",
			expHost:         "1.2.3.4",
			expPortPathOrID: "1234",
			expDatabaseName: "mydb",
		},
		// #15
		{
			dsn:             `host='ain\'t_single_quote' port='port\\slash' dbname='my db spaced'`,
			expHost:         `ain\'t_single_quote`,
			expPortPathOrID: `5432`,
			expDatabaseName: "my db spaced",
		},
		// #16
		{
			dsn:             `host=localhost port=so=does=this`,
			expHost:         "localhost",
			expPortPathOrID: "5432",
		},
		// #17
		{
			dsn:             "host=1.2.3.4 hostaddr=5.6.7.8",
			expHost:         "5.6.7.8",
			expPortPathOrID: "5432",
		},
		// #18
		{
			dsn:             "hostaddr=5.6.7.8 host=1.2.3.4",
			expHost:         "5.6.7.8",
			expPortPathOrID: "5432",
		},
		// #19
		{
			dsn:             "hostaddr=1.2.3.4",
			expHost:         "1.2.3.4",
			expPortPathOrID: "5432",
		},
		// #20
		{
			dsn:             "host=example.com,example.org port=80,443",
			expHost:         "example.com",
			expPortPathOrID: "80",
		},
		// #21
		{
			dsn:             "hostaddr=example.com,example.org port=80,443",
			expHost:         "example.com",
			expPortPathOrID: "80",
		},
		// #22
		{
			dsn:             "hostaddr='' host='' port=80,",
			expHost:         "localhost",
			expPortPathOrID: "80",
		},
		// #23
		{
			dsn:             "host=/path/to/socket",
			expHost:         "localhost",
			expPortPathOrID: "/path/to/socket/.s.PGSQL.5432",
		},
		// #24
		{
			dsn:             "port=1234 host=/path/to/socket",
			expHost:         "localhost",
			expPortPathOrID: "/path/to/socket/.s.PGSQL.1234",
		},
		// #25
		{
			dsn:             "host=/path/to/socket port=1234",
			expHost:         "localhost",
			expPortPathOrID: "/path/to/socket/.s.PGSQL.1234",
		},

		// env vars
		// #26
		{
			dsn:             "host=host_string port=port_string dbname=dbname_string",
			expHost:         "host_string",
			expPortPathOrID: "5432",
			expDatabaseName: "dbname_string",
			env: map[string]string{
				"PGHOST":     "host_env",
				"PGPORT":     "port_env",
				"PGDATABASE": "dbname_env",
			},
		},
		// #27
		{
			dsn:             "",
			expHost:         "host_env",
			expPortPathOrID: "5432",
			expDatabaseName: "dbname_env",
			env: map[string]string{
				"PGHOST":     "host_env",
				"PGPORT":     "port_env",
				"PGDATABASE": "dbname_env",
			},
		},
		// #28
		{
			dsn:             "host=host_string",
			expHost:         "host_string",
			expPortPathOrID: "5432",
			env: map[string]string{
				"PGHOSTADDR": "hostaddr_env",
			},
		},
		// #29
		{
			dsn:             "hostaddr=hostaddr_string",
			expHost:         "hostaddr_string",
			expPortPathOrID: "5432",
			env: map[string]string{
				"PGHOST": "host_env",
			},
		},
		// #30
		{
			dsn:             "host=host_string hostaddr=hostaddr_string",
			expHost:         "hostaddr_string",
			expPortPathOrID: "5432",
			env: map[string]string{
				"PGHOST": "host_env",
			},
		},
	}

	for i, test := range testcases {
		getenv := func(env string) string {
			return test.env[env]
		}

		s := &newrelic.DatastoreSegment{}
		parseDSN(getenv)(s, test.dsn)

		if test.expHost != s.Host {
			t.Errorf(`testcase #%d, incorrect host, expected="%s", actual="%s"`, i, test.expHost, s.Host)
		}
		if test.expPortPathOrID != s.PortPathOrID {
			t.Errorf(`testcase #%d, incorrect port path or id, expected="%s", actual="%s"`, i, test.expPortPathOrID, s.PortPathOrID)
		}
		if test.expDatabaseName != s.DatabaseName {
			t.Errorf(`testcase #%d, incorrect database name, expected="%s", actual="%s"`, i, test.expDatabaseName, s.DatabaseName)
		}
	}
}
