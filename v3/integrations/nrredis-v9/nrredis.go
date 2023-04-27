// Copyright 2023 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrredis instruments github.com/redis/go-redis/v9.
//
// Use this package to instrument your redis/go-redis/v9 calls without having to
// manually create DatastoreSegments.
package nrredis

import (
	"context"
	"net"
	"strings"

	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	redis "github.com/redis/go-redis/v9"
)

func init() { internal.TrackUsage("integration", "datastore", "redis") }

type contextKeyType struct{}

type hook struct {
	segment newrelic.DatastoreSegment
}

var _ redis.Hook = (*hook)(nil)

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
	if opts == nil {
		return h
	}

	// Per https://pkg.go.dev/github.com/redis/go-redis#Options the
	// network should either be tcp or unix, and the default is tcp.
	if opts.Network == "unix" {
		h.segment.Host = "localhost"
		h.segment.PortPathOrID = opts.Addr
		return h
	}
	if host, port, err := net.SplitHostPort(opts.Addr); err == nil {
		if host == "" {
			host = "localhost"
		}
		h.segment.Host = host
		h.segment.PortPathOrID = port
	}
	return h
}

func (h hook) before(ctx context.Context, operation string) context.Context {
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return ctx
	}
	s := h.segment
	s.StartTime = txn.StartSegmentNow()
	s.Operation = operation
	ctx = context.WithValue(ctx, segmentContextKey, &s)
	return ctx
}

func (h hook) after(ctx context.Context) {
	if segment, ok := ctx.Value(segmentContextKey).(interface{ End() }); ok {
		segment.End()
	}
}

func pipelineOperation(cmds []redis.Cmder) string {
	operations := make([]string, 0, len(cmds))
	for _, cmd := range cmds {
		operations = append(operations, cmd.Name())
	}
	return "pipeline:" + strings.Join(operations, ",")
}

func (h hook) DialHook(next redis.DialHook) redis.DialHook {
	return next // just continue the hook
}

func (h hook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		ctx = h.before(ctx, cmd.Name())
		err := next(ctx, cmd)
		h.after(ctx)
		return err
	}
}

func (h hook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		ctx = h.before(ctx, pipelineOperation(cmds))
		err := next(ctx, cmds)
		h.after(ctx)
		return err
	}
}
