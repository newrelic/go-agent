// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrpq

import (
	"testing"

	newrelic "github.com/newrelic/go-agent"
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
		{
			dsn:             "postgresql://",
			expHost:         "localhost",
			expPortPathOrID: "5432",
			expDatabaseName: "",
		},
		{
			dsn:             "postgresql://localhost",
			expHost:         "localhost",
			expPortPathOrID: "5432",
			expDatabaseName: "",
		},
		{
			dsn:             "postgresql://localhost:5433",
			expHost:         "localhost",
			expPortPathOrID: "5433",
			expDatabaseName: "",
		},
		{
			dsn:             "postgresql://localhost/mydb",
			expHost:         "localhost",
			expPortPathOrID: "5432",
			expDatabaseName: "mydb",
		},
		{
			dsn:             "postgresql://user@localhost",
			expHost:         "localhost",
			expPortPathOrID: "5432",
			expDatabaseName: "",
		},
		{
			dsn:             "postgresql://other@localhost/otherdb?connect_timeout=10&application_name=myapp",
			expHost:         "localhost",
			expPortPathOrID: "5432",
			expDatabaseName: "otherdb",
		},
		{
			dsn:             "postgresql:///mydb?host=myhost.com&port=5433",
			expHost:         "myhost.com",
			expPortPathOrID: "5433",
			expDatabaseName: "mydb",
		},
		{
			dsn:             "postgresql://[2001:db8::1234]/database",
			expHost:         "2001:db8::1234",
			expPortPathOrID: "5432",
			expDatabaseName: "database",
		},
		{
			dsn:             "postgresql://[2001:db8::1234]:7890/database",
			expHost:         "2001:db8::1234",
			expPortPathOrID: "7890",
			expDatabaseName: "database",
		},
		{
			dsn:             "postgresql:///dbname?host=/var/lib/postgresql",
			expHost:         "localhost",
			expPortPathOrID: "/var/lib/postgresql/.s.PGSQL.5432",
			expDatabaseName: "dbname",
		},
		{
			dsn:             "postgresql://%2Fvar%2Flib%2Fpostgresql/dbname",
			expHost:         "",
			expPortPathOrID: "",
			expDatabaseName: "",
		},

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

func TestNewConnector(t *testing.T) {
	connector, err := NewConnector("client_encoding=")
	if err == nil {
		t.Error("error expected from invalid dsn")
	}
	if connector != nil {
		t.Error("nil connector expected from invalid dsn")
	}
	connector, err = NewConnector("host=localhost port=5432 user=postgres dbname=postgres password=docker sslmode=disable")
	if err != nil {
		t.Error("nil error expected from valid dsn", err)
	}
	if connector == nil {
		t.Error("non-nil connector expected from valid dsn")
	}
}
