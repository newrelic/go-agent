module github.com/newrelic/go-agent/v3/integrations/nrstan/examples

// This module exists to avoid a dependency on nrnrats.

go 1.13

require (
	github.com/nats-io/stan.go v0.5.0
	github.com/newrelic/go-agent/v3 v3.4.0
	github.com/newrelic/go-agent/v3/integrations/nrnats v0.0.0
	github.com/newrelic/go-agent/v3/integrations/nrstan v0.0.0
)

replace github.com/newrelic/go-agent/v3 => ../../../

replace github.com/newrelic/go-agent/v3/integrations/nrstan => ../

replace github.com/newrelic/go-agent/v3/integrations/nrnats => ../../nrnats/
