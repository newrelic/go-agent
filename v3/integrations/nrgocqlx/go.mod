module github.com/newrelic/go-agent/v3/integrations/nrgocqlx

go 1.25


replace github.com/gocql/gocql => github.com/scylladb/gocql v1.16.0

require (
	github.com/gocql/gocql v1.7.0
	github.com/newrelic/go-agent/v3 v3.44.1
	github.com/scylladb/gocqlx/v3 v3.0.4
)


replace github.com/newrelic/go-agent/v3 => ../..
