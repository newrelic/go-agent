package nrredis

import (
	"context"
	"net"
	"testing"

	redis "github.com/go-redis/redis/v7"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
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
	client.WithContext(ctx).Ping()
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
	client.WithContext(ctx).Ping()
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
	})
}

func TestPipelineOperation(t *testing.T) {
	// As of Jan 16, 2020, it is impossible to test pipeline operations using
	// a &net.TCPConn{}, so we will have to make do with this.
	if op := pipelineOperation(nil); op != "pipeline:" {
		t.Error(op)
	}
	cmds := []redis.Cmder{redis.NewCmd("GET"), redis.NewCmd("SET")}
	if op := pipelineOperation(cmds); op != "pipeline:get,set" {
		t.Error(op)
	}
}

func TestNewHookAddress(t *testing.T) {
	testcases := []struct {
		network string
		address string
		expHost string
		expPort string
	}{
		// examples from net.Dial https://godoc.org/net#Dial
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
