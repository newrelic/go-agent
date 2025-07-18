module github.com/newrelic/go-agent/v3/integrations/nrconnect/example

go 1.23.0

require (
	connectrpc.com/connect v1.16.2
	github.com/newrelic/go-agent/v3 v3.40.1
	github.com/newrelic/go-agent/v3/integrations/nrconnect v0.0.0
	golang.org/x/net v0.38.0
	google.golang.org/protobuf v1.34.2
)

require (
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/grpc v1.65.0 // indirect
)

replace github.com/newrelic/go-agent/v3/integrations/nrconnect => ..

replace github.com/newrelic/go-agent/v3 => ../../..
