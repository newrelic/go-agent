// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/graphql-go/graphql"
	handler "github.com/graphql-go/graphql-go-handler"

	"github.com/newrelic/go-agent/v3/integrations/nrgraphqlgo"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var schema = func() graphql.Schema {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"latestPost": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						time.Sleep(time.Second)
						return "Hello World!", nil
					},
				},
				"randomNumber": &graphql.Field{
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						time.Sleep(time.Second)
						return 5, nil
					},
				},
				"erroring": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						time.Sleep(time.Second)
						return nil, errors.New("oooops")
					},
				},
			},
		}),
		// 1. Add the nrgraphqlgo.Extension to the schema
		Extensions: []graphql.Extension{nrgraphqlgo.Extension{}},
	})
	if err != nil {
		panic(err)
	}
	return schema
}()

func main() {
	// 2. Create the New Relic application
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Example GraphQL App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	h := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})

	// 3. Make sure to instrument your HTTP handler, which will
	// create/end transactions, record error codes, and add
	// the transactions to the context.
	http.Handle(newrelic.WrapHandle(app, "/graphql", h))

	// You can test your example query with curl:
	//   curl -X POST \
	//   -H "Content-Type: application/json" \
	//   -d '{"query": "{latestPost, randomNumber}"}' \
	//   localhost:8080/graphql
	http.ListenAndServe(":8080", nil)
}
