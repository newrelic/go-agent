// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrgocql instruments https://github.com/apache/cassandra-gocql-driver
//
// Use this package to instrument your Cassandra/ScyllaDB calls without having to manually
// create DatastoreSegments. To do so, set the query observer in the cluster configuration:
//
//	cluster := gocql.NewCluster("127.0.0.1")
//	cluster.Keyspace = "example"
//	cluster.QueryObserver = nrgocql.NewQueryObserver(nil)
//	session, err := cluster.CreateSession()
//
// Then add the current transaction to the context used in any query:
//
//	ctx := newrelic.NewContext(context.Background(), txn)
//	err := session.Query(`SELECT id, text FROM tweet WHERE timeline = ?`, "me").ExecContext(ctx)
package nrgocql

import (
	"context"
	"strconv"
	"strings"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	oldgocql "github.com/gocql/gocql"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "datastore", "gocql") }

type queryObserver[T any] struct {
	original interface {
		ObserveQuery(ctx context.Context, query T)
	}
}

// NewQueryObserver returns a gocql.QueryObserver that creates
// newrelic.DatastoreSegment for each database query. If provided, the
// original gocql.QueryObserver will be called as well.
func NewQueryObserver[T any](original interface {
	ObserveQuery(ctx context.Context, query T)
}) *queryObserver[T] {
	return &queryObserver[T]{
		original: original,
	}
}

// ObserveQuery implements the gocql.QueryObserver interface
func (o *queryObserver[T]) ObserveQuery(ctx context.Context, query T) {
	// Call original observer if present
	if observer, ok := any(o.original).(interface{ ObserveQuery(context.Context, T) }); ok {
		if observer != nil {
			observer.ObserveQuery(ctx, query)
		}
	}

	// Get transaction from context
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return
	}

	// Create and immediately end the segment since ObserveQuery is called after completion
	var host, statement, keyspace string
	var port int
	switch q := any(query).(type) {
	case gocql.ObservedQuery:
		if q.Host != nil {
			host = q.Host.HostID()
			port = q.Host.Port()
		}
		statement = q.Statement
		keyspace = q.Keyspace
	case oldgocql.ObservedQuery:
		if q.Host != nil {
			host = q.Host.HostID()
			port = q.Host.Port()
		}
		statement = q.Statement
		keyspace = q.Keyspace
	default:

	}

	sgmt := &newrelic.DatastoreSegment{
		StartTime:          txn.StartSegmentNow(),
		Product:            newrelic.DatastoreCassandra,
		Operation:          extractOperation(statement),
		ParameterizedQuery: statement,
		Host:               host,
		PortPathOrID:       strconv.Itoa(port),
		DatabaseName:       keyspace,
	}

	sgmt.End()
}

// extractOperation extracts the operation type from a CQL statement
// e.g., "SELECT", "INSERT", "UPDATE", "DELETE"
func extractOperation(statement string) string {
	statement = strings.TrimSpace(statement)
	if len(statement) == 0 {
		return "unknown"
	}

	// Find the first word (operation)
	idx := strings.IndexAny(statement, " \t\n")
	if idx == -1 {
		return strings.ToUpper(statement)
	}
	return strings.ToUpper(statement[:idx])
}
