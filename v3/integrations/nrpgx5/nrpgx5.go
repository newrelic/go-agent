// Package nrpgx5 instruments https://github.com/jackc/pgx/v5.
//
// Use this package to instrument your PostgreSQL calls using the pgx
// library.
//
// This integration is specifically aimed at instrumenting applications which
// use the pgx/v5 library to directly communicate with the Postgres database server
// (i.e., not via the standard database/sql library).
//
// To instrument your database operations, you will need to call nrpgx5.NewTracer() to obtain
// a pgx.Tracer value. You can do this either with a normal pgx.ParseConfig() call or the
// pgxpool.ParseConfig() call if you wish to use pgx connection pools.
//
// For example:
//
//    import (
//    	"github.com/jackc/pgx/v5"
// 	   "github.com/newrelic/go-agent/v3/integrations/nrpgx5"
//    	"github.com/newrelic/go-agent/v3/newrelic"
//    )
//
//    func main() {
// 	   cfg, err := pgx.ParseConfig("postgres://postgres:postgres@localhost:5432") // OR pgxpools.ParseConfig(...)
// 	   if err != nil {
//        panic(err)
//     }
//
// 	   cfg.Tracer = nrpgx5.NewTracer()
//     conn, err := pgx.ConnectConfig(context.Background(), cfg)
//     if err != nil {
//        panic(err)
//     }
//
// See the programs in the example directory for working examples of each use case.

package nrpgx5

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/sqlparse"
)

func init() {
	internal.TrackUsage("integration", "driver", "nrpgx5")
}

type (
	Tracer struct {
		BaseSegment         newrelic.DatastoreSegment
		ParseQuery          func(segment *newrelic.DatastoreSegment, query string)
		SendQueryParameters bool
	}

	nrPgxSegmentType string
)

const (
	querySegmentKey   nrPgxSegmentType = "nrPgx5Segment"
	prepareSegmentKey nrPgxSegmentType = "prepareNrPgx5Segment"
	batchSegmentKey   nrPgxSegmentType = "batchNrPgx5Segment"
	querySecurityKey  nrPgxSegmentType = "nrPgx5SecurityToken"
)

type TracerOption func(*Tracer)

// NewTracer creates a new value which implements pgx.BatchTracer, pgx.ConnectTracer, pgx.PrepareTracer, and pgx.QueryTracer.
// This value will be used to facilitate instrumentation of the database operations performed.
// When establishing a connection to the database, the recommended usage is to do something like the following:
//    cfg, err := pgx.ParseConfig("...")
//    if err != nil { ... }
//    cfg.Tracer = nrpgx5.NewTracer()
//    conn, err := pgx.ConnectConfig(context.Background(), cfg)
//
// If you do not wish to have SQL query parameters included in the telemetry data, add the WithQueryParameters
// option, like so:
//    cfg.Tracer = nrpgx5.NewTracer(nrpgx5.WithQueryParameters(false))
//
// (The default is to collect query parameters, but you can explicitly select this by passing true to WithQueryParameters.)
//
// Note that query parameters may nevertheless be suppressed from the telemetry data due to agent configuration,
// agent feature set, or policy independint of whether it's enabled here.
func NewTracer(o ...TracerOption) *Tracer {
	t := &Tracer{
		ParseQuery:          sqlparse.ParseQuery,
		SendQueryParameters: true,
	}

	for _, opt := range o {
		opt(t)
	}

	return t
}

// WithQueryParameters is an option which may be passed to a call to NewTracer. It controls
// whether or not to include the SQL query parameters in the telemetry data collected as part of
// instrumenting database operations.
//
// By default this is enabled. To disable it, call NewTracer as NewTracer(WithQueryParameters(false)).
//
// Note that query parameters may nevertheless be suppressed from the telemetry data due to agent configuration,
// agent feature set, or policy independint of whether it's enabled here.
func WithQueryParameters(enabled bool) TracerOption {
	return func(t *Tracer) {
		t.SendQueryParameters = enabled
	}
}

