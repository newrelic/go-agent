module github.com/newrelic/go-agent/v3/integrations/nrnats
// As of Dec 2019, 1.11 is the earliest version of Go tested by nats:
// https://github.com/nats-io/nats.go/blob/master/.travis.yml
go 1.18
require (
	github.com/nats-io/nats-server v1.4.1
	github.com/nats-io/nats.go v1.25.0
	github.com/newrelic/go-agent/v3 v3.21.0
)
