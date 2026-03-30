// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrgocql instruments https://github.com/scylladb/gocqlx/
package nrgocqlx

import (
	"context"
	"reflect"
	"strconv"

	gocql "github.com/gocql/gocql"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/scylladb/gocqlx/v3"
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

func (x *NRGocqlxQueryxWrapper) Bind(v ...any) *NRGocqlxQueryxWrapper {
	return &NRGocqlxQueryxWrapper{x.Queryx.Bind(v...)}
}

func (x *NRGocqlxQueryxWrapper) BindMap(arg map[string]any) *NRGocqlxQueryxWrapper {
	return &NRGocqlxQueryxWrapper{x.Queryx.BindMap(arg)}
}

func (x *NRGocqlxQueryxWrapper) BindStruct(arg any) *NRGocqlxQueryxWrapper {
	return &NRGocqlxQueryxWrapper{x.Queryx.BindStruct(arg)}
}

func (x *NRGocqlxQueryxWrapper) BindStructMap(arg0 any, arg1 map[string]any) *NRGocqlxQueryxWrapper {
	return &NRGocqlxQueryxWrapper{x.Queryx.BindStructMap(arg0, arg1)}
}

func (x *NRGocqlxQueryxWrapper) SelectRelease(dest any) error {
	return execOriginal(x.Queryx.Query.Context(), func(ctx context.Context, dest any) error {
		x.Queryx.Query = x.Queryx.Query.WithContext(ctx)
		return x.Queryx.SelectRelease(dest)
	}, dest)
}

func (x *NRGocqlxQueryxWrapper) Select(dest any) error {
	return execOriginal(x.Queryx.Query.Context(), func(ctx context.Context, dest any) error {
		x.Queryx.Query = x.Queryx.Query.WithContext(ctx)
		return x.Queryx.Select(dest)
	}, dest)
}

func (x *NRGocqlxQueryxWrapper) GetRelease(dest any) error {
	return execOriginal(x.Queryx.Query.Context(), func(ctx context.Context, dest any) error {
		x.Queryx.Query = x.Queryx.Query.WithContext(ctx)
		return x.Queryx.GetRelease(dest)
	}, dest)
}

func (x *NRGocqlxQueryxWrapper) Get(dest any) error {
	return execOriginal(x.Queryx.Query.Context(), func(ctx context.Context, dest any) error {
		x.Queryx.Query = x.Queryx.Query.WithContext(ctx)
		return x.Queryx.Get(dest)
	}, dest)
}

func (x *NRGocqlxQueryxWrapper) ExecRelease() error {
	return execOriginal(x.Queryx.Query.Context(), func(ctx context.Context, dest any) error {
		x.Queryx.Query = x.Queryx.Query.WithContext(ctx)
		return x.Queryx.ExecRelease()
	}, nil)
}

func (x *NRGocqlxQueryxWrapper) Exec() error {
	return execOriginal(x.Queryx.Query.Context(), func(ctx context.Context, dest any) error {
		x.Queryx.Query = x.Queryx.Query.WithContext(ctx)
		return x.Queryx.Exec()
	}, nil)
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
