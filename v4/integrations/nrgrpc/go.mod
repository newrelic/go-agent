module github.com/newrelic/go-agent/v4/integrations/nrgrpc

// As of Dec 2019, the grpc go.mod file uses 1.11:
// https://github.com/grpc/grpc-go/blob/master/go.mod
go 1.11

replace github.com/newrelic/go-agent/v4 => ../../

require (
	// protobuf v1.3.0 is the earliest version using modules, we use v1.3.1
	// because all dependencies were removed in this version.
	github.com/golang/protobuf v1.3.3
	github.com/newrelic/go-agent/v4 v4.0.0
	go.opentelemetry.io/otel v0.16.0
	// v1.15.0 is the earliest version of grpc using modules.
	google.golang.org/grpc v1.30.0
)
