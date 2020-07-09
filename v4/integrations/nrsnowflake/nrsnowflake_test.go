// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrsnowflake

import (
	"testing"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/snowflakedb/gosnowflake"
)

func TestParseDSN(t *testing.T) {
	testcases := []struct {
		dsn             string
		expHost         string
		expPortPathOrID string
		expDatabaseName string
	}{
		{
			dsn:             "user:password@account/database/schema",
			expHost:         "account.snowflakecomputing.com",
			expPortPathOrID: "443",
			expDatabaseName: "database",
		},
		{
			dsn:             "user:password@host:123/database/schema?account=user_account",
			expHost:         "host",
			expPortPathOrID: "123",
			expDatabaseName: "database",
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
		cfgHost         string
		cfgPort         int
		cfgDBName       string
		expHost         string
		expPortPathOrID string
		expDatabaseName string
	}{
		{
			cfgDBName:       "mydb",
			expDatabaseName: "mydb",
			expPortPathOrID: "",
		},
		{
			cfgHost:         "unixgram",
			cfgPort:         123,
			expHost:         "unixgram",
			expPortPathOrID: "123",
		},
	}

	for _, test := range testcases {
		s := &newrelic.DatastoreSegment{}
		cfg := &gosnowflake.Config{
			Host:     test.cfgHost,
			Port:     test.cfgPort,
			Database: test.cfgDBName,
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
