package nrmssql

import (
	"github.com/newrelic/go-agent/v3/newrelic"
	"testing"
)

func TestParseDSN(t *testing.T) {
	testcases := []struct {
		dsn             string
		expHost         string
		expPortPathOrID string
		expDatabaseName string
	}{
		// examples from https://github.com/denisenkom/go-mssqldb/blob/master/msdsn/conn_str_test.go
		{
			dsn:             "server=server\\instance;database=testdb;user id=tester;password=pwd",
			expHost:         "server",
			expPortPathOrID: "0",
			expDatabaseName: "testdb",
		},
		{
			dsn:             "server=(local)",
			expHost:         "localhost",
			expPortPathOrID: "0",
			expDatabaseName: "",
		},
		{
			dsn:             "sqlserver://someuser@somehost?connection+timeout=30",
			expHost:         "somehost",
			expPortPathOrID: "0",
			expDatabaseName: "",
		},
		{
			dsn:             "sqlserver://someuser:foo%3A%2F%5C%21~%40;bar@somehost:1434?connection+timeout=30",
			expHost:         "somehost",
			expPortPathOrID: "1434",
			expDatabaseName: "",
		},
		{
			dsn:             "Server=mssql.test.local; Initial Catalog=test; User ID=xxxxxxx; Password=abccxxxxxx;",
			expHost:         "mssql.test.local",
			expPortPathOrID: "0",
			expDatabaseName: "test",
		},
		{
			dsn:             "sport=invalid",
			expHost:         "localhost",
			expPortPathOrID: "0",
			expDatabaseName: "",
		},
	}

	for _, test := range testcases {
		s := &newrelic.DatastoreSegment{}
		parseDSN(s, test.dsn)
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
