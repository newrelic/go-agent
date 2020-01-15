package nrredis_test

import (
	"context"
	"fmt"

	redis "github.com/go-redis/redis/v7"
	nrredis "github.com/newrelic/go-agent/v3/integrations/nrredis-v7"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func getTransaction() *newrelic.Transaction { return nil }

func Example() {
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
