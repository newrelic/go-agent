module github.com/newrelic/go-agent/v3/integrations/nrgraphgophers

// As of Jan 2020, the graphql-go go.mod file uses 1.13:
// https://github.com/graph-gophers/graphql-go/blob/master/go.mod
go 1.21

require (
	// graphql-go has no tagged releases as of Jan 2020.
	github.com/graph-gophers/graphql-go v1.3.0
	github.com/newrelic/go-agent/v3 v3.36.0
)


replace github.com/newrelic/go-agent/v3 => ../..
