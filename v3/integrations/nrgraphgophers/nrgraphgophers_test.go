// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrgraphgophers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/introspection"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func TestFieldManagementSync(t *testing.T) {
	tracer := NewTracer().(*tracer)
	id1 := tracer.newRequestID()
	id2 := tracer.newRequestID()
	if id1 == id2 {
		t.Fatal(id1, id2)
	}
	if async := tracer.startField(id1); async {
		t.Fatal(async)
	}
	if async := tracer.startField(id2); async {
		t.Fatal(async)
	}
	tracer.stopField(id1)
	if async := tracer.startField(id1); async {
		t.Fatal(async)
	}
	tracer.stopField(id2)
	tracer.stopField(id1)
	if tracer.activeFields[id1] != 0 || tracer.activeFields[id2] != 0 {
		t.Fatal(tracer.activeFields)
	}
	tracer.removeFields(id2)
	tracer.removeFields(id1)
	if len(tracer.activeFields) != 0 {
		t.Fatal(tracer.activeFields)
	}
}

func TestFieldManagementAsync(t *testing.T) {
	tracer := NewTracer().(*tracer)
	id1 := tracer.newRequestID()
	id2 := tracer.newRequestID()
	if id1 == id2 {
		t.Fatal(id1, id2)
	}
	if async := tracer.startField(id1); async {
		t.Fatal(async)
	}
	if async := tracer.startField(id1); !async {
		t.Fatal(async)
	}
	if async := tracer.startField(id2); async {
		t.Fatal(async)
	}
	tracer.stopField(id1)
	if async := tracer.startField(id1); !async {
		t.Fatal(async)
	}
	tracer.stopField(id2)
	tracer.stopField(id1)
	tracer.stopField(id1)
	if tracer.activeFields[id1] != 0 || tracer.activeFields[id2] != 0 {
		t.Fatal(tracer.activeFields)
	}
	tracer.removeFields(id2)
	tracer.removeFields(id1)
	if len(tracer.activeFields) != 0 {
		t.Fatal(tracer.activeFields)
	}
}

func TestQueryWithAsyncFields(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	tracer := NewTracer()
	txn := app.StartTransaction("hello")
	txn.SetWebRequestHTTP(nil)
	ctx := newrelic.NewContext(context.Background(), txn)
	ctx, queryFinish := tracer.TraceQuery(ctx, "queryString", "MyOperation", map[string]interface{}{}, map[string]*introspection.Type{})

	_, fieldFinish1 := tracer.TraceField(ctx, "label", "typeName", "field1", true, map[string]interface{}{})
	_, fieldFinish2 := tracer.TraceField(ctx, "label", "typeName", "field2", true, map[string]interface{}{})

	fieldFinish1(nil)
	fieldFinish2(nil)
	queryFinish(nil)

	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction"},
		{Name: "WebTransaction/Go/hello"},
		{Name: "WebTransactionTotalTime"},
		{Name: "WebTransactionTotalTime/Go/hello"},
		{Name: "Apdex"},
		{Name: "Apdex/Go/hello"},
		{Name: "HttpDispatcher"},
		{Name: "Custom/MyOperation"},
		{Name: "Custom/MyOperation", Scope: "WebTransaction/Go/hello"},
		{Name: "Custom/field2"},
		{Name: "Custom/field2", Scope: "WebTransaction/Go/hello"},
		{Name: "Custom/field1"},
		{Name: "Custom/field1", Scope: "WebTransaction/Go/hello"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Forced: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Forced: nil},
	})
}

type query struct{}

func (*query) Hello() string { return "hello world" }

func (*query) Problem() (string, error) { return "", errors.New("something went wrong") }

func (*query) Zip() string { return "zip" }

func (*query) Zap() string { return "zap" }

const (
	querySchema = `type Query {
		hello: String!
		problem: String!
		zip: String!
		zap: String!
	}`
)

func TestQueryRequest(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	opt := graphql.Tracer(NewTracer())
	schema := graphql.MustParseSchema(querySchema, &query{}, opt)
	handler := &relay.Handler{Schema: schema}
	mux := http.NewServeMux()
	mux.Handle(newrelic.WrapHandle(app.Application, "/", handler))
	body := `{
			"query": "query HelloOperation { hello }",
			"operationName": "HelloOperation"
		}`
	req, err := http.NewRequest("POST", "/", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)
	if b := rw.Body.String(); b != `{"data":{"hello":"hello world"}}` {
		t.Error(b)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction"},
		{Name: "WebTransaction/Go/POST /"},
		{Name: "WebTransactionTotalTime"},
		{Name: "WebTransactionTotalTime/Go/POST /"},
		{Name: "Apdex"},
		{Name: "Apdex/Go/POST /"},
		{Name: "HttpDispatcher"},
		{Name: "Custom/HelloOperation"},
		{Name: "Custom/HelloOperation", Scope: "WebTransaction/Go/POST /"},
		{Name: "Custom/hello"},
		{Name: "Custom/hello", Scope: "WebTransaction/Go/POST /"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Forced: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Forced: nil},
	})
}

