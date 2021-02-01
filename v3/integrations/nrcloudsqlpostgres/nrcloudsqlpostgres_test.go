// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrcloudsqlpostgres

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
		// key,value pairs
		{
			dsn:             "host=1.2.3.4 port=1234 dbname=mydb",
			expHost:         "1.2.3.4",
			expPortPathOrID: "1234",
			expDatabaseName: "mydb",
		},
		{
			dsn:             "host =1.2.3.4 port= 1234 dbname = mydb",
			expHost:         "1.2.3.4",
			expPortPathOrID: "1234",
			expDatabaseName: "mydb",
		},
		{
			dsn:             "host =        1.2.3.4 port=\t\t1234 dbname =\n\t\t\tmydb",
			expHost:         "1.2.3.4",
			expPortPathOrID: "1234",
			expDatabaseName: "mydb",
		},
		{
			dsn:             "host ='1.2.3.4' port= '1234' dbname = 'mydb'",
			expHost:         "1.2.3.4",
			expPortPathOrID: "1234",
			expDatabaseName: "mydb",
		},
		{
			dsn:             `host='ain\'t_single_quote' port='port\\slash' dbname='my db spaced'`,
			expHost:         `ain\'t_single_quote`,
			expPortPathOrID: `port\\slash`,
			expDatabaseName: "my db spaced",
		},
		{
			dsn:             `host=localhost port=so=does=this`,
			expHost:         "localhost",
			expPortPathOrID: "so=does=this",
		},
		{
			dsn:             "host=1.2.3.4 hostaddr=5.6.7.8",
			expHost:         "5.6.7.8",
			expPortPathOrID: "5432",
		},
		{
			dsn:             "hostaddr=5.6.7.8 host=1.2.3.4",
			expHost:         "5.6.7.8",
			expPortPathOrID: "5432",
		},
		{
			dsn:             "hostaddr=1.2.3.4",
			expHost:         "1.2.3.4",
			expPortPathOrID: "5432",
		},
		{
			dsn:             "host=example.com,example.org port=80,443",
			expHost:         "example.com",
			expPortPathOrID: "80",
		},
		{
			dsn:             "hostaddr=example.com,example.org port=80,443",
			expHost:         "example.com",
			expPortPathOrID: "80",
		},
		{
			dsn:             "hostaddr='' host='' port=80,",
			expHost:         "localhost",
			expPortPathOrID: "80",
		},
		{
			dsn:             "host=/path/to/socket",
			expHost:         "localhost",
			expPortPathOrID: "/path/to/socket/.s.PGSQL.5432",
		},
		{
			dsn:             "port=1234 host=/path/to/socket",
			expHost:         "localhost",
			expPortPathOrID: "/path/to/socket/.s.PGSQL.1234",
		},
		{
			dsn:             "host=/path/to/socket port=1234",
			expHost:         "localhost",
			expPortPathOrID: "/path/to/socket/.s.PGSQL.1234",
		},

		// env vars
		{
			dsn:             "host=host_string port=port_string dbname=dbname_string",
			expHost:         "host_string",
			expPortPathOrID: "port_string",
			expDatabaseName: "dbname_string",
			env: map[string]string{
				"PGHOST":     "host_env",
				"PGPORT":     "port_env",
				"PGDATABASE": "dbname_env",
			},
		},
		{
			dsn:             "",
			expHost:         "host_env",
			expPortPathOrID: "port_env",
			expDatabaseName: "dbname_env",
			env: map[string]string{
				"PGHOST":     "host_env",
				"PGPORT":     "port_env",
				"PGDATABASE": "dbname_env",
			},
		},
		{
			dsn:             "host=host_string",
			expHost:         "host_string",
			expPortPathOrID: "5432",
			env: map[string]string{
				"PGHOSTADDR": "hostaddr_env",
			},
		},
		{
			dsn:             "hostaddr=hostaddr_string",
			expHost:         "hostaddr_string",
			expPortPathOrID: "5432",
			env: map[string]string{
				"PGHOST": "host_env",
			},
		},
		{
			dsn:             "host=host_string hostaddr=hostaddr_string",
			expHost:         "hostaddr_string",
			expPortPathOrID: "5432",
			env: map[string]string{
				"PGHOST": "host_env",
			},
		},
	}

	for _, test := range testcases {
		getenv := func(env string) string {
			return test.env[env]
		}

		s := &newrelic.DatastoreSegment{}
		parseDSN(getenv)(s, test.dsn)

		if test.expHost != s.Host {
			t.Errorf(`incorrect host, expected="%s", actual="%s"`, test.expHost, s.Host)
		}
		if test.expPortPathOrID != s.PortPathOrID {
			t.Errorf(`incorrect port path or id, expected="%s", actual="%s"`, test.expPortPathOrID, s.PortPathOrID)
		}
		if test.expDatabaseName != s.DatabaseName {
			t.Errorf(`incorrect database name, expected="%s", actual="%s"`, test.expDatabaseName, s.DatabaseName)
		}
	}
}
