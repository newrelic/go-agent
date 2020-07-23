// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	// 1. Instead of importing github.com/snowflakedb/gosnowflake, import the
	// nrsnowflake integration
	_ "github.com/newrelic/go-agent/v3/integrations/nrsnowflake"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	// 2. Instead of opening "snowflake", open "nrsnowflake"
	db, err := sql.Open("nrsnowflake", "root@/information_schema")
	if err != nil {
		panic(err)
	}

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Snowflake app"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if err != nil {
		log.Fatal(err)
	}
	app.WaitForConnection(5 * time.Second)
	defer app.Shutdown(5 * time.Second)

	txn := app.StartTransaction("snowflakeQuery")
	defer txn.End()
	// 3. Add the transaction to the context
	ctx := newrelic.NewContext(context.Background(), txn)

	// 4. Call methods on the db using the context
	row := db.QueryRowContext(ctx, "SELECT count(*) from tables")
	var count int
	row.Scan(&count)

	fmt.Println("number of tables in information_schema", count)
}
