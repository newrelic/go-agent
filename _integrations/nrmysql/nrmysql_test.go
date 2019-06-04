package nrmysql

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
	}{
		// examples from https://github.com/go-sql-driver/mysql README
		{
			dsn:             "user@unix(/path/to/socket)/dbname",
			expHost:         "localhost",
			expPortPathOrID: "/path/to/socket",
			expDatabaseName: "dbname",
		},
		{
			dsn:             "root:pw@unix(/tmp/mysql.sock)/myDatabase?loc=Local",
			expHost:         "localhost",
			expPortPathOrID: "/tmp/mysql.sock",
			expDatabaseName: "myDatabase",
		},
		{
			dsn:             "user:password@tcp(localhost:5555)/dbname?tls=skip-verify&autocommit=true",
			expHost:         "localhost",
			expPortPathOrID: "5555",
			expDatabaseName: "dbname",
		},
		{
			dsn:             "user:password@/dbname?sql_mode=TRADITIONAL",
			expHost:         "127.0.0.1",
			expPortPathOrID: "3306",
			expDatabaseName: "dbname",
		},
		{
			dsn:             "user:password@tcp([de:ad:be:ef::ca:fe]:80)/dbname?timeout=90s&collation=utf8mb4_unicode_ci",
			expHost:         "de:ad:be:ef::ca:fe",
			expPortPathOrID: "80",
			expDatabaseName: "dbname",
		},
		{
			dsn:             "id:password@tcp(your-amazonaws-uri.com:3306)/dbname",
			expHost:         "your-amazonaws-uri.com",
			expPortPathOrID: "3306",
			expDatabaseName: "dbname",
		},
		{
			dsn:             "user@cloudsql(project-id:instance-name)/dbname",
			expHost:         "project-id:instance-name",
			expPortPathOrID: "",
			expDatabaseName: "dbname",
		},
		{
			dsn:             "user@cloudsql(project-id:regionname:instance-name)/dbname",
			expHost:         "project-id:regionname:instance-name",
			expPortPathOrID: "",
			expDatabaseName: "dbname",
		},
		{
			dsn:             "user:password@tcp/dbname?charset=utf8mb4,utf8&sys_var=esc%40ped",
			expHost:         "127.0.0.1",
			expPortPathOrID: "3306",
			expDatabaseName: "dbname",
		},
		{
			dsn:             "user:password@/dbname",
			expHost:         "127.0.0.1",
			expPortPathOrID: "3306",
			expDatabaseName: "dbname",
		},
		{
			dsn:             "user:password@/",
			expHost:         "127.0.0.1",
			expPortPathOrID: "3306",
			expDatabaseName: "",
		},
	}

	for _, test := range testcases {
		s := &newrelic.DatastoreSegment{}
		ParseDSN(s, test.dsn)
		if test.expHost != s.Host {
			t.Errorf(`incorrect host, expected="%s", actual="%s"`, test.expHost, s.Host)
		}
		if test.expPortPathOrID != s.PortPathOrID {
			t.Errorf(`incorrect port path or id, expected="%s", actual="%s"`, test.expPortPathOrID, s.PortPathOrID)
		}
		if test.expDatabaseName != s.DatabaseName {
			t.Errorf(`incorrect port path or id, expected="%s", actual="%s"`, test.expDatabaseName, s.DatabaseName)
		}
	}
}
