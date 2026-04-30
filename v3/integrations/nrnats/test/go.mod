module github.com/newrelic/go-agent/v3/integrations/test

// This module exists to avoid having extra nrnats module dependencies.
go 1.25

replace github.com/newrelic/go-agent/v3/integrations/nrnats v1.0.0 => ../

replace github.com/newrelic/go-agent/v3 => ../../..
