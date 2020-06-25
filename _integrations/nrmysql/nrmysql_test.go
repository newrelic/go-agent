// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrmysql

import (
	"testing"

	"github.com/go-sql-driver/mysql"
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
		{
			dsn:             "this is not a dsn",
			expHost:         "",
			expPortPathOrID: "",
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

func TestParseConfig(t *testing.T) {
	testcases := []struct {
		cfgNet          string
		cfgAddr         string
		cfgDBName       string
		expHost         string
		expPortPathOrID string
		expDatabaseName string
	}{
		{
			cfgDBName:       "mydb",
			expDatabaseName: "mydb",
		},
		{
			cfgNet:          "unixgram",
			cfgAddr:         "/path/to/my/sock",
			expHost:         "localhost",
			expPortPathOrID: "/path/to/my/sock",
		},
		{
			cfgNet:          "unixpacket",
			cfgAddr:         "/path/to/my/sock",
			expHost:         "localhost",
			expPortPathOrID: "/path/to/my/sock",
		},
		{
			cfgNet:          "udp",
			cfgAddr:         "[fe80::1%lo0]:53",
			expHost:         "fe80::1%lo0",
			expPortPathOrID: "53",
		},
		{
			cfgNet:          "tcp",
			cfgAddr:         ":80",
			expHost:         "localhost",
			expPortPathOrID: "80",
		},
		{
			cfgNet:          "ip4:1",
			cfgAddr:         "192.0.2.1",
			expHost:         "192.0.2.1",
			expPortPathOrID: "",
		},
		{
			cfgNet:          "tcp6",
			cfgAddr:         "golang.org:http",
			expHost:         "golang.org",
			expPortPathOrID: "http",
		},
		{
			cfgNet:          "ip6:ipv6-icmp",
			cfgAddr:         "2001:db8::1",
			expHost:         "2001:db8::1",
			expPortPathOrID: "",
		},
	}

	for _, test := range testcases {
		s := &newrelic.DatastoreSegment{}
		cfg := &mysql.Config{
			Net:    test.cfgNet,
			Addr:   test.cfgAddr,
			DBName: test.cfgDBName,
		}
		parseConfig(s, cfg)
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
