// Copyright 2023 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrredis instruments github.com/redis/go-redis/v9.
//
// Use this package to instrument your redis/go-redis/v9 calls without having to
// manually create DatastoreSegments.
package nrredis

import (
	"context"
	"fmt"
	"net"
	"slices"
	"strings"

	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	redis "github.com/redis/go-redis/v9"
)

func init() { internal.TrackUsage("integration", "datastore", "redis") }

type contextKeyType struct{}

type hook struct {
	includeKeys        bool
	operationSet       []string
	agentConfiguration struct {
		retrieved              bool
		queryParametersEnabled bool
		rawQueryEnabled        bool
	}
	segment newrelic.DatastoreSegment
}

var _ redis.Hook = (*hook)(nil)

var (
	segmentContextKey = contextKeyType(struct{}{})
)

// NewHookWithOptions is like NewHook but allows integration-specific
// options to be included as well, such as ConfigDatastoreKeysEanbled.
func NewHookWithOptions(opts *redis.Options, o ...nrredisOpts) redis.Hook {
	h := hook{}
	for _, opt := range o {
		opt(&h)
	}
	return newHook(h, opts)
}

type nrredisOpts func(*hook)

// ConfigDatastoreKeysEnabled controls whether we report the names of
// keys along with the datastore operations in our telemetry. Since the
// keys themselves might contain sensitive information in some databases
// unlike, say, the general case of a parameterized SQL query with placeholders,
// this is disabled by default. However, if you know your keys are safe to
// expose in your telemetry data and wish to see them there, call this method
// on your hook value with a true parameter.
//
// N.B. for Redis database operations, note that for our purposes what we are
// referring to here as "keys" are in fact merely the 2nd parameter in the
// operation parameter list being sent to the Redis server. Typically this
// will be the key or similar ID for the operation at hand, but this will vary
// based on the particular operation being performed. Take care to ensure that
// it is acceptable to record this parameter in your telemetry dataset before
// enabling this option, or restrict the operations for which you wish to expose
// this data by also specifying the ConfigLimitOperations option.
//
// If the agent has also enabled raw database queries via the ConfigDatastoreRawQuery
// option, then the full redis operation will be exposed instead of just the operation
// and following parameter since that option enables the forwarding of the full database
// command string and all data.
func ConfigDatastoreKeysEnabled(enabled bool) func(*hook) {
	return func(h *hook) {
		h.includeKeys = enabled
	}
}

// ConfigLimitOperations restricts the set of operations which will report their
// keys (assuming ConfigDatastoreKeysEnabled is also given with a true value)
// to only those operations whose names match those passed to this option.
func ConfigLimitOperations(name ...string) func(*hook) {
	return func(h *hook) {
		for _, n := range name {
			h.operationSet = append(h.operationSet, n)
		}
	}
}

// NewHook creates a redis.Hook to instrument Redis calls.  Add it to your
// client, then ensure that all calls contain a context which includes the
// transaction.  The options are optional.  Provide them to get instance metrics
// broken out by host and port.  The hook returned can be used with
// redis.Client, redis.ClusterClient, and redis.Ring.
func NewHook(opts *redis.Options) redis.Hook {
	h := hook{}
	return newHook(h, opts)
}

func newHook(h hook, opts *redis.Options) redis.Hook {
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
		if ctx != nil && h.includeKeys {
			// Only go to the expense of collecting this data if we are going to
			// possibly be reporting it out later, so check the agent's configuration
			// (but only once)
			if !h.agentConfiguration.retrieved {
				if txn := newrelic.FromContext(ctx); txn != nil {
					if cfg, isValid := txn.Application().Config(); isValid {
						h.agentConfiguration.retrieved = true
						h.agentConfiguration.queryParametersEnabled = cfg.DatastoreTracer.QueryParameters.Enabled
						h.agentConfiguration.rawQueryEnabled = cfg.DatastoreTracer.RawQuery.Enabled
					}
				}
			}
			operationName := cmd.Name()
			if len(h.operationSet) == 0 || slices.ContainsFunc(h.operationSet, func(op string) bool { return strings.EqualFold(operationName, op) }) {
				args := cmd.Args()
				if args != nil && len(args) > 0 {
					if h.agentConfiguration.rawQueryEnabled {
						h.segment.RawQuery = fmt.Sprintf("%v", args)
					}
					if len(args) > 1 {
						if h.agentConfiguration.queryParametersEnabled {
							if h.segment.QueryParameters == nil {
								h.segment.QueryParameters = make(map[string]any)
							}
							h.segment.QueryParameters["key"] = args[1]
							h.segment.ParameterizedQuery = fmt.Sprintf("%v %v", args[0], args[1])
						}
					}
				}
			}
		}
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
