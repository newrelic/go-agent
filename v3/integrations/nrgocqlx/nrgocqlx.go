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

/*
queryObserver contains the implementation for ObserveQuery
and a field for the original ObserveQuery if the user chooses to
call it
*/
type queryObserver struct {
	original interface {
		ObserveQuery(ctx context.Context, query gocql.ObservedQuery)
	}
}

/*
Wrapper for gocqlx.Session that implements gocqlx.Session.ContextQuery. All other gocqlx.Session
functions are accessible through this struct's embedded field *gocqlx.Session. This wrapper
does not implement any gocql.Session functions, however those functions are also accessible through
embedded fields. The wrapper's implementation of ContextQuery must be used in order to instrument
with New Relic properly.
*/
type NRGocqlxSessionWrapper struct {
	*gocqlx.Session
}

/*
Wrapper for gocqlx.Queryx that implements gocqlx.Queryx functions: Bind, BindMap, BindStruct,
BindStructMap, SelectRelease, Select, GetRelease, Get, GetCAS, GetCASRelease, ExecRelease, Exec,
ExecCAS, ExecCASRelease and Scan. All other gocqlx.Queryx functions are accessible through this
struct's embedded field *gocqlx.Queryx.  This wrapper does not implement any gocql.Query functions,
however those functions are also accessible through embedded fields.  NOTE: In order to properly
instrument with New Relic, you must use only the implemented functions for NRGocqlxQueryxWrapper,
especially if you are chaining.
*/
type NRGocqlxQueryxWrapper struct {
	*gocqlx.Queryx
}

/*
Call that wraps a gocqlx.Session.  This function takes a gocql.ClusterConfig and returns a
NRGocqlxSessionWrapper.
*/
func NRGoCQLXWrapSession(cluster *gocql.ClusterConfig) (*NRGocqlxSessionWrapper, error) {
	session, err := gocqlx.WrapSession(cluster.CreateSession())
	if err != nil {
		return nil, err
	}
	return &NRGocqlxSessionWrapper{&session}, nil
}

/*
Executes a passed in function while beginning a New Relic Datastore Segment. This function does
not accept a spread operator and only will function for one destination. If a transaction
cannot be pulled from context, no segment will be created but the passed in function will still execute. The
segment gets populated with its StartTime and Product as the function that is called will enrich the rest of
the segment.  The segment is stored in context to be enriched later.
*/
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

/*
Executes a passed in function while beginning a New Relic Datastore Segment. This function does
accepts a spread operator and only will function for multiple destinations. If a transaction
cannot be pulled from context, no segment will be created but the passed in function will still execute. The
segment gets populated with its StartTime and Product as the function that is called will enrich the rest of
the segment.  The segment is stored in context to be enriched later.
*/
func execOriginalSpread(ctx context.Context, fn func(ctx context.Context, dest ...any) error, dest ...any) error {
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
	ctx = context.WithValue(ctx, "nrGocqlxSegment", sgmt)
	return fn(ctx, dest...) // enriching of sgmt called withing fn()
}

