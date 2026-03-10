package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/newrelic/go-agent/v3/integrations/nrgocql"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	// Initialize New Relic
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Cassandra Example"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = app.WaitForConnection(10 * time.Second)
	if err != nil {
		log.Fatal(err)
	}
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "example"
	cluster.Consistency = gocql.Quorum
	cluster.ConnectTimeout = 15 * time.Second
	cluster.Timeout = 10 * time.Second

	// Set the New Relic query observer
	cluster.QueryObserver = nrgocql.NewQueryObserver[gocql.ObservedQuery](nil)

	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	// Start a New Relic transaction
	txn := app.StartTransaction("cassandra-operations")

	// Add transaction to context
	ctx := newrelic.NewContext(context.Background(), txn)

	// insert a tweet
	if err := session.Query(`INSERT INTO tweet (timeline, id, text) VALUES (?, ?, ?)`,
		"me", gocql.TimeUUID(), "hello world").ExecContext(ctx); err != nil {
		log.Fatal(err)
	}

	var id gocql.UUID
	var text string

	/* Search for a specific set of records whose 'timeline' column matches
	 * the value 'me'. The secondary index that we created earlier will be
	 * used for optimizing the search */
	if err := session.Query(`SELECT id, text FROM tweet WHERE timeline = ? LIMIT 1`,
		"me").Consistency(gocql.One).ScanContext(ctx, &id, &text); err != nil {
		log.Fatal(err)
	}
	txn.End()
	fmt.Println("Tweet:", id, text)
	fmt.Println()
	app.Shutdown(10 * time.Second)
}
