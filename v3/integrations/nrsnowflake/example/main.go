package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/newrelic/go-agent/v3/integrations/nrsnowflake"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func mustGetEnv(key string) string {
	if val := os.Getenv(key); "" != val {
		return val
	}
	panic(fmt.Sprintf("environment variable %s unset", key))
}

func main() {
	db, err := sql.Open("nrsnowflake", "root@/information_schema")
	if err != nil {
		panic(err)
	}

	app, err := newrelic.NewApplication(newrelic.ConfigAppName("Snowflake app"), newrelic.ConfigLicense(mustGetEnv("NEW_RELIC_LICENSE_KEY")), newrelic.NewDebugLogger(os.Stdout))
	if err != nil {
		log.Fatal(err)
	}

	app.WaitForConnection(5 * time.Second)
	txn := app.StartTransaction("snowflakeQuery")

	ctx := newrelic.NewContext(context.Background(), txn)
	row := db.QueryRowContext(ctx, "SELECT count(*) from tables")
	var count int
	row.Scan(&count)

	txn.End()
	app.Shutdown(5 * time.Second)

	fmt.Println("number of tables in information_schema", count)
}
