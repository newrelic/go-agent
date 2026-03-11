package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	gocql "github.com/gocql/gocql"
	"github.com/newrelic/go-agent/v3/integrations/nrgocql"
	gocqlx "github.com/scylladb/gocqlx/v3"
	"github.com/scylladb/gocqlx/v3/qb"
	"github.com/scylladb/gocqlx/v3/table"

	"github.com/newrelic/go-agent/v3/newrelic"
)

type Tweet struct {
	Timeline string
	ID       gocql.UUID
	Text     string
}

func main() {
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
	cluster.Consistency = gocql.One
	cluster.Keyspace = "example"
	cluster.ConnectTimeout = 15 * time.Second
	cluster.Timeout = 10 * time.Second

	// Set the New Relic query observer
	cluster.QueryObserver = nrgocql.NewQueryObserver[gocql.ObservedQuery](nil)

	session, err := gocqlx.WrapSession(cluster.CreateSession())
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	// Start a New Relic transaction
	txn := app.StartTransaction("gocqlx-operations")

	// Add transaction to context
	ctx := newrelic.NewContext(context.Background(), txn)
	var tweetMetadata = table.Metadata{
		Name:    "tweet",
		Columns: []string{"timeline", "id", "text"},
		PartKey: []string{"timeline"},
		SortKey: []string{"id"},
	}
	tweetTable := table.New(tweetMetadata)

	var tweets []Tweet
	stmt, names := tweetTable.Select()
	q := session.ContextQuery(ctx, stmt, names).BindMap(qb.M{"timeline": "me"})
	if err := q.SelectRelease(&tweets); err != nil {
		log.Fatal(err)
	}
	txn.End()
	fmt.Println(tweets)
	app.Shutdown(10 * time.Second)

}
