module github.com/newrelic/go-agent/v3/integrations/nrconnect

go 1.24

require (
	connectrpc.com/connect v1.16.2
	github.com/newrelic/go-agent/v3 v3.41.0
	google.golang.org/protobuf v1.34.2
)

replace github.com/newrelic/go-agent/v3 => ../..
