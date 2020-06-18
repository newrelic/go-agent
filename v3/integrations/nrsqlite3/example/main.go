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
	if nil != err {
		panic(err)
	}
	app.WaitForConnection(5 * time.Second)
	txn := app.StartTransaction("sqliteQuery")

	ctx := newrelic.NewContext(context.Background(), txn)
	row := db.QueryRowContext(ctx, "SELECT count(*) from zaps")
	var count int
	row.Scan(&count)

	txn.End()
	app.Shutdown(5 * time.Second)

	fmt.Println("number of entries in table", count)
}
