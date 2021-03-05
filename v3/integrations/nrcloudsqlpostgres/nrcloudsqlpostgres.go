// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// +build go1.13

// Package nrcloudsqlpostgres instruments https://github.com/GoogleCloudPlatform/cloudsql-proxy.
//
// Use this package to instrument your PostgreSQL calls without having to manually
// create DatastoreSegments.  This is done in a two step process:
//
// 1. Use this package's driver in place of the postgres driver.
//
// If your code is using sql.Open like this:
//
//	import (
//		_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres"
//	)
//
//	func main() {
//		db, err := sql.Open("cloudsqlpostgres", "host=project:region:instance user=postgres dbname=postgres password=password sslmode=disable")
//	}
//
// Then change the side-effect import to this package, and open "nrcloudsqlpostgres" instead:
//
//	import (
//		_ "salesloft.com/costello/core/newrelic/driver/nrcloudsqlproxy"
//	)
//
//	func main() {
//		db, err := sql.Open("nrcloudsqlpostgres", "host=project:region:instance user=postgres dbname=postgres password=password sslmode=disable")
//	}
//
// 2. Provide a context containing a newrelic.Transaction to all exec and query
// methods on sql.DB, sql.Conn, and sql.Tx.  This requires using the
// context methods ExecContext, QueryContext, and QueryRowContext in place of
// Exec, Query, and QueryRow respectively.  For example, instead of the
// following:
//
//	row := db.QueryRow("SELECT count(*) FROM pg_catalog.pg_tables")
//
// Do this:
//
//	ctx := newrelic.NewContext(context.Background(), txn)
//	row := db.QueryRowContext(ctx, "SELECT count(*) FROM pg_catalog.pg_tables")
//
// Unfortunately, sql.Stmt exec and query calls are not supported since pq.stmt
// does not have ExecContext and QueryContext methods (as of June 2019, see
// https://github.com/lib/pq/pull/768).

package nrcloudsqlpostgres

import (
	"database/sql"
	"os"
	"regexp"
	"strings"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/sqlparse"

	"github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres"
)

var (
	baseBuilder = newrelic.SQLDriverSegmentBuilder{
		BaseSegment: newrelic.DatastoreSegment{
			Product: newrelic.DatastorePostgres,
		},
		ParseQuery: sqlparse.ParseQuery,
		ParseDSN:   parseDSN(os.Getenv),
	}
)

func init() {
	sql.Register("nrcloudsqlpostgres", newrelic.InstrumentSQLDriver(&postgres.Driver{}, baseBuilder))
	internal.TrackUsage("integration", "driver", "cloudsqlpostgres")
}

var dsnSplit = regexp.MustCompile(`(\w+)\s*=\s*('[^=]*'|[^'\s]+)`)

func getFirstHost(value string) string {
	host := strings.SplitN(value, ",", 2)[0]
	host = strings.Trim(host, "[]")
	return host
}

func parseDSN(getenv func(string) string) func(*newrelic.DatastoreSegment, string) {
	return func(s *newrelic.DatastoreSegment, dsn string) {
		host := getenv("PGHOST")
		hostaddr := ""
		ppoid := getenv("PGPORT")
		dbname := getenv("PGDATABASE")

		for _, split := range dsnSplit.FindAllStringSubmatch(dsn, -1) {
			if len(split) != 3 {
				continue
			}
			key := split[1]
			value := strings.Trim(split[2], `'`)

			switch key {
			case "dbname":
				dbname = value
			case "host":
				host = getFirstHost(value)
			case "hostaddr":
				hostaddr = getFirstHost(value)
			case "port":
				ppoid = strings.SplitN(value, ",", 2)[0]
			}
		}

		if "" != hostaddr {
			host = hostaddr
		} else if "" == host {
			host = "localhost"
		}
		if "" == ppoid {
			ppoid = "5432"
		}

		s.Host = host
		s.PortPathOrID = ppoid
		s.DatabaseName = dbname
	}
}
