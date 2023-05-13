module github.com/newrelic/go-agent/v3/integrations/nrgrpc

go 1.18

require (
	// protobuf v1.3.0 is the earliest version using modules, we use v1.3.1
	// because all dependencies were removed in this version.
	github.com/golang/protobuf v1.5.3
	github.com/newrelic/go-agent/v3 v3.22.0
	// v1.15.0 is the earliest version of grpc using modules.
	google.golang.org/grpc v1.54.0
)

require (
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)

replace github.com/newrelic/go-agent/v3 v3.22.0 => ../..
