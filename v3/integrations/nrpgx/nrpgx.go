// Copyright 2021 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// +build go1.10

// Package nrpgx instruments https://github.com/jackc/pgx/v4.
//
// Use this package to instrument your PostgreSQL calls using the pgx
// library.
//
// USING WITH PGX AS A DATABASE/SQL DRIVER
//
// The pgx library may be used as a database/sql driver rather than making
// direct calls into pgx itself. In this scenario, just use the nrpgx integration
// in place of the pgx driver. In other words,
// if your code without New Relic's agent looks like this:
//
//	import (
//      "database/sql"
//		_ "github.com/jackc/pgx/v4/stdlib"
//	)
//
//	func main() {
//		db, err := sql.Open("pgx", "user=pqgotest dbname=pqgotest sslmode=verify-full")
//	}
//
// Then change the side-effect import to this package, and open "nrpgx" instead:
//
//	import (
//      "database/sql"
//
//		_ "github.com/newrelic/go-agent/v3/integrations/nrpgx"
//	)
//
//	func main() {
//		db, err := sql.Open("nrpgx", "user=pqgotest dbname=pqgotest sslmode=verify-full")
//	}
//
// Next, provide a context containing a newrelic.Transaction to all exec and query
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
// A working example is shown here:
// https://github.com/newrelic/go-agent/tree/master/v3/integrations/nrpgx/example/sql_compat/main.go
//
//
// USING WITH DIRECT PGX CALLS WITHOUT DATABASE/SQL
//
// This mode of operation is not supported by the nrpgx integration at this time.
//
package nrpgx

