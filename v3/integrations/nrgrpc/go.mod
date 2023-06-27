module github.com/newrelic/go-agent/v3/integrations/nrgrpc
go 1.19
require (
	// protobuf v1.3.0 is the earliest version using modules, we use v1.3.1
	// because all dependencies were removed in this version.
	github.com/golang/protobuf v1.5.3
	github.com/newrelic/go-agent/v3 v3.23.0
	// v1.15.0 is the earliest version of grpc using modules.
	google.golang.org/grpc v1.54.0
)
replace github.com/newrelic/go-agent/v3 => ../..
