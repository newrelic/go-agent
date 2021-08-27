// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//
// To run this example, be sure the environment variable NEW_RELIC_LICENSE_KEY
// is set to your license key. Postgres must be running on the default port
// 5432 on localhost, and have a password "docker". An easy (albeit insecure)
// way to test this is to issue the following command to run a postgres database
// in a docker container:
//    docker run --rm -e POSTGRES_PASSWORD=docker -p 5432:5432 postgres
//
// Run that in the background or in a separate window, and then run this program
// to access that database.
//

package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/newrelic/go-agent/v3/integrations/nrpq"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	// docker run --rm -e POSTGRES_PASSWORD=docker -p 5432:5432 postgres
	db, err := sql.Open("nrpostgres", "host=localhost port=5432 user=postgres dbname=postgres password=docker sslmode=disable")
	if err != nil {
		panic(err)
	}

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("PostgreSQL App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if nil != err {
		panic(err)
	}
	app.WaitForConnection(5 * time.Second)
	txn := app.StartTransaction("postgresQuery")

	ctx := newrelic.NewContext(context.Background(), txn)
	row := db.QueryRowContext(ctx, "SELECT count(*) FROM pg_catalog.pg_tables")
	var count int
	row.Scan(&count)

	txn.End()
	app.Shutdown(5 * time.Second)

	fmt.Println("number of entries in pg_catalog.pg_tables", count)
}
