module github.com/newrelic/go-agent/v3/integrations/nrgraphqlgo/example

go 1.24

require (
	github.com/graphql-go/graphql v0.8.1
	github.com/graphql-go/graphql-go-handler v0.2.3
	github.com/newrelic/go-agent/v3 v3.41.0
	github.com/newrelic/go-agent/v3/integrations/nrgraphqlgo v1.0.0
)

replace github.com/newrelic/go-agent/v3/integrations/nrgraphqlgo => ../

replace github.com/newrelic/go-agent/v3 => ../../..