// TraceConnectStart is called at the beginning of Connect and ConnectConfig calls, as
// what is essentially a callback from the pgx/v5 library to us so we can trace the operation.
// The returned context is used for
// the rest of the call and will be passed to TraceConnectEnd.
func (t *Tracer) TraceConnectStart(ctx context.Context, data pgx.TraceConnectStartData) context.Context {
	t.BaseSegment = newrelic.DatastoreSegment{
		Product:      newrelic.DatastorePostgres,
		Host:         data.ConnConfig.Host,
		PortPathOrID: strconv.FormatUint(uint64(data.ConnConfig.Port), 10),
		DatabaseName: data.ConnConfig.Database,
	}

	return ctx
}

// TraceConnectEnd is called by pgx/v5 at the end of the Connect and ConnectConfig calls.
func (Tracer) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {}

// TraceQueryStart is called by pgx/v5 at the beginning of Query, QueryRow, and Exec calls.
// The returned context is used for the
// rest of the call and will be passed to TraceQueryEnd.
// This starts a new datastore segment in the transaction stored in the passed context.
func (t *Tracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	segment := t.BaseSegment
	segment.StartTime = newrelic.FromContext(ctx).StartSegmentNow()
	segment.ParameterizedQuery = data.SQL
	if t.SendQueryParameters {
		segment.QueryParameters = t.getQueryParameters(data.Args)
	}

	// fill Operation and Collection
	t.ParseQuery(&segment, data.SQL)
	if newrelic.IsSecurityAgentPresent() {
		stoken := newrelic.GetSecurityAgentInterface().SendEvent("SQL", data.SQL, data.Args)
		ctx = context.WithValue(ctx, querySecurityKey, stoken)
	}

	return context.WithValue(ctx, querySegmentKey, &segment)
}

// TraceQueryEnd is called by pgx/v5 at the completion of Query, QueryRow, and Exec calls.
// This will terminate the datastore segment started when the database operation was started.
func (t *Tracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	segment, ok := ctx.Value(querySegmentKey).(*newrelic.DatastoreSegment)
	if !ok {
		return
	}
	if newrelic.IsSecurityAgentPresent() {
		if stoken := ctx.Value(querySecurityKey); stoken != nil {
			newrelic.GetSecurityAgentInterface().SendExitEvent(stoken, nil)
			ctx = context.WithValue(ctx, querySecurityKey, nil)
		}
	}
	segment.End()
}

func (t *Tracer) getQueryParameters(args []interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for i, arg := range args {
		result["$"+strconv.Itoa(i)] = arg
	}
	return result
}

// TraceBatchStart is called at the beginning of SendBatch calls. The returned context is used for the
// rest of the call and will be passed to TraceBatchQuery and TraceBatchEnd.
func (t *Tracer) TraceBatchStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchStartData) context.Context {
	segment := t.BaseSegment
	segment.StartTime = newrelic.FromContext(ctx).StartSegmentNow()
	segment.Operation = "batch"
	segment.Collection = ""

	return context.WithValue(ctx, batchSegmentKey, &segment)
}

// TraceBatchQuery is called for each batched query operation. We will add the SQL statement to the segment's
// ParameterizedQuery value.
func (t *Tracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	segment, ok := ctx.Value(batchSegmentKey).(*newrelic.DatastoreSegment)
	if !ok {
		return
	}

	segment.ParameterizedQuery += data.SQL + "\n"
}

// TraceBatchEnd is called at the end of a batch. Here we will terminate the datastore segment we started when
// the batch was started.
func (t *Tracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData) {
	segment, ok := ctx.Value(batchSegmentKey).(*newrelic.DatastoreSegment)
	if !ok {
		return
	}
	segment.End()
}

// TracePrepareStart is called at the beginning of Prepare calls. The returned context is used for the
// rest of the call and will be passed to TracePrepareEnd.
//
// The Query and QueryRow will call prepare, so here we don't do any additional work (otherwise
// we'd duplicate segment data).
func (t *Tracer) TracePrepareStart(ctx context.Context, conn *pgx.Conn, data pgx.TracePrepareStartData) context.Context {
	return ctx
}

// TracePrepareEnd implements pgx.PrepareTracer.
func (t *Tracer) TracePrepareEnd(ctx context.Context, conn *pgx.Conn, data pgx.TracePrepareEndData) {
}
