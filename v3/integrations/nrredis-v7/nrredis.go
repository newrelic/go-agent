// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrredis instruments github.com/go-redis/redis/v7.
//
// Use this package to instrument your go-redis/redis/v7 calls without having to
// manually create DatastoreSegments.
package nrredis

import (
	"context"
	"net"
	"strings"

	redis "github.com/go-redis/redis/v7"
	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "datastore", "redis") }

type contextKeyType struct{}

type hook struct {
	segment newrelic.DatastoreSegment
}

var (
	segmentContextKey = contextKeyType(struct{}{})
)

// NewHook creates a redis.Hook to instrument Redis calls.  Add it to your
// client, then ensure that all calls contain a context which includes the
// transaction.  The options are optional.  Provide them to get instance metrics
// broken out by host and port.  The hook returned can be used with
// redis.Client, redis.ClusterClient, and redis.Ring.
func NewHook(opts *redis.Options) redis.Hook {
	h := hook{}
	h.segment.Product = newrelic.DatastoreRedis
	if opts != nil {
		// Per https://godoc.org/github.com/go-redis/redis#Options the
		// network should either be tcp or unix, and the default is tcp.
		if opts.Network == "unix" {
			h.segment.Host = "localhost"
			h.segment.PortPathOrID = opts.Addr
		} else if host, port, err := net.SplitHostPort(opts.Addr); err == nil {
			if "" == host {
				host = "localhost"
			}
			h.segment.Host = host
			h.segment.PortPathOrID = port
		}
	}
	return h
}

func (h hook) before(ctx context.Context, operation string) (context.Context, error) {
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return ctx, nil
	}
	s := h.segment
	s.StartTime = txn.StartSegmentNow()
	s.Operation = operation
	ctx = context.WithValue(ctx, segmentContextKey, &s)
	return ctx, nil
}

func (h hook) after(ctx context.Context) {
	if segment, ok := ctx.Value(segmentContextKey).(interface{ End() }); ok {
		segment.End()
	}
}

func (h hook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	return h.before(ctx, cmd.Name())
}

func (h hook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	h.after(ctx)
	return nil
}

func pipelineOperation(cmds []redis.Cmder) string {
	operations := make([]string, 0, len(cmds))
	for _, cmd := range cmds {
		operations = append(operations, cmd.Name())
	}
	return "pipeline:" + strings.Join(operations, ",")
}

func (h hook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	return h.before(ctx, pipelineOperation(cmds))
}

func (h hook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	h.after(ctx)
	return nil
}
