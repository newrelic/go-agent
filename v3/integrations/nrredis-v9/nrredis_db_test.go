//go:build local_redis_test
// +build local_redis_test

// Copyright 2023 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrredis

import (
	"context"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
	redis "github.com/redis/go-redis/v9"
)

// Performs live database testing with an instance of a local Redis database
// on port 6379.
func TestRealDatabaseOperations(t *testing.T) {
	db := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	app := integrationsupport.NewTestApp(nil, nil)
	//txn := app.StartTransaction("build")
	ctx := context.Background()
	//ctx := newrelic.NewContext(context.Background(), txn)
	db.AddHook(NewHookWithOptions(nil,
		ConfigDatastoreKeysEnabled(true),
		ConfigLimitOperations("get", "set"),
	))

	size, err := db.DBSize(ctx).Result()
	if err != nil {
		t.Fatalf("unable to get db size: %v", err)
	}
	if size != 0 {
		t.Fatalf("database is not empty (size=%v), refusing to overwrite existing data; empty database before running tests", size)
	}

	testData := []struct {
		Key   string
		Value any
	}{
		{"Foo", "Bar"},
		{"Spam", "Eggs"},
		{"answer", 42},
		{"maybe", true},
	}

	for i, d := range testData {
		if err := db.Set(ctx, d.Key, d.Value, 0).Err(); err != nil {
			t.Fatalf("database store of item %d failed: %v", i, err)
		}
	}
	//txn.End()
	txn2 := app.StartTransaction("query")
	ctx2 := newrelic.NewContext(context.Background(), txn2)
	for i, d := range testData {
		r := db.Get(ctx2, d.Key)
		v, err := r.Result()
		if err != nil {
			t.Fatalf("retrieval of item %d failed: %v", i, err)
		}
		switch d.Value.(type) {
		case int:
			ri, err := r.Int()
			if err != nil {
				t.Errorf("retrieved value \"%s\" of item %d isn't an integer", v, i)
			}
			if ri != d.Value {
				t.Errorf("retrieved value of item %d was %v, expected %v", i, ri, d.Value)
			}
		case bool:
			rb, err := r.Bool()
			if err != nil {
				t.Errorf("retrieved value \"%s\" of item %d isn't a boolean", v, i)
			}
			if rb != d.Value {
				t.Errorf("retrieved value of item %d was %v, expected %v", i, rb, d.Value)
			}
		default:
			if v != d.Value {
				t.Errorf("retrieved value of item %d was %v, expected %v", i, v, d.Value)
			}
		}
	}
	txn2.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/query", Forced: nil},
		{Name: "OtherTransactionTotalTime/Go/query", Forced: nil},
		{Name: "OtherTransaction/all", Forced: nil},
		{Name: "OtherTransactionTotalTime", Forced: nil},
		{Name: "Datastore/operation/Redis/get", Forced: nil},
		{Name: "Datastore/operation/Redis/get", Scope: "OtherTransaction/Go/query", Forced: nil},
		{Name: "Datastore/all", Forced: nil},
		{Name: "Datastore/allOther", Forced: nil},
		{Name: "Datastore/Redis/all", Forced: nil},
		{Name: "Datastore/Redis/allOther", Forced: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Forced: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Forced: nil},
	})
	//app.ExpectSpanEvents(t, []internal.WantEvent{})
	app.ExpectSlowQueries(t, []internal.WantSlowQuery{})
	// c.DatastoreTracer.SlowQuery.Threshold = 10 * time.Millisecond
	// c.DatastoreTracer.RawQuery.Enabled = true
}
