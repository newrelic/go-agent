// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrelasticsearch_test

import (
	"context"

	elasticsearch "github.com/elastic/go-elasticsearch/v7"
	nrelasticsearch "github.com/newrelic/go-agent/v3/integrations/nrelasticsearch-v7"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func getTransaction() *newrelic.Transaction { return nil }

func Example() {
	// Step 1: Use nrelasticsearch.NewRoundTripper to assign the
	// elasticsearch.Config's Transport field.
	cfg := elasticsearch.Config{
		Transport: nrelasticsearch.NewRoundTripper(nil),
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	// Step 2: Ensure that all calls using the elasticsearch client have
	// a context which includes the newrelic.Transaction.
	txn := getTransaction()
	ctx := newrelic.NewContext(context.Background(), txn)
	client.Info(client.Info.WithContext(ctx))
}
