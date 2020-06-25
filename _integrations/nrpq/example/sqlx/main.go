// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// An application that illustrates how to instrument jmoiron/sqlx with DatastoreSegments
//
// To run this example, be sure the environment varible NEW_RELIC_LICENSE_KEY
// is set to your license key.  Postgres must be running on the default port
// 5432 and have a user "foo" and a database "bar".
//
// Adding instrumentation for the SQLx package is easy.  It means you can
// make database calls without having to manually create DatastoreSegments.
// Setup can be done in two steps:
//
// Set up your driver
//
// If you are using one of our currently supported database drivers (see
// https://docs.newrelic.com/docs/agents/go-agent/get-started/go-agent-compatibility-requirements#frameworks),
// follow the instructions on installing the driver.
//
// As an example, for the `lib/pq` driver, you will use the newrelic
// integration's driver in place of the postgres driver.  If your code is using
// sqlx.Open with `lib/pq` like this:
//
//	import (
//		"github.com/jmoiron/sqlx"
//		_ "github.com/lib/pq"
//	)
//
//	func main() {
//		db, err := sqlx.Open("postgres", "user=pqgotest dbname=pqgotest sslmode=verify-full")
//	}
//
// Then change the side-effect import to the integration package, and open
// "nrpostgres" instead:
//
//	import (
//		"github.com/jmoiron/sqlx"
//		_ "github.com/newrelic/go-agent/_integrations/nrpq"
//	)
//
//	func main() {
//		db, err := sqlx.Open("nrpostgres", "user=pqgotest dbname=pqgotest sslmode=verify-full")
//	}
//
// If you are not using one of the supported database drivers, use the
// `InstrumentSQLDriver`
// (https://godoc.org/github.com/newrelic/go-agent#InstrumentSQLDriver) API.
// See
// https://github.com/newrelic/go-agent/blob/master/_integrations/nrmysql/nrmysql.go
// for a full example.
//
// Add context to your database calls
//
// Next, you must provide a context containing a newrelic.Transaction to all
// methods on sqlx.DB, sqlx.NamedStmt, sqlx.Stmt, and sqlx.Tx that make a
// database call.  For example, instead of the following:
//
//	err := db.Get(&jason, "SELECT * FROM person WHERE first_name=$1", "Jason")
//
// Do this:
//
//	ctx := newrelic.NewContext(context.Background(), txn)
//	err := db.GetContext(ctx, &jason, "SELECT * FROM person WHERE first_name=$1", "Jason")
//
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	newrelic "github.com/newrelic/go-agent"
	_ "github.com/newrelic/go-agent/_integrations/nrpq"
)

var schema = `
CREATE TABLE person (
    first_name text,
    last_name text,
    email text
)`

// Person is a person in the database
type Person struct {
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string
}

func mustGetEnv(key string) string {
	if val := os.Getenv(key); "" != val {
		return val
	}
	panic(fmt.Sprintf("environment variable %s unset", key))
}

func createApp() newrelic.Application {
	cfg := newrelic.NewConfig("SQLx", mustGetEnv("NEW_RELIC_LICENSE_KEY"))
	cfg.Logger = newrelic.NewDebugLogger(os.Stdout)
	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		log.Fatalln(err)
	}
	if err := app.WaitForConnection(5 * time.Second); nil != err {
		log.Fatalln(err)
	}
	return app
}

func main() {
	// Create application
	app := createApp()
	defer app.Shutdown(10 * time.Second)
	// Start a transaction
	txn := app.StartTransaction("main", nil, nil)
	defer txn.End()
	// Add transaction to context
	ctx := newrelic.NewContext(context.Background(), txn)

	// Connect to database using the "nrpostgres" driver
	db, err := sqlx.Connect("nrpostgres", "user=foo dbname=bar sslmode=disable")
	if err != nil {
		log.Fatalln(err)
	}

	// Create database table if it does not exist already
	// When the context is passed, DatastoreSegments will be created
	db.ExecContext(ctx, schema)

	// Add people to the database
	// When the context is passed, DatastoreSegments will be created
	tx := db.MustBegin()
	tx.MustExecContext(ctx, "INSERT INTO person (first_name, last_name, email) VALUES ($1, $2, $3)", "Jason", "Moiron", "jmoiron@jmoiron.net")
	tx.MustExecContext(ctx, "INSERT INTO person (first_name, last_name, email) VALUES ($1, $2, $3)", "John", "Doe", "johndoeDNE@gmail.net")
	tx.Commit()

	// Read from the database
	// When the context is passed, DatastoreSegments will be created
	people := []Person{}
	db.SelectContext(ctx, &people, "SELECT * FROM person ORDER BY first_name ASC")
	jason := Person{}
	db.GetContext(ctx, &jason, "SELECT * FROM person WHERE first_name=$1", "Jason")
}
