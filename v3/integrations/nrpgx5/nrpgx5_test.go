package nrpgx5

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
	"github.com/stretchr/testify/assert"
)

func TestTracer_Trace_CRUD(t *testing.T) {
	con, finish := getTestCon()
	defer finish()

	tests := []struct {
		name   string
		fn     func(context.Context, *pgx.Conn)
		metric []internal.WantMetric
	}{
		{
			name: "query should send the metric after the row close",
			fn: func(ctx context.Context, con *pgx.Conn) {
				rows, _ := con.Query(ctx, "SELECT id, name, timestamp FROM mytable LIMIT $1", 2)
				rows.Close()
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/operation/Postgres/select"},
				{Name: "Datastore/statement/Postgres/mytable/select"},
			},
		},
		{
			name: "queryrow should send the metric after scan",
			fn: func(ctx context.Context, con *pgx.Conn) {
				row := con.QueryRow(ctx, "SELECT id, name, timestamp FROM mytable")
				_ = row.Scan()
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/operation/Postgres/select"},
				{Name: "Datastore/statement/Postgres/mytable/select"},
			},
		},
		{
			name: "insert should send the metric",
			fn: func(ctx context.Context, con *pgx.Conn) {
				_, _ = con.Exec(ctx, "INSERT INTO mytable(name) VALUES ($1)", "myname is")
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/operation/Postgres/insert"},
				{Name: "Datastore/statement/Postgres/mytable/insert"},
			},
		},
		{
			name: "update should send the metric",
			fn: func(ctx context.Context, con *pgx.Conn) {
				_, _ = con.Exec(ctx, "UPDATE mytable set name = $2 WHERE id = $1", 1, "myname is")
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/operation/Postgres/update"},
				{Name: "Datastore/statement/Postgres/mytable/update"},
			},
		},
		{
			name: "delete should send the metric",
			fn: func(ctx context.Context, con *pgx.Conn) {
				_, _ = con.Exec(ctx, "DELETE FROM mytable WHERE id = $1", 4)
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/operation/Postgres/delete"},
				{Name: "Datastore/statement/Postgres/mytable/delete"},
			},
		},
		{
			name: "select 1 should send the metric",
			fn: func(ctx context.Context, con *pgx.Conn) {
				_, _ = con.Exec(ctx, "SELECT 1")
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/operation/Postgres/select"},
			},
		},
		{
			name: "query error should also send the metric",
			fn: func(ctx context.Context, con *pgx.Conn) {
				_, _ = con.Query(ctx, "SELECT * FROM non_existent_table")
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/operation/Postgres/select"},
				{Name: "Datastore/statement/Postgres/non_existent_table/select"},
			},
		},
		{
			name: "exec error should also send the metric",
			fn: func(ctx context.Context, con *pgx.Conn) {
				_, _ = con.Exec(ctx, "INSERT INTO non_existent_table(name) VALUES ($1)", "wrong name")
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/operation/Postgres/insert"},
				{Name: "Datastore/statement/Postgres/non_existent_table/insert"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := integrationsupport.NewBasicTestApp()
			txn := app.StartTransaction(t.Name())
			ctx := newrelic.NewContext(context.Background(), txn)

			tt.fn(ctx, con)

			txn.End()
			app.ExpectMetricsPresent(t, tt.metric)
		})
	}
}

func TestTracer_connect(t *testing.T) {
	conn, finish := getTestCon()
	defer finish()

	cfg := conn.Config()
	tracer := cfg.Tracer.(*Tracer)

	// hostname will
	t.Run("connect should set tracer host port and database", func(t *testing.T) {
		assert.Equal(t, cfg.Host, tracer.BaseSegment.Host)
		assert.Equal(t, cfg.Database, tracer.BaseSegment.DatabaseName)
		assert.Equal(t, strconv.FormatUint(uint64(cfg.Port), 10), tracer.BaseSegment.PortPathOrID)
	})

	t.Run("exec should send metric with instance host and port ", func(t *testing.T) {
		app := integrationsupport.NewBasicTestApp()

		txn := app.StartTransaction(t.Name())

		ctx := newrelic.NewContext(context.Background(), txn)
		_, _ = conn.Exec(ctx, "INSERT INTO mytable(name) VALUES ($1)", "myname is")

		txn.End()

		app.ExpectMetricsPresent(t, []internal.WantMetric{
			{Name: "Datastore/instance/Postgres/" + os.Getenv("PG_HOST") + "/" + tracer.BaseSegment.PortPathOrID},
		})
	})
}

func TestTracer_batch(t *testing.T) {
	conn, finish := getTestCon()
	defer finish()

	cfg := conn.Config()
	tracer := cfg.Tracer.(*Tracer)

	t.Run("exec should send metric with instance host and port ", func(t *testing.T) {
		app := integrationsupport.NewBasicTestApp()

		txn := app.StartTransaction(t.Name())

		ctx := newrelic.NewContext(context.Background(), txn)
		batch := &pgx.Batch{}
		_ = batch.Queue("INSERT INTO mytable(name) VALUES ($1)", "name a")
		_ = batch.Queue("INSERT INTO mytable(name) VALUES ($1)", "name b")
		_ = batch.Queue("INSERT INTO mytable(name) VALUES ($1)", "name c")
		_ = batch.Queue("SELECT id FROM mytable ORDER by id DESC LIMIT 1")
		result := conn.SendBatch(ctx, batch)

		_ = result.Close()

		txn.End()

		app.ExpectMetricsPresent(t, []internal.WantMetric{
			{Name: "Datastore/instance/Postgres/" + os.Getenv("PG_HOST") + "/" + tracer.BaseSegment.PortPathOrID},
			{Name: "Datastore/operation/Postgres/batch"},
		})
	})
}

func TestTracer_inPool(t *testing.T) {
	dsn := getDSN()

	cfg, _ := pgxpool.ParseConfig(dsn)
	cfg.ConnConfig.Tracer = NewTracer()

	u, _ := url.Parse(dsn)

	con, _ := pgxpool.NewWithConfig(context.Background(), cfg)

	tests := []struct {
		name   string
		fn     func(context.Context, *pgxpool.Pool)
		metric []internal.WantMetric
	}{
		{
			name: "query should send the metric after the row close",
			fn: func(ctx context.Context, con *pgxpool.Pool) {
				rows, _ := con.Query(ctx, "SELECT id, name, timestamp FROM mytable LIMIT $1", 2)
				rows.Close()
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/operation/Postgres/select"},
				{Name: "Datastore/statement/Postgres/mytable/select"},
			},
		},
		{
			name: "queryrow should send the metric after scan",
			fn: func(ctx context.Context, con *pgxpool.Pool) {
				row := con.QueryRow(ctx, "SELECT id, name, timestamp FROM mytable")
				_ = row.Scan()
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/operation/Postgres/select"},
				{Name: "Datastore/statement/Postgres/mytable/select"},
			},
		},
		{
			name: "insert should send the metric",
			fn: func(ctx context.Context, con *pgxpool.Pool) {
				_, _ = con.Exec(ctx, "INSERT INTO mytable(name) VALUES ($1)", "myname is")
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/operation/Postgres/insert"},
				{Name: "Datastore/statement/Postgres/mytable/insert"},
			},
		},
		{
			name: "update should send the metric",
			fn: func(ctx context.Context, con *pgxpool.Pool) {
				_, _ = con.Exec(ctx, "UPDATE mytable set name = $2 WHERE id = $1", 1, "myname is")
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/operation/Postgres/update"},
				{Name: "Datastore/statement/Postgres/mytable/update"},
			},
		},
		{
			name: "delete should send the metric",
			fn: func(ctx context.Context, con *pgxpool.Pool) {
				_, _ = con.Exec(ctx, "DELETE FROM mytable WHERE id = $1", 4)
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/operation/Postgres/delete"},
				{Name: "Datastore/statement/Postgres/mytable/delete"},
			},
		},
		{
			name: "select 1 should send the metric",
			fn: func(ctx context.Context, con *pgxpool.Pool) {
				_, _ = con.Exec(ctx, "SELECT 1")
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/operation/Postgres/select"},
			},
		},
		{
			name: "metric should send the metric database instance",
			fn: func(ctx context.Context, con *pgxpool.Pool) {
				_, _ = con.Exec(ctx, "SELECT 1")
			},
			metric: []internal.WantMetric{
				{Name: "Datastore/instance/Postgres/" + os.Getenv("PG_HOST") + "/" + u.Port()},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := integrationsupport.NewBasicTestApp()
			txn := app.StartTransaction(t.Name())
			ctx := newrelic.NewContext(context.Background(), txn)

			tt.fn(ctx, con)

			txn.End()
			app.ExpectMetricsPresent(t, tt.metric)
		})
	}
}

func getTestCon() (*pgx.Conn, func()) {
	dsn := getDSN()

	cfg, _ := pgx.ParseConfig(dsn)

	cfg.Tracer = NewTracer()

	con, _ := pgx.ConnectConfig(context.Background(), cfg)

	return con, func() {
		_ = con.Close(context.Background())
	}
}

func getDSN() string {
	pgHost := os.Getenv("PG_HOST")
	pgPort := os.Getenv("PG_PORT")
	pgUser := os.Getenv("PG_USER")
	pgPass := os.Getenv("PG_PW")
	pgDB := os.Getenv("PG_DB")
	pgParam := os.Getenv("PG_PARAM")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s%s", pgUser, pgPass, pgHost, pgPort, pgDB, pgParam)

	return dsn
}
