// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v7"
	nrelasticsearch "github.com/newrelic/go-agent/v3/integrations/nrelasticsearch-v7"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	// Step 1: Use nrelasticsearch.NewRoundTripper to assign the
	// elasticsearch.Config's Transport field.
	cfg := elasticsearch.Config{
		Transport: nrelasticsearch.NewRoundTripper(nil),
	}

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Elastic App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if err != nil {
		panic(err)
	}
	app.WaitForConnection(5 * time.Second)
	txn := app.StartTransaction("elastic")

	// Step 2: Ensure that all calls using the elasticsearch client have a
	// context which includes the newrelic.Transaction.
	ctx := newrelic.NewContext(context.Background(), txn)
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	res, err := es.Info(es.Info.WithContext(ctx))
	if err != nil {
		panic(err)
	}
	if res.IsError() {
		panic(err)
	}
	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		panic(err)
	}
	fmt.Println("ELASTIC SEARCH INFO", elasticsearch.Version, r["version"].(map[string]interface{})["number"])

	txn.End()
	app.Shutdown(5 * time.Second)
}
