// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// +build go1.10

// Package nrsqlite3 instruments https://github.com/mattn/go-sqlite3.
//
// Use this package to instrument your SQLite calls without having to manually
// create DatastoreSegments.  This is done in a two step process:
//
// 1. Use this package's driver in place of the sqlite3 driver.
//
// If your code is using sql.Open like this:
//
//	import (
//		_ "github.com/mattn/go-sqlite3"
//	)
//
//	func main() {
//		db, err := sql.Open("sqlite3", "./foo.db")
//	}
//
// Then change the side-effect import to this package, and open "nrsqlite3" instead:
//
//	import (
//		_ "github.com/newrelic/go-agent/_integrations/nrsqlite3"
//	)
//
//	func main() {
//		db, err := sql.Open("nrsqlite3", "./foo.db")
//	}
//
// If you are registering a custom sqlite3 driver with special behavior then
// you must wrap your driver instance using nrsqlite3.InstrumentSQLDriver.  For
// example, if your code looks like this:
//
//	func main() {
//		sql.Register("sqlite3_with_extensions", &sqlite3.SQLiteDriver{
//			Extensions: []string{
//				"sqlite3_mod_regexp",
//			},
//		})
//		db, err := sql.Open("sqlite3_with_extensions", ":memory:")
//	}
//
// Then instrument the driver like this:
//
//	func main() {
//		sql.Register("sqlite3_with_extensions", nrsqlite3.InstrumentSQLDriver(&sqlite3.SQLiteDriver{
//			Extensions: []string{
//				"sqlite3_mod_regexp",
//			},
//		}))
//		db, err := sql.Open("sqlite3_with_extensions", ":memory:")
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
//
// A working example is shown here:
// https://github.com/newrelic/go-agent/tree/master/_integrations/nrsqlite3/example/main.go
package nrsqlite3

import (
	"database/sql"
	"database/sql/driver"
	"path/filepath"
	"strings"

	sqlite3 "github.com/mattn/go-sqlite3"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/sqlparse"
)

var (
	baseBuilder = newrelic.SQLDriverSegmentBuilder{
		BaseSegment: newrelic.DatastoreSegment{
			Product: newrelic.DatastoreSQLite,
		},
		ParseQuery: sqlparse.ParseQuery,
		ParseDSN:   parseDSN,
	}
)

func init() {
	sql.Register("nrsqlite3", InstrumentSQLDriver(&sqlite3.SQLiteDriver{}))
	internal.TrackUsage("integration", "driver", "sqlite3")
}

// InstrumentSQLDriver wraps an sqlite3.SQLiteDriver to add instrumentation.
// For example, if you are registering a custom SQLiteDriver like this:
//
//	sql.Register("sqlite3_with_extensions",
//		&sqlite3.SQLiteDriver{
//			Extensions: []string{
//				"sqlite3_mod_regexp",
//			},
//		})
//
// Then add instrumentation like this:
//
//	sql.Register("sqlite3_with_extensions",
//		nrsqlite3.InstrumentSQLDriver(&sqlite3.SQLiteDriver{
//			Extensions: []string{
//				"sqlite3_mod_regexp",
//			},
//		}))
//
func InstrumentSQLDriver(d *sqlite3.SQLiteDriver) driver.Driver {
	return newrelic.InstrumentSQLDriver(d, baseBuilder)
}

func getPortPathOrID(dsn string) (ppoid string) {
	ppoid = strings.Split(dsn, "?")[0]
	ppoid = strings.TrimPrefix(ppoid, "file:")

	if ":memory:" != ppoid && "" != ppoid {
		if abs, err := filepath.Abs(ppoid); nil == err {
			ppoid = abs
		}
	}

	return
}

// ParseDSN accepts a DSN string and sets the Host, PortPathOrID, and
// DatabaseName fields on a newrelic.DatastoreSegment.
func parseDSN(s *newrelic.DatastoreSegment, dsn string) {
	// See https://godoc.org/github.com/mattn/go-sqlite3#SQLiteDriver.Open
	s.Host = "localhost"
	s.PortPathOrID = getPortPathOrID(dsn)
	s.DatabaseName = ""
}
