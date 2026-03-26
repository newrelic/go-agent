// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrgocql instruments https://github.com/apache/cassandra-gocql-driver
package nrgocql

import (
	"context"
	"reflect"
	"strconv"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "datastore", "gocql") }

type queryObserver struct {
	original interface {
		ObserveQuery(ctx context.Context, query gocql.ObservedQuery)
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

func (s *NRGoCQLSessionWrapper) Query(stmt string, values ...any) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{s.Session.Query(stmt, values...)}
}

func (q *NRGocqlQueryWrapper) Consistency(c gocql.Consistency) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{Query: q.Query.Consistency(c)}
}

func (q *NRGocqlQueryWrapper) PageSize(n int) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{Query: q.Query.PageSize(n)}
}

func (q *NRGocqlQueryWrapper) Bind(v ...interface{}) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{Query: q.Query.Bind(v...)}
}

func (q *NRGocqlQueryWrapper) RetryPolicy(r gocql.RetryPolicy) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{Query: q.Query.RetryPolicy(r)}
}

func (q *NRGocqlQueryWrapper) Idempotent(value bool) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{Query: q.Query.Idempotent(value)}
}

func (q *NRGocqlQueryWrapper) SerialConsistency(cons gocql.Consistency) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{Query: q.Query.SerialConsistency(cons)}
}

func (q *NRGocqlQueryWrapper) PageState(state []byte) *NRGocqlQueryWrapper {
	return &NRGocqlQueryWrapper{Query: q.Query.PageState(state)}
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
func NewQueryObserver(original interface {
	ObserveQuery(ctx context.Context, query gocql.ObservedQuery)
}) *queryObserver {
	if original != nil && reflect.ValueOf(original).IsNil() {
		original = nil
	}
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
	segment, ok := ctx.Value("nrGocqlSegment").(*newrelic.DatastoreSegment)
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
