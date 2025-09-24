module github.com/newrelic/go-agent/v3/integrations/test

// This module exists to avoid having extra nrnats module dependencies.
go 1.24

replace github.com/newrelic/go-agent/v3/integrations/nrnats v1.0.0 => ../

require (
	github.com/nats-io/nats-server v1.4.1
	github.com/nats-io/nats.go v1.36.0
	github.com/newrelic/go-agent/v3 v3.41.0
	github.com/newrelic/go-agent/v3/integrations/nrnats v1.0.0
)

replace github.com/newrelic/go-agent/v3 => ../../..
