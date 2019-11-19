module github.com/newrelic/go-agent/v3/integrations/nrgrpc

go 1.13

require (
	github.com/golang/protobuf v1.2.0
	github.com/newrelic/go-agent/v3 v3.0.0
	// v1.15.0 is the earliest version of grpc using modules.
	google.golang.org/grpc v1.15.0
)
