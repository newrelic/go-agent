module github.com/newrelic/go-agent/v3/integrations/test

// This module exists to avoid having extra nrnats module dependencies.

go 1.13

require (
	github.com/nats-io/gnatsd v1.4.1 // indirect
	github.com/nats-io/nats-server v1.4.1
	github.com/nats-io/nats.go v1.8.0
	github.com/newrelic/go-agent/v3 v3.4.0
	github.com/newrelic/go-agent/v3/integrations/nrnats v0.0.0
)

replace github.com/newrelic/go-agent/v3 => ../../../

replace github.com/newrelic/go-agent/v3/integrations/nrnats => ../
