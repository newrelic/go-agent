module github.com/newrelic/go-agent/v3/integrations/nrgrpc

// As of Dec 2019, the grpc go.mod file uses 1.11:
// https://github.com/grpc/grpc-go/blob/master/go.mod
go 1.11

require (
	github.com/golang/lint v0.0.0-20180702182130-06c8688daad7 // indirect
	// protobuf v1.3.0 is the earliest version using modules, we use v1.3.1
	// because all dependencies were removed in this version.
	github.com/golang/protobuf v1.3.3
	github.com/kisielk/gotool v1.0.0 // indirect
	github.com/newrelic/go-agent/v3 v3.0.0
	// v1.15.0 is the earliest version of grpc using modules.
	google.golang.org/grpc v1.27.0
)