/*
Executes a passed in function while beginning a New Relic Datastore Segment. This function is to be
used with any lightweight queries.  It will return a bool in addition to an errorto indicate if a
query was applied.  If a transactioncannot be pulled from context, no segment will be created but
the passed in function will still execute. The segment gets populated with its StartTime and Product
as the function that is called will enrich the rest ofthe segment.  The segment is stored in context
to be enriched later.
*/
func execOriginalCAS(ctx context.Context, fn func(ctx context.Context, dest any) (bool, error), dest any) (bool, error) {
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

/*
Returns a wrapper NRGocqlxQueryxWrapper which contains embedded fields and overridden implementations
for gocqlx.Queryx.
*/
func (s *NRGocqlxSessionWrapper) ContextQuery(ctx context.Context, stmt string, names []string) *NRGocqlxQueryxWrapper {
	return &NRGocqlxQueryxWrapper{s.Session.ContextQuery(ctx, stmt, names)}
}

/*
Sets the arguments of a query.  Use this function, which belongs to the wrapper NRGocqlxQueryxWrapper,
to set the arguments and return a NRGocqlxQueryxWrapper.  If you use gocqlx.Queryx.Bind, you will not be able
to instrument with New Relic.
*/
func (x *NRGocqlxQueryxWrapper) Bind(v ...any) *NRGocqlxQueryxWrapper {
	return &NRGocqlxQueryxWrapper{x.Queryx.Bind(v...)}
}

/*
Sets the arguments of a query using a map.  Use this function, which belongs to the wrapper NRGocqlxQueryxWrapper,
to set the arguments and return a NRGocqlxQueryxWrapper.  If you use gocqlx.Queryx.BindMap, you will not be able
to instrument with New Relic.
*/
func (x *NRGocqlxQueryxWrapper) BindMap(arg map[string]any) *NRGocqlxQueryxWrapper {
	return &NRGocqlxQueryxWrapper{x.Queryx.BindMap(arg)}
}

/*
Sets the arguments of a query using a struct.  Use this function, which belongs to the wrapper NRGocqlxQueryxWrapper,
to set the arguments and return a NRGocqlxQueryxWrapper.  If you use gocqlx.Queryx.BindStruct, you will not be able
to instrument with New Relic.
*/
func (x *NRGocqlxQueryxWrapper) BindStruct(arg any) *NRGocqlxQueryxWrapper {
	return &NRGocqlxQueryxWrapper{x.Queryx.BindStruct(arg)}
}

/*
Sets the arguments of a struct on a map.  Use this function, which belongs to the wrapper NRGocqlxQueryxWrapper,
to set the arguments and return a NRGocqlxQueryxWrapper.  If you use gocqlx.Queryx.BindStructMap, you will not be able
to instrument with New Relic.
*/
func (x *NRGocqlxQueryxWrapper) BindStructMap(arg0 any, arg1 map[string]any) *NRGocqlxQueryxWrapper {
	return &NRGocqlxQueryxWrapper{x.Queryx.BindStructMap(arg0, arg1)}
}

/*
Run a Select query and release it immediately after.  Released queries cannot be reused.  This function calls
execOriginal with a function that calls gocqlx.Queryx.SelectRelease with updated context.
*/
func (x *NRGocqlxQueryxWrapper) SelectRelease(dest any) error {
	return execOriginal(x.Context(), func(ctx context.Context, dest any) error {
		x.Query = x.Query.WithContext(ctx)
		return x.Queryx.SelectRelease(dest)
	}, dest)
}

/*
Run a Select query. This function calls execOriginal with a function that calls gocqlx.Queryx.Select with
updated context.
*/
func (x *NRGocqlxQueryxWrapper) Select(dest any) error {
	return execOriginal(x.Context(), func(ctx context.Context, dest any) error {
		x.Query = x.Query.WithContext(ctx)
		return x.Queryx.Select(dest)
	}, dest)
}

/*
Run a Get query and release it immediately after.  Released queries cannot be reused.  This function calls
execOriginal with a function that calls gocqlx.Queryx.Get with updated context.
*/
func (x *NRGocqlxQueryxWrapper) GetRelease(dest any) error {
	return execOriginal(x.Context(), func(ctx context.Context, dest any) error {
		x.Query = x.Query.WithContext(ctx)
		return x.Queryx.GetRelease(dest)
	}, dest)
}

/*
Run a Get query.  This function calls execOriginal with a function that calls gocqlx.Queryx.Get with
updated context.
*/
func (x *NRGocqlxQueryxWrapper) Get(dest any) error {
	return execOriginal(x.Context(), func(ctx context.Context, dest any) error {
		x.Query = x.Query.WithContext(ctx)
		return x.Queryx.Get(dest)
	}, dest)
}

/*
Run a Get lightweight transaction and release it immediately after.  Released queries cannot be reused.  This function
calls execOriginalCAS with a function that calls gocqlx.Queryx.GetCASRelease with updated context.
*/
func (x *NRGocqlxQueryxWrapper) GetCASRelease(dest any) (bool, error) {
	return execOriginalCAS(x.Context(), func(ctx context.Context, dest any) (bool, error) {
		x.Query = x.Query.WithContext(ctx)
		return x.Queryx.GetCASRelease(dest)
	}, dest)
}

/*
Run a Get lightweight transaction.  This function calls execOriginalCAS with a function that calls gocqlx.Queryx.GetCAS
with updated context.
*/
func (x *NRGocqlxQueryxWrapper) GetCAS(dest any) (bool, error) {
	return execOriginalCAS(x.Context(), func(ctx context.Context, dest any) (bool, error) {
		x.Query = x.Query.WithContext(ctx)
		return x.Queryx.GetCAS(dest)
	}, dest)
}

/*
Run an Exec query and release it immediately after.  Released queries cannot be reused.  This function calls
execOriginal with a function that calls gocqlx.Queryx.ExecRelease with updated context.
*/
func (x *NRGocqlxQueryxWrapper) ExecRelease() error {
	return execOriginal(x.Context(), func(ctx context.Context, dest any) error {
		x.Query = x.Query.WithContext(ctx)
		return x.Queryx.ExecRelease()
	}, nil)
}

/*
Run an Exec query.  This function calls execOriginal with a function that calls gocqlx.Queryx.Exec with
updated context.
*/
func (x *NRGocqlxQueryxWrapper) Exec() error {
	return execOriginal(x.Context(), func(ctx context.Context, dest any) error {
		x.Query = x.Query.WithContext(ctx)
		return x.Queryx.Exec()
	}, nil)
}

/*
Run an Exec lightweight transaction.  This function calls execOriginalCAS with a function that calls gocqlx.Queryx.ExecCAS
with updated context.
*/
func (x *NRGocqlxQueryxWrapper) ExecCAS() (bool, error) {
	return execOriginalCAS(x.Context(), func(ctx context.Context, dest any) (bool, error) {
		x.Query = x.Query.WithContext(ctx)
		return x.Queryx.ExecCAS()
	}, nil)
}

/*
Run an Exec lightweight transaction and release it immediately after.  Released queries cannot be reused.  This function
calls execOriginalCAS with a function that calls gocqlx.Queryx.ExecCASRelease with updated context.
*/
func (x *NRGocqlxQueryxWrapper) ExecCASRelease() (bool, error) {
	return execOriginalCAS(x.Context(), func(ctx context.Context, dest any) (bool, error) {
		x.Query = x.Query.WithContext(ctx)
		return x.Queryx.ExecCASRelease()
	}, nil)
}

/*
Run a query and copies the columns of the first selected row into the values pointed at by dest and discards the rest.
This function calls execOriginal with a function that calls gocqlx.Queryx.Scan with updated context.
*/
func (x *NRGocqlxQueryxWrapper) Scan(v ...any) error {
	return execOriginalSpread(x.Context(), func(ctx context.Context, dest ...any) error {
		x.Query = x.Query.WithContext(ctx)
		return x.Queryx.Scan(dest...)
	}, v...)
}

/*
NewQueryObserver returns a gocql.QueryObserver that creates newrelic.DatastoreSegment for each database query. If provided,
the original gocql.QueryObserver will be called as well.
*/
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

/*
ObserveQuery is the implementation for the gocql.QueryObserver.  This will run after the
query is executed.  It will execute the original implementation of ObserveQuery if it is
passed in.  If there is no new relic transaction in context, it will return early.  Otherwise,
it will take the segment from context, and enrich it with query data.
*/
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
