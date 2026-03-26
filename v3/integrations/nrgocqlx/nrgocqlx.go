// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrgocql instruments https://github.com/scylladb/gocqlx/
package nrgocqlx

import (
	"context"
	"strconv"

	gocql "github.com/gocql/gocql"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "datastore", "gocql") }

type queryObserver struct {
	original interface {
		ObserveQuery(ctx context.Context, query gocql.ObservedQuery)
	}
}

// NewQueryObserver returns a gocql.QueryObserver that creates
// newrelic.DatastoreSegment for each database query. If provided, the
// original gocql.QueryObserver will be called as well.
func NewQueryObserver(original interface {
	ObserveQuery(ctx context.Context, query gocql.ObservedQuery)
}) *queryObserver {
	return &queryObserver{
		original: original,
	}
}

// ObserveQuery implements the gocql.QueryObserver interface
func (o *queryObserver) ObserveQuery(ctx context.Context, query gocql.ObservedQuery) {

	if o.original != nil {
		o.original.ObserveQuery(ctx, query)
	}

	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return
	}

	var host, statement, keyspace string
	var port int

	if query.Host != nil {
		host = query.Host.HostID()
		port = query.Host.Port()
	}
	statement = query.Statement
	keyspace = query.Keyspace

	// enrich segment
	segment, ok := ctx.Value("nrGocqlxSegment").(*newrelic.DatastoreSegment)
	if !ok {
		return
	}
	segment.ParameterizedQuery = statement
	segment.Host = host
	segment.Collection = "tableNameExample"
	segment.PortPathOrID = strconv.Itoa(port)
	segment.DatabaseName = keyspace

	// security agent?
}
