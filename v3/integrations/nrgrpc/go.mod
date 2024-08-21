module github.com/newrelic/go-agent/v3/integrations/nrgrpc

go 1.21

toolchain go1.22.6

require (
	github.com/newrelic/go-agent/v3 v3.33.1
	github.com/newrelic/go-agent/v3/integrations/nrsecurityagent v1.1.0
	// v1.15.0 is the earliest version of grpc using modules.
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
)

require github.com/golang/protobuf v1.5.4

replace github.com/newrelic/go-agent/v3/integrations/nrsecurityagent => ../../integrations/nrsecurityagent

replace github.com/newrelic/go-agent/v3 => ../..
