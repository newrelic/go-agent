package nrredis_test

import (
	"context"
	"fmt"

	redis "github.com/go-redis/redis/v7"
	nrredis "github.com/newrelic/go-agent/v3/integrations/nrredis-v7"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
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
	pong, err := client.WithContext(ctx).Ping().Result()
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
	pong, err := client.WithContext(ctx).Ping().Result()
	fmt.Println(pong, err)
}
