module github.com/newrelic/go-agent/v3/integrations/nrconnect/example

go 1.25.0

require (
	connectrpc.com/connect v1.16.2
	github.com/newrelic/go-agent/v3 v3.44.1
	github.com/newrelic/go-agent/v3/integrations/nrconnect v0.0.0
	golang.org/x/net v0.55.0
	google.golang.org/protobuf v1.36.11
)

require (
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
	google.golang.org/grpc v1.80.0 // indirect
)

replace github.com/newrelic/go-agent/v3/integrations/nrconnect => ..

replace github.com/newrelic/go-agent/v3 => ../../..
