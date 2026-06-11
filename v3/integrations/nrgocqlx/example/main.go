// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Run a local Scylla instance in Docker:
//
//	docker run --name my-scylla \
//	  -p 9042:9042 \
//	  -d scylladb/scylla \
//	  --developer-mode 1 \
//	  --memory 1G \
//	  --smp 1 \
//	  --rpc-address 0.0.0.0 \
//	  --broadcast-rpc-address 127.0.0.1
//
// Create the keyspace:
//
//	docker exec -it my-scylla cqlsh -e "CREATE KEYSPACE IF NOT EXISTS example WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};"
//
// Create the table:
//
//	docker exec -it my-scylla cqlsh -e "CREATE TABLE IF NOT EXISTS example.tweet (timeline text, id uuid, text text, PRIMARY KEY (timeline, id));"
//
// Set NEW_RELIC_LICENSE_KEY then run with `go run main.go`.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	gocql "github.com/gocql/gocql"
	"github.com/scylladb/gocqlx/v3/qb"
	"github.com/scylladb/gocqlx/v3/table"

	"github.com/newrelic/go-agent/v3/integrations/nrgocqlx"
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
	defer app.Shutdown(10 * time.Second)

	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Consistency = gocql.One
	cluster.Keyspace = "example"
	cluster.ConnectTimeout = 15 * time.Second
	cluster.Timeout = 10 * time.Second

	// Set the New Relic query and batch observers
	cluster.QueryObserver = nrgocqlx.NewQueryObserver(nil)
	cluster.BatchObserver = nrgocqlx.NewBatchObserver(nil)
	session, err := nrgocqlx.NRGoCQLXWrapSession(cluster)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	// Start a New Relic transaction
	txn := app.StartTransaction("gocqlx-operations")
	defer txn.End()

	// Add transaction to context
	ctx := newrelic.NewContext(context.Background(), txn)
	var tweetMetadata = table.Metadata{
		Name:    "tweet",
		Columns: []string{"timeline", "id", "text"},
		PartKey: []string{"timeline"},
		SortKey: []string{"id"},
	}
	tweetTable := table.New(tweetMetadata)

	uuid := gocql.TimeUUID()

	/*
		Insert a tweet with the above UUID
	*/
	insertStruct := Tweet{
		Timeline: "timeline",
		ID:       uuid,
		Text:     "hello world",
	}
	stmt, names := tweetTable.Insert()
	insertQuery := session.ContextQuery(ctx, stmt, names).BindStruct(insertStruct)
	if err := insertQuery.ExecRelease(); err != nil {
		log.Fatal(err)
	}

	/*
		Insert several more tweets in a single batch
	*/
	batchTweets := []Tweet{
		{Timeline: "timeline", ID: gocql.TimeUUID(), Text: "hello batch 1"},
		{Timeline: "timeline", ID: gocql.TimeUUID(), Text: "hello batch 2"},
		{Timeline: "timeline", ID: gocql.TimeUUID(), Text: "hello batch 3"},
	}
	batchQuery := session.ContextQuery(ctx, stmt, names)
	batch := session.ContextBatch(ctx, gocql.LoggedBatch)
	for _, t := range batchTweets {
		if err := batch.BindStruct(batchQuery, t); err != nil {
			log.Fatal(err)
		}
	}
	if err := batch.Exec(); err != nil {
		log.Fatal(err)
	}

	/*
		Select all tweets with the above timeline
	*/
	var tweets []Tweet
	selectMap := qb.M{"timeline": "timeline"}
	stmt, names = tweetTable.Select()
	selectQuery := session.ContextQuery(ctx, stmt, names).BindMap(selectMap)
	if err := selectQuery.SelectRelease(&tweets); err != nil {
		log.Fatal(err)
	}

	/*
		Get tweet with the above UUID
	*/
	getStruct := Tweet{
		Timeline: "timeline",
		ID:       uuid,
		Text:     "hello world",
	}
	stmt, names = tweetTable.Get()
	getQuery := session.ContextQuery(ctx, stmt, names).BindStruct(getStruct)
	if err := getQuery.GetRelease(&getStruct); err != nil {
		log.Fatal(err)
	}

	/*
		Display results
	*/
	fmt.Printf("\n\n\nNew Inserted row: %v\n\n\n", getStruct)
	fmt.Printf("\n\n\nTweets containing timeline: %v", tweets)

}
