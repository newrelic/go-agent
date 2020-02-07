package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/graphql-go/graphql"
	nrgraphql "github.com/newrelic/go-agent/v3/integrations/nr-graphql-go"
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
		// 1. Add the nrgraphql.Extension to the schema
		Extensions: []graphql.Extension{nrgraphql.Extension{}},
	})
	if err != nil {
		panic(err)
	}
	return schema
}()

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Example GraphQL App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}
	app.WaitForConnection(10 * time.Second)
	defer app.Shutdown(10 * time.Second)

	// 2. Ensure the graphql code is called from within a running transaction
	txn := app.StartTransaction("main")
	defer txn.End()

	// 3. Add the transaction to the context
	ctx := newrelic.NewContext(context.Background(), txn)

	query := `query{
		latestPost,
		randomNumber,
		erroring,
	}`
	res := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
		// 4. Add the context to the calling params
		Context: ctx,
	})

	for _, e := range res.Errors {
		fmt.Println("graphql result contained error:", e)
	}
}
