module github.com/newrelic/go-agent/v4/integrations/nrstan

// As of Dec 2019, 1.11 is the earliest Go version tested by Stan:
// https://github.com/nats-io/stan.go/blob/master/.travis.yml
go 1.11

require (
	github.com/nats-io/stan.go v0.5.0
	github.com/newrelic/go-agent/v4 v4.0.0
)
