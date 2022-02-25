// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// +build go1.10

package newrelic

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
)

var (
	driverTestMetrics = []internal.WantMetric{
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/operation/MySQL/myoperation", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/mycollection/myoperation", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/mycollection/myoperation", Scope: "OtherTransaction/Go/hello", Forced: false, Data: nil},
		{Name: "Datastore/instance/MySQL/myhost/myport", Scope: "", Forced: false, Data: nil},
	}
)

type testDriver struct{}
type testConnector struct{}
type testConn struct{}
type testStmt struct{}

func (d testDriver) OpenConnector(name string) (driver.Connector, error) { return testConnector{}, nil }
func (d testDriver) Open(name string) (driver.Conn, error)               { return testConn{}, nil }

func (c testConnector) Connect(context.Context) (driver.Conn, error) { return testConn{}, nil }
func (c testConnector) Driver() driver.Driver                        { return testDriver{} }

func (c testConn) Prepare(query string) (driver.Stmt, error) { return testStmt{}, nil }
func (c testConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return testStmt{}, nil
}
func (c testConn) Close() error              { return nil }
func (c testConn) Begin() (driver.Tx, error) { return nil, nil }
func (c testConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return nil, nil
}
func (c testConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return nil, nil
}

func (s testStmt) Close() error {
	return nil
}
func (s testStmt) NumInput() int {
	return 0
}
func (s testStmt) Exec(args []driver.Value) (driver.Result, error) {
	return nil, nil
}
func (s testStmt) Query(args []driver.Value) (driver.Rows, error) {
	return nil, nil
}
func (s testStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	return nil, nil
}
func (s testStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	return nil, nil
}

var (
	testBuilder = SQLDriverSegmentBuilder{
		BaseSegment: DatastoreSegment{
			Product: DatastoreMySQL,
		},
		ParseDSN: func(segment *DatastoreSegment, dsn string) {
			fields := strings.Split(dsn, ",")
			segment.Host = fields[0]
			segment.PortPathOrID = fields[1]
			segment.DatabaseName = fields[2]
		},
		ParseQuery: func(segment *DatastoreSegment, query string) {
			fields := strings.Split(query, ",")
			segment.Operation = fields[0]
			segment.Collection = fields[1]
		},
	}
)

func TestDriverStmtExecContext(t *testing.T) {
	// Test that driver.Stmt.ExecContext calls get instrumented.
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	dr := InstrumentSQLDriver(testDriver{}, testBuilder)
	txn := app.StartTransaction("hello")
	conn, _ := dr.Open("myhost,myport,mydatabase")
	stmt, _ := conn.Prepare("myoperation,mycollection")
	ctx := NewContext(context.Background(), txn)
	stmt.(driver.StmtExecContext).ExecContext(ctx, nil)
	txn.End()
	app.ExpectMetrics(t, driverTestMetrics)
}

func TestDriverStmtQueryContext(t *testing.T) {
	// Test that driver.Stmt.PrepareContext calls get instrumented.
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	dr := InstrumentSQLDriver(testDriver{}, testBuilder)
	txn := app.StartTransaction("hello")
	conn, _ := dr.Open("myhost,myport,mydatabase")
	stmt, _ := conn.(driver.ConnPrepareContext).PrepareContext(nil, "myoperation,mycollection")
	ctx := NewContext(context.Background(), txn)
	stmt.(driver.StmtQueryContext).QueryContext(ctx, nil)
	txn.End()
	app.ExpectMetrics(t, driverTestMetrics)
}

func TestDriverConnExecContext(t *testing.T) {
	// Test that driver.Conn.ExecContext calls get instrumented.
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	dr := InstrumentSQLDriver(testDriver{}, testBuilder)
	txn := app.StartTransaction("hello")
	conn, _ := dr.Open("myhost,myport,mydatabase")
	ctx := NewContext(context.Background(), txn)
	conn.(driver.ExecerContext).ExecContext(ctx, "myoperation,mycollection", nil)
	txn.End()
	app.ExpectMetrics(t, driverTestMetrics)
}

func TestDriverConnQueryContext(t *testing.T) {
	// Test that driver.Conn.QueryContext calls get instrumented.
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	dr := InstrumentSQLDriver(testDriver{}, testBuilder)
	txn := app.StartTransaction("hello")
	conn, _ := dr.Open("myhost,myport,mydatabase")
	ctx := NewContext(context.Background(), txn)
	conn.(driver.QueryerContext).QueryContext(ctx, "myoperation,mycollection", nil)
	txn.End()
	app.ExpectMetrics(t, driverTestMetrics)
}

