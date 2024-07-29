// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build go1.10
// +build go1.10

// Package nrmssql instruments github.com/microsoft/go-mssqldb.
//
// Use this package to instrument your MSSQL calls without having to manually
// create DatastoreSegments.  This is done in a two step process:
//
// 1. Use this package's driver in place of the mssql driver.
//
// If your code is using sql.Open like this:
//
//	import (
//		_ "github.com/microsoft/go-mssqldb"
//	)
//
//	func main() {
//		db, err := sql.Open("mssql", "server=localhost;user id=sa;database=master;app name=MyAppName")
//	}
//
// Then change the side-effect import to this package, and open "nrmssql" instead:
//
//	import (
//		_ "github.com/newrelic/go-agent/v3/integrations/nrmssql"
//	)
//
//	func main() {
//		db, err := sql.Open("nrmssql", "server=localhost;user id=sa;database=master;app name=MyAppName")
//	}
//
// 2. Provide a context containing a newrelic.Transaction to all exec and query
// methods on sql.DB, sql.Conn, sql.Tx, and sql.Stmt.  This requires using the
// context methods ExecContext, QueryContext, and QueryRowContext in place of
// Exec, Query, and QueryRow respectively.  For example, instead of the
// following:
//
//	row := db.QueryRow("SELECT count(*) from tables")
//
// Do this:
//
//	ctx := newrelic.NewContext(context.Background(), txn)
//	row := db.QueryRowContext(ctx, "SELECT count(*) from tables")
package nrmssql

import (
	"database/sql"
	"fmt"

	mssql "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/msdsn"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/sqlparse"
)

var (
	baseBuilder = newrelic.SQLDriverSegmentBuilder{
		BaseSegment: newrelic.DatastoreSegment{
			Product: newrelic.DatastoreMSSQL,
		},
		ParseQuery: sqlparse.ParseQuery,
		ParseDSN:   parseDSN,
	}
)

func init() {
	sql.Register("nrmssql", newrelic.InstrumentSQLDriver(&mssql.Driver{}, baseBuilder))
	internal.TrackUsage("integration", "driver", "mssql")
}

func parseDSN(s *newrelic.DatastoreSegment, dsn string) {
	cfg, err := msdsn.Parse(dsn)
	if nil != err {
		return
	}

	s.DatabaseName = cfg.Database
	s.Host = cfg.Host
	s.PortPathOrID = fmt.Sprint(cfg.Port)
}
