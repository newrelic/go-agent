// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"
	"net/http"
	"os"

	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/newrelic/go-agent/v3/integrations/nrgraphgophers"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

type query struct{}

func (*query) Hello() string { return "Hello, world!" }

func main() {
	// First create your New Relic Application:
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("GraphQL App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if nil != err {
		panic(err)
	}

	s := `type Query { hello: String! }`

	// Then add a graphql.Tracer(nrgraphgophers.NewTracer()) option to your
	// schema parsing to get field and query segment instrumentation:
	opt := graphql.Tracer(nrgraphgophers.NewTracer())
	schema := graphql.MustParseSchema(s, &query{}, opt)

	// Finally, instrument your request handler using newrelic.WrapHandle
	// to create transactions for requests:
	http.Handle(newrelic.WrapHandle(app, "/graphql", &relay.Handler{Schema: schema}))

	// To test, run:
	// curl -X POST -d '{"query": "query HelloOperation { hello }" }' localhost:8000/graphql
	log.Fatal(http.ListenAndServe(":8000", nil))
}
