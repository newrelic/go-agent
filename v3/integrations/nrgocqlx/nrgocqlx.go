// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrgocql instruments https://github.com/scylladb/gocqlx/
package nrgocqlx

import (
	"context"
	"reflect"
	"strconv"
	"strings"
	"time"

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
batchObserver contains the implementation for ObserveBatch
and a field for the original ObserveBatch if the user chooses to
call it
*/
type batchObserver struct {
	original interface {
		ObserveBatch(ctx context.Context, batch gocql.ObservedBatch)
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
	segmentRunner    func(fn func() error) error
	CASSegmentRunner func(fn func() (bool, error)) (bool, error)
}

type NRGocqlxBatchWrapper struct {
	*gocqlx.Batch
	segmentRunner    func(fn func() error) error
	CASSegmentRunner func(fn func() (bool, error)) (bool, error)
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
func execOriginal(ctx context.Context, fn func(ctx context.Context) error, contextKey string) error {
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return fn(ctx)
	}

	// start datastore segment
	sgmt := &newrelic.DatastoreSegment{
		StartTime: txn.StartSegmentNow(),
		Product:   newrelic.DatastoreCassandra,
	}
	defer sgmt.End()

	// securtiy agent?
	ctx = context.WithValue(ctx, contextKey, sgmt)
	return fn(ctx) // enriching of sgmt called withing fn()
}

/*
Executes a passed in function while beginning a New Relic Datastore Segment. This function is to be
used with any lightweight queries.  It will return a bool in addition to an errorto indicate if a
query was applied.  If a transactioncannot be pulled from context, no segment will be created but
the passed in function will still execute. The segment gets populated with its StartTime and Product
as the function that is called will enrich the rest ofthe segment.  The segment is stored in context
to be enriched later.
*/
func execOriginalCAS(ctx context.Context, fn func(ctx context.Context) (bool, error), contextKey string) (bool, error) {
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return fn(ctx)
	}

	// start datastore segment
	sgmt := &newrelic.DatastoreSegment{
		StartTime: txn.StartSegmentNow(),
		Product:   newrelic.DatastoreCassandra,
	}
	defer sgmt.End()

	// securtiy agent?
	ctx = context.WithValue(ctx, contextKey, sgmt)
	return fn(ctx) // enriching of sgmt called withing fn()
}

/*
Returns a new NRGocqlxQueryxWrapper.  This sets the Queryx field to the passed in parameter queryx,
the segmentRunner field, and the CASSegmentRunner.
*/
func newNRGocqlxQueryxWrapper(queryx *gocqlx.Queryx) *NRGocqlxQueryxWrapper {
	w := &NRGocqlxQueryxWrapper{Queryx: queryx}
	w.segmentRunner = func(fn func() error) error {
		return execOriginal(w.Context(), func(ctx context.Context) error {
			w.Query = w.Query.WithContext(ctx)
			return fn()
		}, "nrGocqlxSegment")
	}
	w.CASSegmentRunner = func(fn func() (bool, error)) (bool, error) {
		return execOriginalCAS(w.Context(), func(ctx context.Context) (bool, error) {
			w.Query = w.Query.WithContext(ctx)
			return fn()
		}, "nrGocqlxSegment")
	}
	return w
}

func newNRGocqlxBatchWrapper(batch *gocqlx.Batch) *NRGocqlxBatchWrapper {
	w := &NRGocqlxBatchWrapper{Batch: batch}
	w.segmentRunner = func(fn func() error) error {
		return execOriginal(w.Context(), func(ctx context.Context) error {
			w.Batch = w.Batch.WithContext(ctx)
			return fn()
		}, "nrGocqlxBatchSegment")
	}
	w.CASSegmentRunner = func(fn func() (bool, error)) (bool, error) {
		return execOriginalCAS(w.Context(), func(ctx context.Context) (bool, error) {
			w.Batch = w.Batch.WithContext(ctx)
			return fn()
		}, "nrGocqlxBatchSegment")
	}
	return w
}

func (b *NRGocqlxBatchWrapper) withBatch(batch *gocqlx.Batch) *NRGocqlxBatchWrapper {
	b.Batch = batch
	return b
}

func (b *NRGocqlxBatchWrapper) Bind(qry *NRGocqlxQueryxWrapper, args ...any) error {
	return b.Batch.Bind(qry.Queryx, args...)
}

func (b *NRGocqlxBatchWrapper) BindMap(qry *NRGocqlxQueryxWrapper, arg map[string]any) error {
	return b.Batch.BindMap(qry.Queryx, arg)
}

func (b *NRGocqlxBatchWrapper) BindStruct(qry *NRGocqlxQueryxWrapper, arg any) error {
	return b.Batch.BindStruct(qry.Queryx, arg)
}

func (b *NRGocqlxBatchWrapper) BindStructMap(qry *NRGocqlxQueryxWrapper, arg0 any, arg1 map[string]any) error {
	return b.Batch.BindStructMap(qry.Queryx, arg0, arg1)
}

func (b *NRGocqlxBatchWrapper) Query(stmt string, args ...any) *NRGocqlxBatchWrapper {
	return b.withBatch(b.Batch.Query(stmt, args...))
}

func (b *NRGocqlxBatchWrapper) WithContext(ctx context.Context) *NRGocqlxBatchWrapper {
	return b.withBatch(b.Batch.WithContext(ctx))
}

func (b *NRGocqlxBatchWrapper) SetRequestTimeout(timeout time.Duration) *NRGocqlxBatchWrapper {
	return b.withBatch(b.Batch.SetRequestTimeout(timeout))
}

func (b *NRGocqlxBatchWrapper) SetHostID(hostID string) *NRGocqlxBatchWrapper {
	return b.withBatch(b.Batch.SetHostID(hostID))
}

func (b *NRGocqlxBatchWrapper) DefaultTimestamp(enable bool) *NRGocqlxBatchWrapper {
	return b.withBatch(b.Batch.DefaultTimestamp(enable))
}

func (b *NRGocqlxBatchWrapper) WithTimestamp(timestamp int64) *NRGocqlxBatchWrapper {
	return b.withBatch(b.Batch.WithTimestamp(timestamp))
}

func (b *NRGocqlxBatchWrapper) Observer(observer gocql.BatchObserver) *NRGocqlxBatchWrapper {
	return b.withBatch(b.Batch.Observer(observer))
}

func (b *NRGocqlxBatchWrapper) RetryPolicy(policy gocql.RetryPolicy) *NRGocqlxBatchWrapper {
	return b.withBatch(b.Batch.RetryPolicy(policy))
}

func (b *NRGocqlxBatchWrapper) SerialConsistency(cons gocql.Consistency) *NRGocqlxBatchWrapper {
	return b.withBatch(b.Batch.SerialConsistency(cons))
}

func (b *NRGocqlxBatchWrapper) SpeculativeExecutionPolicy(policy gocql.SpeculativeExecutionPolicy) *NRGocqlxBatchWrapper {
	return b.withBatch(b.Batch.SpeculativeExecutionPolicy(policy))
}

func (b *NRGocqlxBatchWrapper) Trace(trace gocql.Tracer) *NRGocqlxBatchWrapper {
	return b.withBatch(b.Batch.Trace(trace))
}

func (b *NRGocqlxBatchWrapper) Exec() error {
	return b.segmentRunner(func() error {
		return b.Batch.Exec()
	})
}

/*
Returns a wrapper NRGocqlxQueryxWrapper which contains embedded fields and overridden implementations
for gocqlx.Queryx.
*/
func (s *NRGocqlxSessionWrapper) ContextQuery(ctx context.Context, stmt string, names []string) *NRGocqlxQueryxWrapper {
	return newNRGocqlxQueryxWrapper(s.Session.ContextQuery(ctx, stmt, names))
}

func (s *NRGocqlxSessionWrapper) ContextBatch(ctx context.Context, bt gocql.BatchType) *NRGocqlxBatchWrapper {
	return newNRGocqlxBatchWrapper(s.Session.ContextBatch(ctx, bt))
}

/*
Sets the NRGocqlxQueryxWrapper.Queryx to the parameter q and returns the wrapper.  This should be
called by any NRGocqlxQueryxWrapper methods (such as Bind) that return a NRGocqlxQueryxWrapper.
*/
func (x *NRGocqlxQueryxWrapper) withQueryx(queryx *gocqlx.Queryx) *NRGocqlxQueryxWrapper {
	x.Queryx = queryx
	return x
}

/*
Sets the arguments of a query.  Use this function, which belongs to the wrapper NRGocqlxQueryxWrapper,
to set the arguments and return a NRGocqlxQueryxWrapper.  If you use gocqlx.Queryx.Bind, you will not be able
to instrument with New Relic.
*/
func (x *NRGocqlxQueryxWrapper) Bind(v ...any) *NRGocqlxQueryxWrapper {
	return x.withQueryx(x.Queryx.Bind(v...))
}

/*
Sets the arguments of a query using a map.  Use this function, which belongs to the wrapper NRGocqlxQueryxWrapper,
to set the arguments and return a NRGocqlxQueryxWrapper.  If you use gocqlx.Queryx.BindMap, you will not be able
to instrument with New Relic.
*/
func (x *NRGocqlxQueryxWrapper) BindMap(arg map[string]any) *NRGocqlxQueryxWrapper {
	return x.withQueryx(x.Queryx.BindMap(arg))
}

/*
Sets the arguments of a query using a struct.  Use this function, which belongs to the wrapper NRGocqlxQueryxWrapper,
to set the arguments and return a NRGocqlxQueryxWrapper.  If you use gocqlx.Queryx.BindStruct, you will not be able
to instrument with New Relic.
*/
func (x *NRGocqlxQueryxWrapper) BindStruct(arg any) *NRGocqlxQueryxWrapper {
	return x.withQueryx(x.Queryx.BindStruct(arg))
}

/*
Sets the arguments of a struct on a map.  Use this function, which belongs to the wrapper NRGocqlxQueryxWrapper,
to set the arguments and return a NRGocqlxQueryxWrapper.  If you use gocqlx.Queryx.BindStructMap, you will not be able
to instrument with New Relic.
*/
func (x *NRGocqlxQueryxWrapper) BindStructMap(arg0 any, arg1 map[string]any) *NRGocqlxQueryxWrapper {
	return x.withQueryx(x.Queryx.BindStructMap(arg0, arg1))
}

/*
Run a Select query and release it immediately after.  Released queries cannot be reused.  This function calls
execOriginal with a function that calls gocqlx.Queryx.SelectRelease with updated context.
*/
func (x *NRGocqlxQueryxWrapper) SelectRelease(dest any) error {
	return x.segmentRunner(func() error { return x.Queryx.SelectRelease(dest) })
}

/*
Run a Select query. This function calls execOriginal with a function that calls gocqlx.Queryx.Select with
updated context.
*/
func (x *NRGocqlxQueryxWrapper) Select(dest any) error {
	return x.segmentRunner(func() error { return x.Queryx.Select(dest) })
}

/*
Run a Get query and release it immediately after.  Released queries cannot be reused.  This function calls
execOriginal with a function that calls gocqlx.Queryx.Get with updated context.
*/
func (x *NRGocqlxQueryxWrapper) GetRelease(dest any) error {
	return x.segmentRunner(func() error { return x.Queryx.GetRelease(dest) })
}

/*
Run a Get query.  This function calls execOriginal with a function that calls gocqlx.Queryx.Get with
updated context.
*/
func (x *NRGocqlxQueryxWrapper) Get(dest any) error {
	return x.segmentRunner(func() error { return x.Queryx.Get(dest) })
}

/*
Run a Get lightweight transaction and release it immediately after.  Released queries cannot be reused.  This function
calls execOriginalCAS with a function that calls gocqlx.Queryx.GetCASRelease with updated context.
*/
func (x *NRGocqlxQueryxWrapper) GetCASRelease(dest any) (bool, error) {
	return x.CASSegmentRunner(func() (bool, error) { return x.Queryx.GetCASRelease(dest) })
}

/*
Run a Get lightweight transaction.  This function calls execOriginalCAS with a function that calls gocqlx.Queryx.GetCAS
with updated context.
*/
func (x *NRGocqlxQueryxWrapper) GetCAS(dest any) (bool, error) {
	return x.CASSegmentRunner(func() (bool, error) { return x.Queryx.GetCAS(dest) })
}

/*
Run an Exec query and release it immediately after.  Released queries cannot be reused.  This function calls
execOriginal with a function that calls gocqlx.Queryx.ExecRelease with updated context.
*/
func (x *NRGocqlxQueryxWrapper) ExecRelease() error {
	return x.segmentRunner(func() error { return x.Queryx.ExecRelease() })
}

/*
Run an Exec query.  This function calls execOriginal with a function that calls gocqlx.Queryx.Exec with
updated context.
*/
func (x *NRGocqlxQueryxWrapper) Exec() error {
	return x.segmentRunner(func() error { return x.Queryx.Exec() })
}

/*
Run an Exec lightweight transaction.  This function calls execOriginalCAS with a function that calls gocqlx.Queryx.ExecCAS
with updated context.
*/
func (x *NRGocqlxQueryxWrapper) ExecCAS() (bool, error) {
	return x.CASSegmentRunner(func() (bool, error) { return x.Queryx.ExecCAS() })
}

/*
Run an Exec lightweight transaction and release it immediately after.  Released queries cannot be reused.  This function
calls execOriginalCAS with a function that calls gocqlx.Queryx.ExecCASRelease with updated context.
*/
func (x *NRGocqlxQueryxWrapper) ExecCASRelease() (bool, error) {
	return x.CASSegmentRunner(func() (bool, error) { return x.Queryx.ExecCASRelease() })
}

/*
Run a query and copies the columns of the first selected row into the values pointed at by dest and discards the rest.
This function calls execOriginal with a function that calls gocqlx.Queryx.Scan with updated context.
*/
func (x *NRGocqlxQueryxWrapper) Scan(v ...any) error {
	return x.segmentRunner(func() error { return x.Queryx.Scan(v...) })
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
NewBatchObserver returns a gocql.BatchObserver that creates newrelic.DatastoreSegment for each database batch query. If provided,
the original gocql.BatchObserver will be called as well.
*/
func NewBatchObserver(original interface {
	ObserveBatch(ctx context.Context, batch gocql.ObservedBatch)
}) *batchObserver {
	if original != nil && reflect.ValueOf(original).IsNil() {
		original = nil
	}
	return &batchObserver{
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

func (o *batchObserver) ObserveBatch(ctx context.Context, batch gocql.ObservedBatch) {
	if o.original != nil {
		o.original.ObserveBatch(ctx, batch)
	}

	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return
	}

	var host, keyspace string
	var statements []string
	var port int

	if batch.Host != nil {
		host = batch.Host.HostID()
		port = batch.Host.Port()
	}
	statements = batch.Statements
	keyspace = batch.Keyspace

	segment, ok := ctx.Value("nrGocqlxBatchSegment").(*newrelic.DatastoreSegment)
	if !ok {
		return
	}
	segment.ParameterizedQuery = strings.Join(statements, "; ") // join statements together
	segment.Host = host
	segment.Collection = "tableNameExample"
	segment.PortPathOrID = strconv.Itoa(port)
	segment.DatabaseName = keyspace
	// enrich segment below
}
