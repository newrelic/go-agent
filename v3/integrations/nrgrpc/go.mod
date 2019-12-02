module github.com/newrelic/go-agent/v3/integrations/nrgrpc

// As of Dec 2019, the grpc go.mod file uses 1.11:
// https://github.com/grpc/grpc-go/blob/master/go.mod
go 1.11

require (
	// protobuf v1.3.0 is the earliest version using modules, also the
	// protoc-gen-go output has no dependencies.
	github.com/golang/protobuf v1.3.0
	github.com/newrelic/go-agent/v3 v3.0.0
	// v1.15.0 is the earliest version of grpc using modules.
	google.golang.org/grpc v1.15.0
)
