// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/newrelic/go-agent/v3/integrations/nrsqlite3"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	db, err := sql.Open("nrsqlite3", ":memory:")
	if err != nil {
		panic(err)
	}

	defer db.Close()

	db.Exec("CREATE TABLE zaps ( zap_num INTEGER )")
	db.Exec("INSERT INTO zaps (zap_num) VALUES (22)")

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("SQLite App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if err != nil {
		panic(err)
	}

	app.WaitForConnection(10 * time.Second)
	txn := app.StartTransaction("sqliteQuery")
	ctx := newrelic.NewContext(context.Background(), txn)
	row := db.QueryRowContext(ctx, "SELECT count(*) from zaps")
	var count int
	row.Scan(&count)
	txn.End()

	txn = app.StartTransaction("CustomSQLQuery")
	s := newrelic.DatastoreSegment{
		Product:            newrelic.DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		QueryParameters: map[string]interface{}{
			"name": "Dracula",
			"age":  439,
		},
		Host:         "mysql-server-1",
		PortPathOrID: "3306",
		DatabaseName: "my_database",
	}
	s.StartTime = txn.StartSegmentNow()
	// ... do the operation
	s.End()
	txn.End()

	app.Shutdown(5 * time.Second)
	fmt.Printf("number of elements in table: %v\n", count)
}