func TestDriverContext(t *testing.T) {
	// Test that driver.OpenConnector returns an instrumented connector.
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	dr := InstrumentSQLDriver(testDriver{}, testBuilder)
	txn := app.StartTransaction("hello")
	connector, _ := dr.(driver.DriverContext).OpenConnector("myhost,myport,mydatabase")
	conn, _ := connector.Connect(nil)
	ctx := NewContext(context.Background(), txn)
	conn.(driver.ExecerContext).ExecContext(ctx, "myoperation,mycollection", nil)
	txn.End()
	app.ExpectMetrics(t, driverTestMetrics)
}

func TestInstrumentSQLConnector(t *testing.T) {
	// Test that connections returned by an instrumented driver.Connector
	// are instrumented.
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	bld := testBuilder
	bld.BaseSegment.Host = "myhost"
	bld.BaseSegment.PortPathOrID = "myport"
	bld.BaseSegment.DatabaseName = "mydatabase"
	connector := InstrumentSQLConnector(testConnector{}, bld)
	txn := app.StartTransaction("hello")
	conn, _ := connector.Connect(nil)
	ctx := NewContext(context.Background(), txn)
	conn.(driver.ExecerContext).ExecContext(ctx, "myoperation,mycollection", nil)
	txn.End()
	app.ExpectMetrics(t, driverTestMetrics)
}

func TestConnectorToDriver(t *testing.T) {
	// Test that driver.Connector.Driver returns an instrumented Driver.
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	connector := InstrumentSQLConnector(testConnector{}, testBuilder)
	txn := app.StartTransaction("hello")
	dr := connector.Driver()
	conn, _ := dr.Open("myhost,myport,mydatabase")
	ctx := NewContext(context.Background(), txn)
	conn.(driver.ExecerContext).ExecContext(ctx, "myoperation,mycollection", nil)
	txn.End()
	app.ExpectMetrics(t, driverTestMetrics)
}

type testConnectorErr struct {
	testConnector
}

func (c testConnectorErr) Connect(context.Context) (driver.Conn, error) { return testConnErr{}, nil }

type testConnErr struct {
	testConn
}

func (c testConnErr) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return nil, driver.ErrSkip
}

func (c testConnErr) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return nil, driver.ErrSkip
}

// Ensure that if the driver used returns driver.ErrSkip that spans still have correct parentage
func TestExecContextErrSkipReturned(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	connector := InstrumentSQLConnector(testConnectorErr{}, testBuilder)
	txn := app.StartTransaction("hello")
	conn, _ := connector.Connect(nil)
	ctx := NewContext(context.Background(), txn)

	conn.(driver.ExecerContext).ExecContext(ctx, "myoperation,mycollection", nil)
	txn.StartSegment("second").End()
	txn.End()

	parentGUID := "4981855ad8681d0d"
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":     "Custom/second",
				"parentId": parentGUID,
				"category": "generic",
			},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"guid":             parentGUID,
				"nr.entryPoint":    true,
				"category":         "generic",
				"transaction.name": "OtherTransaction/Go/hello",
			},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

// Ensure that if the driver used returns driver.ErrSkip that spans still have correct parentage
func TestQueryContextErrSkipReturned(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	connector := InstrumentSQLConnector(testConnectorErr{}, testBuilder)
	txn := app.StartTransaction("hello")
	conn, _ := connector.Connect(nil)
	ctx := NewContext(context.Background(), txn)
	conn.(driver.QueryerContext).QueryContext(ctx, "myoperation,mycollection", nil)
	txn.StartSegment("second").End()
	txn.End()
	parentGUID := "4981855ad8681d0d"
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":     "Custom/second",
				"parentId": parentGUID,
				"category": "generic",
			},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"guid":             parentGUID,
				"nr.entryPoint":    true,
				"category":         "generic",
				"transaction.name": "OtherTransaction/Go/hello",
			},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

// Ensure we don't panic if the txn is nil
func TestSQLNoTxnNoCry(t *testing.T) {
	connector := InstrumentSQLConnector(testConnector{}, testBuilder)
	conn, _ := connector.Connect(nil)
	conn.(driver.QueryerContext).QueryContext(context.Background(), "myoperation,mycollection", nil)
}
