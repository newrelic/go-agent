// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	redis "github.com/go-redis/redis/v8"
	nrredis "github.com/newrelic/go-agent/v3/integrations/nrredis-v8"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Redis App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if err != nil {
		panic(err)
	}

	// normally, production code wouldn't require the WaitForConnection call,
	// but for an extremely short-lived script, we want to be sure we are
	// connected before we've already exited.
	app.WaitForConnection(10 * time.Second)

	txn := app.StartTransaction("ping txn")

	opts := &redis.Options{
		Addr: "localhost:6379",
	}
	client := redis.NewClient(opts)

	//
	// Step 1:  Add a nrredis.NewHook() to your redis client.
	//
	client.AddHook(nrredis.NewHook(opts))

	//
	// Step 2: Ensure that all client calls contain a context which includes
	// the transaction.
	//
	ctx := newrelic.NewContext(context.Background(), txn)
	pipe := client.WithContext(ctx).Pipeline()
	incr := pipe.Incr(ctx, "pipeline_counter")
	pipe.Expire(ctx, "pipeline_counter", time.Hour)
	_, err = pipe.Exec(ctx)
	fmt.Println(incr.Val(), err)

	result, err := client.Do(ctx, "INFO", "STATS").Result()
	if err != nil {
		panic(err)
	}
	hits := 0
	misses := 0
	if stats, ok := result.(string); ok {
		sc := bufio.NewScanner(strings.NewReader(stats))
		for sc.Scan() {
			fields := strings.Split(sc.Text(), ":")
			if len(fields) == 2 {
				if v, err := strconv.Atoi(fields[1]); err == nil {
					switch fields[0] {
					case "keyspace_hits":
						hits = v
					case "keyspace_misses":
						misses = v
					}
				}
			}
		}
	}
	if hits+misses > 0 {
		app.RecordCustomMetric("Custom/RedisCache/HitRatio", float64(hits)/(float64(hits+misses)))
	}

	txn.End()
	app.Shutdown(5 * time.Second)
}
