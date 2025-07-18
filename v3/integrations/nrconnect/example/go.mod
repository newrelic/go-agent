module github.com/newrelic/go-agent/v3/integrations/nrconnect/example

go 1.22

require (
	connectrpc.com/connect v1.16.2
	github.com/newrelic/go-agent/v3 v3.40.1
	github.com/newrelic/go-agent/v3/integrations/nrconnect v0.0.0
	golang.org/x/net v0.25.0
	google.golang.org/protobuf v1.34.2
)


replace github.com/newrelic/go-agent/v3/integrations/nrconnect => ..

replace github.com/newrelic/go-agent/v3 => ../../..
