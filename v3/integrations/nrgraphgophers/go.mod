module github.com/newrelic/go-agent/v3/integrations/nrgraphgophers

// As of Jan 2020, the graphql-go go.mod file uses 1.13:
// https://github.com/graph-gophers/graphql-go/blob/master/go.mod
go 1.13

require (
	// graphql-go has no tagged releases as of Jan 2020.
	github.com/graph-gophers/graphql-go v0.0.0-20200207002730-8334863f2c8b
	github.com/newrelic/go-agent/v3 v3.17.0
)
