module github.com/newrelic/go-agent/v3/integrations/nrnats

// As of Jun 2023, 1.19 is the earliest version of Go tested by nats:
// https://github.com/nats-io/nats.go/blob/master/.travis.yml
go 1.25

require (
	github.com/nats-io/nats-server/v2 v2.10.22
	github.com/nats-io/nats.go v1.36.0
	github.com/newrelic/go-agent/v3 v3.43.2
)


replace github.com/newrelic/go-agent/v3 => ../..
