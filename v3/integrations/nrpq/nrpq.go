// +build go1.10

// Package nrpq instruments https://github.com/lib/pq.
//
// Use this package to instrument your PostgreSQL calls without having to manually
// create DatastoreSegments.  This is done in a two step process:
//
// 1. Use this package's driver in place of the postgres driver.
//
// If your code is using sql.Open like this:
//
//	import (
//		_ "github.com/lib/pq"
//	)
//
//	func main() {
//		db, err := sql.Open("postgres", "user=pqgotest dbname=pqgotest sslmode=verify-full")
//	}
//
// Then change the side-effect import to this package, and open "nrpostgres" instead:
//
//	import (
//		_ "github.com/newrelic/go-agent/v3/integrations/nrpq"
//	)
//
//	func main() {
//		db, err := sql.Open("nrpostgres", "user=pqgotest dbname=pqgotest sslmode=verify-full")
//	}
//
// If your code is using pq.NewConnector, simply use nrpq.NewConnector
// instead.
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
//
// A working example is shown here:
// https://github.com/newrelic/go-agent/tree/master/v3/integrations/nrpq/example/main.go
package nrpq

import (
	"database/sql"
	"database/sql/driver"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/lib/pq"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/sqlparse"
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

// NewConnector can be used in place of pq.NewConnector to get an instrumented
// PostgreSQL connector.
func NewConnector(dsn string) (driver.Connector, error) {
	connector, err := pq.NewConnector(dsn)
	if nil != err || nil == connector {
		// Return nil rather than 'connector' since a nil pointer would
		// be returned as a non-nil driver.Connector.
		return nil, err
	}
	bld := baseBuilder
	bld.ParseDSN(&bld.BaseSegment, dsn)
	return newrelic.InstrumentSQLConnector(connector, bld), nil
}

func init() {
	sql.Register("nrpostgres", newrelic.InstrumentSQLDriver(&pq.Driver{}, baseBuilder))
	internal.TrackUsage("integration", "driver", "postgres")
}

var dsnSplit = regexp.MustCompile(`(\w+)\s*=\s*('[^=]*'|[^'\s]+)`)

func getFirstHost(value string) string {
	host := strings.SplitN(value, ",", 2)[0]
	host = strings.Trim(host, "[]")
	return host
}

func parseDSN(getenv func(string) string) func(*newrelic.DatastoreSegment, string) {
	return func(s *newrelic.DatastoreSegment, dsn string) {
		if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
			var err error
			dsn, err = pq.ParseURL(dsn)
			if nil != err {
				return
			}
		}

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
		if strings.HasPrefix(host, "/") {
			// this is a unix socket
			ppoid = path.Join(host, ".s.PGSQL."+ppoid)
			host = "localhost"
		}

		s.Host = host
		s.PortPathOrID = ppoid
		s.DatabaseName = dbname
	}
}
