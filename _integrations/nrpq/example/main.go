// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	newrelic "github.com/newrelic/go-agent"
	_ "github.com/newrelic/go-agent/_integrations/nrpq"
)

func mustGetEnv(key string) string {
	if val := os.Getenv(key); "" != val {
		return val
	}
	panic(fmt.Sprintf("environment variable %s unset", key))
}

func main() {
	// docker run --rm -e POSTGRES_PASSWORD=docker -p 5432:5432 postgres
	db, err := sql.Open("nrpostgres", "host=localhost port=5432 user=postgres dbname=postgres password=docker sslmode=disable")
	if err != nil {
		panic(err)
	}

	cfg := newrelic.NewConfig("PostgreSQL App", mustGetEnv("NEW_RELIC_LICENSE_KEY"))
	cfg.Logger = newrelic.NewDebugLogger(os.Stdout)
	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		panic(err)
	}
	app.WaitForConnection(5 * time.Second)
	txn := app.StartTransaction("postgresQuery", nil, nil)

	ctx := newrelic.NewContext(context.Background(), txn)
	row := db.QueryRowContext(ctx, "SELECT count(*) FROM pg_catalog.pg_tables")
	var count int
	row.Scan(&count)

	txn.End()
	app.Shutdown(5 * time.Second)

	fmt.Println("number of entries in pg_catalog.pg_tables", count)
}
