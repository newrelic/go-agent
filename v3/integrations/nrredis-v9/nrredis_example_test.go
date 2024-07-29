// Copyright 2023 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrredis_test

import (
	"context"
	"fmt"

	nrredis "github.com/newrelic/go-agent/v3/integrations/nrredis-v9"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	redis "github.com/redis/go-redis/v9"
)

func getTransaction() *newrelic.Transaction { return nil }

func Example_client() {
	opts := &redis.Options{Addr: "localhost:6379"}
	client := redis.NewClient(opts)

	//
	// Step 1:  Add a nrredis.NewHook() to your redis client.
	//
	client.AddHook(nrredis.NewHook(opts))

	//
	// Step 2: Ensure that all client calls contain a context with includes
	// the transaction.
	//
	txn := getTransaction()
	ctx := newrelic.NewContext(context.Background(), txn)
	pong, err := client.Ping(ctx).Result()
	fmt.Println(pong, err)
}

func Example_clusterClient() {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{":7000", ":7001", ":7002", ":7003", ":7004", ":7005"},
	})

	//
	// Step 1:  Add a nrredis.NewHook() to your redis cluster client.
	//
	client.AddHook(nrredis.NewHook(nil))

	//
	// Step 2: Ensure that all client calls contain a context with includes
	// the transaction.
	//
	txn := getTransaction()
	ctx := newrelic.NewContext(context.Background(), txn)
	pong, err := client.Ping(ctx).Result()
	fmt.Println(pong, err)
}
