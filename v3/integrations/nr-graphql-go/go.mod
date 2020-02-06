module github.com/newrelic/go-agent/v3/integrations/nr-graphql-go

go 1.13

require (
	// The nrgraphql.Extension requires a commit that is on the graphql-go
	// master branch but not yet part of an official release, as of Jan 2020.
	github.com/graphql-go/graphql v0.7.9-0.20200205010002-d92501231054
	github.com/newrelic/go-agent/v3 v3.3.0
)
