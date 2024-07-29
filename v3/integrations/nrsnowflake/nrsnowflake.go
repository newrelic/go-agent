// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrsnowflake instruments github.com/snowflakedb/gosnowflake
//
// Use this package to instrument your Snowflake calls without having to manually
// create DatastoreSegments.  This is done in a two step process:
//
// 1. Use this package's driver in place of the snowflake driver.
//
// If your code is using sql.Open like this:
//
//	import (
//		_ "github.com/snowflakedb/gosnowflake"
//	)
//
//	func main() {
//		db, err := sql.Open("snowflake", "user@unix(/path/to/socket)/dbname")
//	}
//
// Then change the side-effect import to this package, and open "nrsnowflake" instead:
//
//	import (
//		_ "github.com/newrelic/go-agent/v3/integrations/nrsnowflake"
//	)
//
//	func main() {
//		db, err := sql.Open("nrsnowflake", "user@unix(/path/to/socket)/dbname")
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
// https://github.com/newrelic/go-agent/tree/master/v3/integrations/nrsnowflake/example/main.go
package nrsnowflake

import (
	"database/sql"
	"strconv"

	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/sqlparse"
	"github.com/snowflakedb/gosnowflake"
)

var (
	baseBuilder = newrelic.SQLDriverSegmentBuilder{
		BaseSegment: newrelic.DatastoreSegment{
			Product: newrelic.DatastoreSnowflake,
		},
		ParseQuery: sqlparse.ParseQuery,
		ParseDSN:   parseDSN,
	}
)

func init() {
	sql.Register("nrsnowflake", newrelic.InstrumentSQLDriver(gosnowflake.SnowflakeDriver{}, baseBuilder))
	internal.TrackUsage("integration", "driver", "snowflake")
}

func parseDSN(s *newrelic.DatastoreSegment, dsn string) {
	cfg, err := gosnowflake.ParseDSN(dsn)
	if nil != err {
		return
	}
	parseConfig(s, cfg)
}

func parseConfig(s *newrelic.DatastoreSegment, cfg *gosnowflake.Config) {
	sPort := strconv.Itoa(cfg.Port)
	if cfg.Port == 0 {
		sPort = ""
	}

	s.DatabaseName = cfg.Database
	s.Host = cfg.Host
	s.PortPathOrID = sPort
}
