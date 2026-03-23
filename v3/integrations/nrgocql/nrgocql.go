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

type NRGoCQLSessionWrapper struct {
	*gocql.Session
}
type NRGocqlQueryWrapper struct {
	*gocql.Query
}

func NRGoCQLNewSession(cfg *gocql.ClusterConfig) (*NRGoCQLSessionWrapper, error) {
	session, err := cfg.CreateSession()
	if err != nil {
		return nil, err
	}
	return &NRGoCQLSessionWrapper{session}, nil
}

func (s *NRGoCQLSessionWrapper) Query(stmt string, values ...any) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{s.Session.Query(stmt, values...)}
}

func execOriginal(ctx context.Context, fn func(ctx context.Context, dest ...any) error, dest ...any) error {
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return fn(ctx, dest...)
	}

	// start datastore segment
	sgmt := &newrelic.DatastoreSegment{
		StartTime: txn.StartSegmentNow(),
		Product:   newrelic.DatastoreCassandra,
	}
	defer sgmt.End()

	// securtiy agent?
	ctx = context.WithValue(ctx, "nrGocqlSegment", sgmt)
	return fn(ctx, dest...) // enriching of sgmt called withing fn()
}

func (q *NRGocqlQueryWrapper) Consistency(c gocql.Consistency) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{q.Query.Consistency(c)}
}

func (q *NRGocqlQueryWrapper) PageSize(n int) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{q.Query.PageSize(n)}
}

func (q *NRGocqlQueryWrapper) Bind(v ...interface{}) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{q.Query.Bind(v...)}
}

func (q *NRGocqlQueryWrapper) RetryPolicy(r gocql.RetryPolicy) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{q.Query.RetryPolicy(r)}
}

func (q *NRGocqlQueryWrapper) Idempotent(value bool) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{q.Query.Idempotent(value)}
}

func (q *NRGocqlQueryWrapper) SerialConsistency(cons gocql.Consistency) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{q.Query.SerialConsistency(cons)}
}

func (q *NRGocqlQueryWrapper) PageState(state []byte) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{q.Query.PageState(state)}
}

func (q *NRGocqlQueryWrapper) ExecContext(ctx context.Context) error {
	return execOriginal(ctx, func(ctx context.Context, dest ...any) error {
		err := q.Query.ExecContext(ctx)
		if err != nil {
			return err
		}
		return nil
	})
}

func (q *NRGocqlQueryWrapper) ScanContext(ctx context.Context, dest ...any) error {
	return execOriginal(ctx, func(ctx context.Context, dest ...any) error {
		err := q.Query.ScanContext(ctx, dest...)
		if err != nil {
			return err
		}
		return nil
	}, dest...)
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
	if observer, ok := any(o.original).(interface{ ObserveQuery(context.Context, T) }); ok {
		if observer != nil {
			observer.ObserveQuery(ctx, query)
		}
	}

	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return
	}

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

	// enrich segment
	segment, ok := ctx.Value("nrGocqlSegment").(*newrelic.DatastoreSegment)
	if !ok {
		return
	}
	segment.Operation = extractOperation(statement)
	segment.ParameterizedQuery = statement
	segment.Host = host
	segment.Collection = "tableNameExample"
	segment.PortPathOrID = strconv.Itoa(port)
	segment.DatabaseName = keyspace

	// security agent?
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
