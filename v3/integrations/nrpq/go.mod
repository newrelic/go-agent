module github.com/newrelic/go-agent/v3/integrations/nrpq

go 1.24

require (
	// NewConnector dsn parsing tests expect v1.1.0 error return behavior.
	github.com/lib/pq v1.1.0
	// v3.3.0 includes the new location of ParseQuery
	github.com/newrelic/go-agent/v3 v3.42.0
)


replace github.com/newrelic/go-agent/v3 => ../..