import (
	"database/sql"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v4/stdlib"
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

func init() {
	sql.Register("nrpgx", newrelic.InstrumentSQLDriver(&stdlib.Driver{}, baseBuilder))
	internal.TrackUsage("integration", "driver", "nrpgx")
}

/*
 * The general format for these is:
 *   postgres[ql]://[user[:pass]@][host][:port][,...][/db][?params]
 *
 *   pgx recognizes:
 *     sql.Open("pgx", "hostname=x host_port=x username=x password=x databasename=x sslmode=x")
 *   or
 *     sql.Open("pgx", URI)
 *   or
 *     driverConfig := stdlib.DriverConfig{...}
 *     sql.RegisterDriverConfig(&driverConfig)
 *     sql.Open("pgx", driverConfig.ConnectionString(URI))
 *
 * params may include
 *   host=name (see below)
 *   hostaddr=IPv4/IPv6 address
 *   port=number
 *   dbname=name
 *   user=name
 *   password=string
 *   passfile=path
 *   connect_timeout=sec
 *   client_encoding=scheme|"auto"
 *   options=command-line opts
 *   application_name=name
 *   fallback_appliction_name=name
 *   keepalives=0|1
 *   keepalives_idle=sec
 *   keepalives_interval=sec
 *   keepalives_count=n
 *   tcp_user_timeout=ms
 *   tty=name
 *   replication=mode
 *   gssencmode=mode
 *   sslmode=disable|allow|prefer|require|verify-ca|verify-full
 *   requiressl=bool
 *   sslcompression=bool
 *   sslcert=file
 *   sslkey=file
 *   sslrootcert=file
 *   sslcrl=file
 *   requirepeer=user
 *   krbsrvname=name
 *   gsslib=name
 *   service=name
 *   target_session_attrs=value
 *   ssl=true  is equivalent to sslmode=require
 *
 * may include %-encoded values
 * host may be a DNS name, IP4 address, or IP6 address in square brackets, or may be
 * a pathname for a UNIX domain socket (but since slash is reserved in URI syntax,
 * this may need to be put in the params list)
 *  (Note that the IP6 address contains colons)
 * multiple hosts may be given as a comma-separated list
 *
 * pgx provides
 *  ParseConnectionString(string) -> ConnConfig, error	(URI or DSN)
 *  ParseDSN(string) -> ConnConfig, error				(DSN only)
 *  ParseURI(string) -> ConnConfig, error				(URI only)
 *
 * which we'll use to parse out all of the above and avoid second-guessing
 * anything or duplicating effort.
 */

//
// parsePort takes a string representation which purports to be a TCP port number
// or a comma-separated list of port numbers, and returns the port number as a
// uint16 value and as a string. If a list was given, the returned values are for
// the first item in the list, discarding the others.
//
// If the string can't be understood as an unsigned integer value, then 0 is
// returned as the uint16 value.
//
func parsePort(p string) (uint16, string) {
	if p == "" {
		return 0, ""
	}

	np, err := strconv.ParseUint(p, 10, 16)
	if err != nil {
		// Maybe multiple values were specified?
		// (The code could be more concise if we did more processing up front,
		// but this avoids needless work if a single number was given, so should
		// be more runtime efficient.)
		if comma := strings.IndexRune(p, ','); comma >= 0 {
			p = p[0:comma]
			np, err = strconv.ParseUint(p, 10, 16)
			if err != nil {
				// even the first one isn't a real number
				np = 0
			}
		} else {
			// Nope, it's just something we can't grok
			np = 0
		}
	}
	return uint16(np), p
}

//
// ip6HostPort is a regular expression to parse IPv6 hostnames (which must be in
// square brackets in the connection strings we're working with). The hostname may
// optionally be followed by a colon and a TCP port number.
//
// Any text after the hostname (and port, if any), is ignored.
//
var ip6HostPort = regexp.MustCompile(`^\[([0-9a-fA-F:]*)\](:([0-9]+))?`)

//
// fullIp6ConnectPattern and fullConnectPattern are regular expressions which
// match a postgres URI connection string using IPv6 addresses, or other forms,
// respectively.
//
var fullIp6ConnectPattern = regexp.MustCompile(
	//                     user     password    host                 port                dbname      params
	//                     __1__    __2__     _________3________     __4__               __5__       __6_
	`^postgres(?:ql)?://(?:(.*?)(?::(.*?))?@)?(\[[0-9a-fA-F:]+\])(?::(.*?))?(?:,.*?)*(?:/(.*?))?(?:\?(.*))?$`)
var fullConnectPattern = regexp.MustCompile(
	//                     user     password  host                 port                dbname      params
	//                     __1__    __2__     ________3________    __4__               __5__       __6_
	`^postgres(?:ql)?://(?:(.*?)(?::(.*?))?@)?([a-zA-Z0-9_.-]+)(?::(.*?))?(?:,.*?)*(?:/(.*?))?(?:\?(.*))?$`)

//
// fullParamPattern is a regular expression to match the key=value pairs of a
// parameterized DSN string.
//
var fullParamPattern = regexp.MustCompile(
	`(\w+)\s*=\s*('[^=]*'|[^'\s]+)`)

//
// parseDSN returns a function which will set datastore segment attributes to show
// the database, host, and port as extracted from a supplied DSN string.
//
func parseDSN(getenv func(string) string) func(*newrelic.DatastoreSegment, string) {
	return func(s *newrelic.DatastoreSegment, dsn string) {

		cc, err := pgx.ParseConnectionString(dsn)
		if err != nil {
			// the connection string is invalid

			// Sometimes we've found that pgx.ParseConnectionString doesn't recognize
			// all patterns so if that call failed, we'll do a little pattern matching
			// of our own and see if we can figure it out.
			cc = pgx.ConnConfig{}
			conn := fullIp6ConnectPattern.FindStringSubmatch(dsn)
			if conn == nil {
				conn = fullConnectPattern.FindStringSubmatch(dsn)
				if conn == nil {
					// maybe it's a parameterized string that ParseConnectionString didn't like.
					for _, par := range fullParamPattern.FindAllStringSubmatch(dsn, -1) {
						if len(par) != 3 {
							continue
						}
						v := strings.Trim(par[2], "'")
						switch par[1] {
						case "dbname":
							cc.Database = v
						case "host":
							// don't overwrite if hostaddr already put a value here
							if cc.Host == "" {
								cc.Host = v
							}
						case "hostaddr":
							cc.Host = v
						case "port":
							cc.Port, _ = parsePort(v)
						}
					}
					if cc.Database == "" && cc.Host == "" && cc.Port == 0 {
						// Ok, we give up.
						return
					}
				}
			}
			if conn != nil {
				cc.Host = conn[3]
				cc.Database = conn[5]
				cc.Port, _ = parsePort(conn[4])
			}
		}

		// default to the environment variables in case
		// we can't find them in the connection string
		var ppoid string

		if p, ok := cc.RuntimeParams["port"]; ok {
			cc.Port, ppoid = parsePort(p)
		}

		if cc.Port == 0 {
			cc.Port, ppoid = parsePort(getenv("PGPORT"))
		} else {
			ppoid = strconv.Itoa(int(cc.Port))
		}

		// explicit hostaddr=xxx overrides the hostname
		if ha, ok := cc.RuntimeParams["hostaddr"]; ok {
			cc.Host = ha
		}

		if cc.Host == "" {
			cc.Host = getenv("PGHOST")
		}

		// The host string could have multiple comma-separated hosts,
		// which could have an attached ":port" at the end, but also
		// note that we can't get too anxious to jump on any colons in
		// the hostname string because the hostname could also be an IPv6
		// address with optional port, as in
		// "[2001:db8:3333:4444:5555:6666:7777:8888]:5432", so we need
		// to look for that explicitly.
		cc.Host = strings.SplitN(cc.Host, ",", 2)[0]

		ip6parts := ip6HostPort.FindStringSubmatch(cc.Host)
		if ip6parts != nil {
			cc.Host = ip6parts[1]
			if ip6parts[2] != "" {
				cc.Port, ppoid = parsePort(ip6parts[2])
			}
		} else {
			if colon := strings.IndexRune(cc.Host, ':'); colon >= 0 {
				// This host had an explicit port number attached to it.
				// Use that in preference to what's in cc.Port.
				if colon+1 < len(cc.Host) {
					cc.Port, ppoid = parsePort(cc.Host[colon+1:])
				}
				cc.Host = cc.Host[0:colon]
			}
		}

		if cc.Database == "" {
			cc.Database = getenv("PGDATABASE")
		}

		// fill in the Postgres defaults explicitly
		if cc.Host == "" {
			cc.Host = "localhost"
		}
		if cc.Port == 0 {
			cc.Port, ppoid = parsePort("5432")
		}

		// Unix sockets are handled a little differently
		if strings.HasPrefix(cc.Host, "/") {
			ppoid = path.Join(cc.Host, ".s.PGSQL."+ppoid)
			cc.Host = "localhost"
		}

		s.Host = cc.Host
		s.PortPathOrID = ppoid
		s.DatabaseName = cc.Database
	}
}
