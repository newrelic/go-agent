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
package nrgocqlx

import (
	"context"
	"strconv"
	"strings"

	gocql "github.com/gocql/gocql"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	gocqlx "github.com/scylladb/gocqlx/v3"
)

func init() { internal.TrackUsage("integration", "datastore", "gocql") }

type queryObserver struct {
	original interface {
		ObserveQuery(ctx context.Context, query gocql.ObservedQuery)
	}
}

type NRGocqlxSessionWrapper struct {
	*gocqlx.Session
}

type NRGocqlxQueryxWrapper struct {
	*gocqlx.Queryx
}

func NRGoCQLXWrapSession(cluster *gocql.ClusterConfig) (*NRGocqlxSessionWrapper, error) {
	session, err := gocqlx.WrapSession(cluster.CreateSession())
	if err != nil {
		return nil, err
	}
	return &NRGocqlxSessionWrapper{&session}, nil
}

func execOriginal(ctx context.Context, fn func(ctx context.Context, dest any) error, dest any) error {
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return fn(ctx, dest)
	}

	// start datastore segment
	sgmt := &newrelic.DatastoreSegment{
		StartTime: txn.StartSegmentNow(),
		Product:   newrelic.DatastoreCassandra,
	}
	defer sgmt.End()

	// securtiy agent?
	ctx = context.WithValue(ctx, "nrGocqlxSegment", sgmt)
	return fn(ctx, dest) // enriching of sgmt called withing fn()
}

func (s *NRGocqlxSessionWrapper) ContextQuery(ctx context.Context, stmt string, names []string) *NRGocqlxQueryxWrapper {
	return &NRGocqlxQueryxWrapper{s.Session.ContextQuery(ctx, stmt, names)}
}

func (x *NRGocqlxQueryxWrapper) BindMap(arg map[string]any) *NRGocqlxQueryxWrapper {
	return &NRGocqlxQueryxWrapper{x.Queryx.BindMap(arg)}
}

func (x *NRGocqlxQueryxWrapper) BindStruct(arg any) *NRGocqlxQueryxWrapper {
	return &NRGocqlxQueryxWrapper{x.Queryx.BindStruct(arg)}
}

func (x *NRGocqlxQueryxWrapper) SelectRelease(dest any) error {
	return execOriginal(x.Queryx.Query.Context(), func(ctx context.Context, dest any) error {
		x.Queryx.Query = x.Queryx.Query.WithContext(ctx) // Update ctx stored in Query with segment
		return x.Queryx.SelectRelease(dest)
	}, dest)
}

func (x *NRGocqlxQueryxWrapper) GetRelease(dest any) error {
	return execOriginal(x.Queryx.Query.Context(), func(ctx context.Context, dest any) error {
		x.Queryx.Query = x.Queryx.Query.WithContext(ctx)
		return x.Queryx.GetRelease(dest)
	}, dest)
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
	if observer, ok := any(o.original).(interface {
		ObserveQuery(context.Context, gocql.ObservedQuery)
	}); ok {
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
