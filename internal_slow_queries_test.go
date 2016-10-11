package newrelic

import (
	"strings"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal"
)

func TestSlowQueryBasic(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "unknown",
		PortPathOrID: "unknown",
	}})
}

func TestSlowQueryLocallyDisabled(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
		cfg.DatastoreTracer.SlowQuery.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{})
}

func TestSlowQueryRemotelyDisabled(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	replyfn := func(reply *internal.ConnectReply) {
		reply.CollectTraces = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{})
}

func TestSlowQueryBelowThreshold(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 1 * time.Hour
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{})
}

func TestSlowQueryDatabaseProvided(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		DatabaseName:       "my_database",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "my_database",
		Host:         "unknown",
		PortPathOrID: "unknown",
	}})
}

func TestSlowQueryHostProvided(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		Host:               "db-server-1",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "db-server-1",
		PortPathOrID: "unknown",
	}})
}

func TestSlowQueryPortProvided(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		PortPathOrID:       "98021",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "unknown",
		PortPathOrID: "98021",
	}})
}

func TestSlowQueryHostPortProvided(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		Host:               "db-server-1",
		PortPathOrID:       "98021",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "db-server-1",
		PortPathOrID: "98021",
	}})
	scope := "WebTransaction/Go/myName"
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Datastore/all", "", true, nil},
		{"Datastore/allWeb", "", true, nil},
		{"Datastore/MySQL/all", "", true, nil},
		{"Datastore/MySQL/allWeb", "", true, nil},
		{"Datastore/operation/MySQL/INSERT", "", false, nil},
		{"Datastore/statement/MySQL/users/INSERT", "", false, nil},
		{"Datastore/statement/MySQL/users/INSERT", scope, false, nil},
		{"Datastore/instance/MySQL/db-server-1/98021", "", false, nil},
	})
}

func TestSlowQueryAggregation(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
	}.End()
	DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
	}.End()
	DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastorePostgres,
		Collection:         "products",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO products (name, price) VALUES ($1, $2)",
	}.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        2,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "unknown",
		PortPathOrID: "unknown",
	}, {
		Count:        1,
		MetricName:   "Datastore/statement/Postgres/products/INSERT",
		Query:        "INSERT INTO products (name, price) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "unknown",
		PortPathOrID: "unknown",
	},
	})
}

func TestSlowQueryMissingQuery(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:  StartSegmentNow(txn),
		Product:    DatastoreMySQL,
		Collection: "users",
		Operation:  "INSERT",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "'INSERT' on 'users' using 'MySQL'",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "unknown",
		PortPathOrID: "unknown",
	}})
}

func TestSlowQueryMissingEverything(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime: StartSegmentNow(txn),
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/operation/Unknown/other",
		Query:        "'other' on 'unknown' using 'Unknown'",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "unknown",
		PortPathOrID: "unknown",
	}})
	scope := "WebTransaction/Go/myName"
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Datastore/all", "", true, nil},
		{"Datastore/allWeb", "", true, nil},
		{"Datastore/Unknown/all", "", true, nil},
		{"Datastore/Unknown/allWeb", "", true, nil},
		{"Datastore/operation/Unknown/other", "", false, nil},
		{"Datastore/operation/Unknown/other", scope, false, nil},
		{"Datastore/instance/Unknown/unknown/unknown", "", false, nil},
	})
}

func TestSlowQueryWithQueryParameters(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	params := map[string]interface{}{
		"str": "zap",
		"int": 123,
	}
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		QueryParameters:    params,
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "unknown",
		PortPathOrID: "unknown",
		Params:       params,
	}})
}

func TestSlowQueryHighSecurity(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
		cfg.HighSecurity = true
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	params := map[string]interface{}{
		"str": "zap",
		"int": 123,
	}
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		QueryParameters:    params,
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "unknown",
		PortPathOrID: "unknown",
		Params:       nil,
	}})
}

func TestSlowQueryInvalidParameters(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	params := map[string]interface{}{
		"str":                               "zap",
		"int":                               123,
		"invalid_value":                     struct{}{},
		strings.Repeat("key-too-long", 100): 1,
		"long-key":                          strings.Repeat("A", 300),
	}
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		QueryParameters:    params,
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "unknown",
		PortPathOrID: "unknown",
		Params: map[string]interface{}{
			"str":      "zap",
			"int":      123,
			"long-key": strings.Repeat("A", 255),
		},
	}})
}

func TestSlowQueryParametersDisabled(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
		cfg.DatastoreTracer.QueryParameters.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	params := map[string]interface{}{
		"str": "zap",
		"int": 123,
	}
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		QueryParameters:    params,
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "unknown",
		PortPathOrID: "unknown",
		Params:       nil,
	}})
}

func TestSlowQueryInstanceDisabledUnknown(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
		cfg.DatastoreTracer.InstanceReporting.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "",
		PortPathOrID: "",
	}})
	scope := "WebTransaction/Go/myName"
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Datastore/all", "", true, nil},
		{"Datastore/allWeb", "", true, nil},
		{"Datastore/MySQL/all", "", true, nil},
		{"Datastore/MySQL/allWeb", "", true, nil},
		{"Datastore/operation/MySQL/INSERT", "", false, nil},
		{"Datastore/statement/MySQL/users/INSERT", "", false, nil},
		{"Datastore/statement/MySQL/users/INSERT", scope, false, nil},
	})
}

func TestSlowQueryInstanceDisabledLocalhost(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
		cfg.DatastoreTracer.InstanceReporting.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		Host:               "localhost",
		PortPathOrID:       "3306",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "",
		PortPathOrID: "",
	}})
	scope := "WebTransaction/Go/myName"
	app.ExpectMetrics(t, []internal.WantMetric{
		{"WebTransaction/Go/myName", "", true, nil},
		{"WebTransaction", "", true, nil},
		{"HttpDispatcher", "", true, nil},
		{"Apdex", "", true, nil},
		{"Apdex/Go/myName", "", false, nil},
		{"Datastore/all", "", true, nil},
		{"Datastore/allWeb", "", true, nil},
		{"Datastore/MySQL/all", "", true, nil},
		{"Datastore/MySQL/allWeb", "", true, nil},
		{"Datastore/operation/MySQL/INSERT", "", false, nil},
		{"Datastore/statement/MySQL/users/INSERT", "", false, nil},
		{"Datastore/statement/MySQL/users/INSERT", scope, false, nil},
	})
}

func TestSlowQueryDatabaseNameDisabled(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
		cfg.DatastoreTracer.DatabaseNameReporting.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		DatabaseName:       "db-server-1",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "unknown",
		PortPathOrID: "unknown",
	}})
}
