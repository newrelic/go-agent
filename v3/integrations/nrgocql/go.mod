module github.com/newrelic/go-agent/v3/integrations/nrgocql

go 1.25

replace github.com/newrelic/go-agent/v3 => ../..

require (
	github.com/apache/cassandra-gocql-driver/v2 v2.0.0
	github.com/gocql/gocql v1.7.0
	github.com/newrelic/go-agent/v3 v3.0.0-00010101000000-000000000000
)