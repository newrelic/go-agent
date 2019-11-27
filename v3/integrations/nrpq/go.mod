module github.com/newrelic/go-agent/v3/integrations/nrpq

go 1.13

require (
	// NewConnector dsn parsing tests expect v1.1.0 error return behavior.
	github.com/lib/pq v1.1.0
	github.com/newrelic/go-agent/v3 v3.0.0
)
