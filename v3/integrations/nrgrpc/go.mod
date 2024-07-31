module github.com/newrelic/go-agent/v3/integrations/nrgrpc

go 1.20

require (
	// protobuf v1.3.0 is the earliest version using modules, we use v1.3.1
	// because all dependencies were removed in this version.
	github.com/golang/protobuf v1.5.3
	github.com/newrelic/go-agent/v3 v3.33.1
	github.com/newrelic/go-agent/v3/integrations/nrsecurityagent v1.1.0
	// v1.15.0 is the earliest version of grpc using modules.
	google.golang.org/grpc v1.56.3
	google.golang.org/protobuf v1.33.0
)


replace github.com/newrelic/go-agent/v3/integrations/nrsecurityagent => ../../integrations/nrsecurityagent

replace github.com/newrelic/go-agent/v3 => ../..
