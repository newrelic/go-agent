module github.com/newrelic/go-agent/v3/integrations/nrgocql/example/gocqlx

go 1.25.0

replace github.com/newrelic/go-agent/v3/integrations/nrgocql => ../../

replace github.com/newrelic/go-agent/v3 => ../../../..

replace github.com/gocql/gocql => github.com/scylladb/gocql v1.16.0

require (
	github.com/gocql/gocql v1.7.0
	github.com/newrelic/go-agent/v3 v3.0.0-00010101000000-000000000000
	github.com/newrelic/go-agent/v3/integrations/nrgocql v0.0.0-00010101000000-000000000000
	github.com/scylladb/gocqlx/v3 v3.0.4
)