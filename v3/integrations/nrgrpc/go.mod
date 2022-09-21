module github.com/newrelic/go-agent/v3/integrations/nrgrpc

// As of Dec 2019, the grpc go.mod file uses 1.11:
// https://github.com/grpc/grpc-go/blob/master/go.mod
go 1.17

require (
	// protobuf v1.3.0 is the earliest version using modules, we use v1.3.1
	// because all dependencies were removed in this version.
	github.com/golang/protobuf v1.5.2
	github.com/newrelic/go-agent/v3 v3.18.2
	// v1.15.0 is the earliest version of grpc using modules.
	google.golang.org/grpc v1.49.0
)

require (
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/sys v0.0.0-20210119212857-b64e53b001e4 // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)
