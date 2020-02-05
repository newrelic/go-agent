package nrgraphql

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var schema = func() graphql.Schema {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "RootQuery",
			Fields: graphql.Fields{
				// fields will get executed alphabetically
				"a_hello": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return "world", nil
					},
				},
				"b_errors": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return nil, errors.New("ooooooops")
					},
				},
				"c_goodbye": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return "world", nil
					},
				},
			},
		}),
		Extensions: []graphql.Extension{NewExtension()},
	})
	if err != nil {
		panic(err)
	}
	return schema
}()

func TestExtensionNoTransaction(t *testing.T) {
	query := `{ a_hello }`
	params := graphql.Params{Schema: schema, RequestString: query}
	resp := graphql.Do(params)
	for _, err := range resp.Errors {
		t.Error("failure to Do:", err)
	}
	js, err := json.Marshal(resp.Data)
	if err != nil {
		t.Error("failure to marshal json:", err)
	}
	if data := string(js); data != `{"a_hello":"world"}` {
		t.Error("incorrect response data:", data)
	}
}

func TestExtensionWithTransaction(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	txn := app.StartTransaction("query")
	ctx := newrelic.NewContext(context.Background(), txn)

	query := `{ a_hello }`
	params := graphql.Params{
		Schema:        schema,
		RequestString: query,
		Context:       ctx,
	}
	resp := graphql.Do(params)
	for _, err := range resp.Errors {
		t.Error("failure to Do:", err)
	}
	js, err := json.Marshal(resp.Data)
	if err != nil {
		t.Error("failure to marshal json:", err)
	}
	if data := string(js); data != `{"a_hello":"world"}` {
		t.Error("incorrect response data:", data)
	}

	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Custom/Execution", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Execution", Scope: "OtherTransaction/Go/query", Forced: false, Data: nil},
		{Name: "Custom/Parse", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Parse", Scope: "OtherTransaction/Go/query", Forced: false, Data: nil},
		{Name: "Custom/Resolve a_hello", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Resolve a_hello", Scope: "OtherTransaction/Go/query", Forced: false, Data: nil},
		{Name: "Custom/Validation", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Validation", Scope: "OtherTransaction/Go/query", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/query", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/query", Scope: "", Forced: false, Data: nil},
	})
}

func TestExtensionResolveErrorFirst(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	txn := app.StartTransaction("query")
	ctx := newrelic.NewContext(context.Background(), txn)

	query := `{ b_errors c_goodbye }`
	params := graphql.Params{
		Schema:        schema,
		RequestString: query,
		Context:       ctx,
	}
	resp := graphql.Do(params)
	if len(resp.Errors) != 1 {
		t.Error("incorrect number of errors on response", resp.Errors)
	}
	js, err := json.Marshal(resp.Data)
	if err != nil {
		t.Error("failure to marshal json:", err)
	}
	if data := string(js); data != `{"b_errors":null,"c_goodbye":"world"}` {
		t.Error("incorrect response data:", data)
	}

	txn.End()
	// if Resolve becomes asynchronous, an error will be logged saying that we
	// tried to end a segment twice.
	app.ExpectNoLoggedErrors(t)
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Custom/Execution", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Execution", Scope: "OtherTransaction/Go/query", Forced: false, Data: nil},
		{Name: "Custom/Parse", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Parse", Scope: "OtherTransaction/Go/query", Forced: false, Data: nil},
		{Name: "Custom/Resolve b_errors", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Resolve b_errors", Scope: "OtherTransaction/Go/query", Forced: false, Data: nil},
		{Name: "Custom/Resolve c_goodbye", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Resolve c_goodbye", Scope: "OtherTransaction/Go/query", Forced: false, Data: nil},
		{Name: "Custom/Validation", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Validation", Scope: "OtherTransaction/Go/query", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/query", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/query", Scope: "", Forced: false, Data: nil},
	})
}

func TestExtensionResolveErrorLast(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	txn := app.StartTransaction("query")
	ctx := newrelic.NewContext(context.Background(), txn)

	query := `{ a_hello b_errors }`
	params := graphql.Params{
		Schema:        schema,
		RequestString: query,
		Context:       ctx,
	}
	resp := graphql.Do(params)
	if len(resp.Errors) != 1 {
		t.Error("incorrect number of errors on response", resp.Errors)
	}
	js, err := json.Marshal(resp.Data)
	if err != nil {
		t.Error("failure to marshal json:", err)
	}
	if data := string(js); data != `{"a_hello":"world","b_errors":null}` {
		t.Error("incorrect response data:", data)
	}

	txn.End()
	// if Resolve becomes asynchronous, an error will be logged saying that we
	// tried to end a segment twice.
	app.ExpectNoLoggedErrors(t)
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "Custom/Execution", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Execution", Scope: "OtherTransaction/Go/query", Forced: false, Data: nil},
		{Name: "Custom/Parse", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Parse", Scope: "OtherTransaction/Go/query", Forced: false, Data: nil},
		{Name: "Custom/Resolve a_hello", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Resolve a_hello", Scope: "OtherTransaction/Go/query", Forced: false, Data: nil},
		{Name: "Custom/Resolve b_errors", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Resolve b_errors", Scope: "OtherTransaction/Go/query", Forced: false, Data: nil},
		{Name: "Custom/Validation", Scope: "", Forced: false, Data: nil},
		{Name: "Custom/Validation", Scope: "OtherTransaction/Go/query", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/query", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/query", Scope: "", Forced: false, Data: nil},
	})
}
