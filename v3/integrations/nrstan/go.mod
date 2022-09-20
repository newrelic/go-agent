module github.com/newrelic/go-agent/v3/integrations/nrstan

// As of Dec 2019, 1.11 is the earliest Go version tested by Stan:
// https://github.com/nats-io/stan.go/blob/master/.travis.yml
go 1.17

require (
	github.com/nats-io/stan.go v0.10.3
	github.com/newrelic/go-agent/v3 v3.18.2
)
