module github.com/newrelic/go-agent/v3/integrations/nrnats

// As of Jun 2023, 1.19 is the earliest version of Go tested by nats:
// https://github.com/nats-io/nats.go/blob/master/.travis.yml
go 1.19

require (
	github.com/nats-io/nats-server v1.4.1
	github.com/nats-io/nats.go v1.28.0
	github.com/newrelic/go-agent/v3 v3.26.0
)


replace github.com/newrelic/go-agent/v3 => ../..
