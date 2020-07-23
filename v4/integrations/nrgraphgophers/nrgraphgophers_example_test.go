// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrgraphgophers_test

import (
	"log"
	"net/http"
	"os"

	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/newrelic/go-agent/v3/integrations/nrgraphgophers"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type query struct{}

func (*query) Hello() string { return "hello world" }

func Example() {
	// First create your New Relic Application:
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("GraphQL App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if nil != err {
		panic(err)
	}

	querySchema := `type Query { hello: String! }`

	// Then add a graphql.Tracer(nrgraphgophers.NewTracer()) option to your
	// schema parsing to get field and query segment instrumentation:
	opt := graphql.Tracer(nrgraphgophers.NewTracer())
	schema := graphql.MustParseSchema(querySchema, &query{}, opt)

	// Finally, instrument your request handler using newrelic.WrapHandle
	// to create transactions for requests:
	http.Handle(newrelic.WrapHandle(app, "/", &relay.Handler{Schema: schema}))
	log.Fatal(http.ListenAndServe(":8000", nil))
}
