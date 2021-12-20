package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/newrelic/go-agent/v3/integrations/nrmssql"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	// Set up a local ms sql docker

	db, err := sql.Open("nrmssql", "server=localhost;user id=sa;database=master;app name=MyAppName")
	if nil != err {
		panic(err)
	}

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("MSSQL App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if nil != err {
		panic(err)
	}
	app.WaitForConnection(5 * time.Second)
	txn := app.StartTransaction("mssqlQuery")

	ctx := newrelic.NewContext(context.Background(), txn)
	row := db.QueryRowContext(ctx, "SELECT count(*) from tables")
	var count int
	row.Scan(&count)

	txn.End()
	app.Shutdown(5 * time.Second)

	fmt.Println("number of tables in information_schema", count)
}
