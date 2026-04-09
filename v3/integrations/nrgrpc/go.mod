module github.com/newrelic/go-agent/v3/integrations/nrgrpc

go 1.25

require (
	// protobuf v1.3.0 is the earliest version using modules, we use v1.3.1
	// because all dependencies were removed in this version.
	github.com/golang/protobuf v1.5.4
	// v1.15.0 is the earliest version of grpc using modules.
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/newrelic/go-agent/v3 v3.43.1
	github.com/newrelic/go-agent/v3/integrations/nrsecurityagent v0.0.0-00010101000000-000000000000
)


replace github.com/newrelic/go-agent/v3/integrations/nrsecurityagent => ../../integrations/nrsecurityagent

replace github.com/newrelic/go-agent/v3 => ../..
