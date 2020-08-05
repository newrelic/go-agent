module github.com/newrelic/go-agent/v4/integrations/nrpq

// As of Dec 2019, go 1.11 is the earliest version of Go tested by lib/pq:
// https://github.com/lib/pq/blob/master/.travis.yml
go 1.11

require (
	// NewConnector dsn parsing tests expect v1.1.0 error return behavior.
	github.com/lib/pq v1.1.0
	github.com/newrelic/go-agent/v4 v4.0.0
)
