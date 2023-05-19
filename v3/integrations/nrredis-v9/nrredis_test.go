// Copyright 2023 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrredis

import (
	"context"
	"net"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	redis "github.com/redis/go-redis/v9"
)

func emptyDialer(context.Context, string, string) (net.Conn, error) {
	return &net.TCPConn{}, nil
}

func TestPing(t *testing.T) {
	opts := &redis.Options{
		Dialer: emptyDialer,
		Addr:   "myhost:myport",
	}
	client := redis.NewClient(opts)

	app := integrationsupport.NewTestApp(nil, nil)
	txn := app.StartTransaction("txnName")
	ctx := newrelic.NewContext(context.Background(), txn)

	client.AddHook(NewHook(nil))
	client.Ping(ctx)
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/txnName", Forced: nil},
		{Name: "OtherTransactionTotalTime/Go/txnName", Forced: nil},
		{Name: "OtherTransaction/all", Forced: nil},
		{Name: "OtherTransactionTotalTime", Forced: nil},
		{Name: "Datastore/all", Forced: nil},
		{Name: "Datastore/allOther", Forced: nil},
		{Name: "Datastore/Redis/all", Forced: nil},
		{Name: "Datastore/Redis/allOther", Forced: nil},
		{Name: "Datastore/operation/Redis/ping", Forced: nil},
		{Name: "Datastore/operation/Redis/ping", Scope: "OtherTransaction/Go/txnName", Forced: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Forced: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Forced: nil},
	})
}

func TestPingWithOptionsAndAddress(t *testing.T) {
	opts := &redis.Options{
		Dialer: emptyDialer,
		Addr:   "myhost:myport",
	}
	client := redis.NewClient(opts)

	app := integrationsupport.NewTestApp(nil, nil)
	txn := app.StartTransaction("txnName")
	ctx := newrelic.NewContext(context.Background(), txn)

	client.AddHook(NewHook(opts))
	client.Ping(ctx)
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/txnName", Forced: nil},
		{Name: "OtherTransactionTotalTime/Go/txnName", Forced: nil},
		{Name: "OtherTransaction/all", Forced: nil},
		{Name: "OtherTransactionTotalTime", Forced: nil},
		{Name: "Datastore/all", Forced: nil},
		{Name: "Datastore/allOther", Forced: nil},
		{Name: "Datastore/Redis/all", Forced: nil},
		{Name: "Datastore/Redis/allOther", Forced: nil},
		{Name: "Datastore/instance/Redis/myhost/myport", Forced: nil},
		{Name: "Datastore/operation/Redis/ping", Forced: nil},
		{Name: "Datastore/operation/Redis/ping", Scope: "OtherTransaction/Go/txnName", Forced: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Forced: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Forced: nil},
	})
}

func TestPingAndHelloWithPipeline(t *testing.T) {
	opts := &redis.Options{
		Dialer: emptyDialer,
		Addr:   "myhost:myport",
	}
	client := redis.NewClient(opts)

	app := integrationsupport.NewTestApp(nil, nil)
	txn := app.StartTransaction("txnName")
	ctx := newrelic.NewContext(context.Background(), txn)

	client.AddHook(NewHook(opts))
	p := client.Pipeline()
	p.Ping(ctx)
	p.Hello(ctx, 3, "", "", "")
	p.Exec(ctx)
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/txnName", Forced: nil},
		{Name: "OtherTransactionTotalTime/Go/txnName", Forced: nil},
		{Name: "OtherTransaction/all", Forced: nil},
		{Name: "OtherTransactionTotalTime", Forced: nil},
		{Name: "Datastore/all", Forced: nil},
		{Name: "Datastore/allOther", Forced: nil},
		{Name: "Datastore/Redis/all", Forced: nil},
		{Name: "Datastore/Redis/allOther", Forced: nil},
		{Name: "Datastore/instance/Redis/myhost/myport", Forced: nil},
		{Name: "Datastore/operation/Redis/pipeline:ping,hello", Forced: nil},
		{Name: "Datastore/operation/Redis/pipeline:ping,hello", Scope: "OtherTransaction/Go/txnName", Forced: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Forced: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Forced: nil},
	})
}

func TestNewHookAddress(t *testing.T) {
	testcases := []struct {
		network string
		address string
		expHost string
		expPort string
	}{
		// examples from net.Dial https://pkg.go.dev/net#Dial
		{
			network: "tcp",
			address: "golang.org:http",
			expHost: "golang.org",
			expPort: "http",
		},
		{
			network: "", // tcp is assumed if missing
			address: "golang.org:http",
			expHost: "golang.org",
			expPort: "http",
		},
		{
			network: "tcp",
			address: "192.0.2.1:http",
			expHost: "192.0.2.1",
			expPort: "http",
		},
		{
			network: "tcp",
			address: "198.51.100.1:80",
			expHost: "198.51.100.1",
			expPort: "80",
		},
		{
			network: "tcp",
			address: ":80",
			expHost: "localhost",
			expPort: "80",
		},
		{
			network: "tcp",
			address: "0.0.0.0:80",
			expHost: "0.0.0.0",
			expPort: "80",
		},
		{
			network: "tcp",
			address: "[::]:80",
			expHost: "::",
			expPort: "80",
		},
		{
			network: "unix",
			address: "path/to/socket",
			expHost: "localhost",
			expPort: "path/to/socket",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.network+","+tc.address, func(t *testing.T) {
			hk := NewHook(&redis.Options{
				Network: tc.network,
				Addr:    tc.address,
			}).(hook)

			if hk.segment.Host != tc.expHost {
				t.Errorf("incorrect host: expect=%s actual=%s",
					tc.expHost, hk.segment.Host)
			}
			if hk.segment.PortPathOrID != tc.expPort {
				t.Errorf("incorrect port: expect=%s actual=%s",
					tc.expPort, hk.segment.PortPathOrID)
			}
		})
	}
}