func TestQueryRequestUnknownOperation(t *testing.T) {
	// Test the situation where "operationName" is not provided in the
	// request body.
	app := integrationsupport.NewBasicTestApp()
	opt := graphql.Tracer(NewTracer())
	schema := graphql.MustParseSchema(querySchema, &query{}, opt)
	handler := &relay.Handler{Schema: schema}
	mux := http.NewServeMux()
	mux.Handle(newrelic.WrapHandle(app.Application, "/", handler))
	body := `{
			"query": "query HelloOperation { hello }"
		}`
	req, err := http.NewRequest("POST", "/", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)
	if b := rw.Body.String(); b != `{"data":{"hello":"hello world"}}` {
		t.Error(b)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction"},
		{Name: "WebTransaction/Go/POST /"},
		{Name: "WebTransactionTotalTime"},
		{Name: "WebTransactionTotalTime/Go/POST /"},
		{Name: "Apdex"},
		{Name: "Apdex/Go/POST /"},
		{Name: "HttpDispatcher"},
		{Name: "Custom/HelloOperation"},
		{Name: "Custom/HelloOperation", Scope: "WebTransaction/Go/POST /"},
		{Name: "Custom/hello"},
		{Name: "Custom/hello", Scope: "WebTransaction/Go/POST /"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Forced: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Forced: nil},
	})
}

func TestQueryRequestError(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	opt := graphql.Tracer(NewTracer())
	schema := graphql.MustParseSchema(querySchema, &query{}, opt)
	handler := &relay.Handler{Schema: schema}
	mux := http.NewServeMux()
	mux.Handle(newrelic.WrapHandle(app.Application, "/", handler))
	body := `{
			"query": "query ProblemOperation { problem }",
			"operationName": "ProblemOperation"
		}`
	req, err := http.NewRequest("POST", "/", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)
	if b := rw.Body.String(); b != `{"errors":[{"message":"something went wrong","path":["problem"]}],"data":null}` {
		t.Error(b)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction"},
		{Name: "WebTransaction/Go/POST /"},
		{Name: "WebTransactionTotalTime"},
		{Name: "WebTransactionTotalTime/Go/POST /"},
		{Name: "Apdex"},
		{Name: "Apdex/Go/POST /"},
		{Name: "HttpDispatcher"},
		{Name: "Custom/ProblemOperation"},
		{Name: "Custom/ProblemOperation", Scope: "WebTransaction/Go/POST /"},
		{Name: "Custom/problem"},
		{Name: "Custom/problem", Scope: "WebTransaction/Go/POST /"},
		{Name: "Errors/all"},
		{Name: "Errors/allWeb"},
		{Name: "Errors/WebTransaction/Go/POST /"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Forced: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Forced: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/all"},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/allWeb"},
	})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/POST /",
		Msg:     "graphql: something went wrong",
	}})
}

func TestQueryRequestMultipleFields(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	opt := graphql.Tracer(NewTracer())
	schema := graphql.MustParseSchema(querySchema, &query{}, opt)
	handler := &relay.Handler{Schema: schema}
	mux := http.NewServeMux()
	mux.Handle(newrelic.WrapHandle(app.Application, "/", handler))
	body := `{
			"query": "query Multiple { zip zap }",
			"operationName": "Multiple"
		}`
	req, err := http.NewRequest("POST", "/", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)
	if b := rw.Body.String(); b != `{"data":{"zip":"zip","zap":"zap"}}` {
		t.Error(b)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction"},
		{Name: "WebTransaction/Go/POST /"},
		{Name: "WebTransactionTotalTime"},
		{Name: "WebTransactionTotalTime/Go/POST /"},
		{Name: "Apdex"},
		{Name: "Apdex/Go/POST /"},
		{Name: "HttpDispatcher"},
		{Name: "Custom/Multiple"},
		{Name: "Custom/Multiple", Scope: "WebTransaction/Go/POST /"},
		{Name: "Custom/zip"},
		{Name: "Custom/zip", Scope: "WebTransaction/Go/POST /"},
		{Name: "Custom/zap"},
		{Name: "Custom/zap", Scope: "WebTransaction/Go/POST /"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Forced: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allWeb", Forced: nil},
	})
}
