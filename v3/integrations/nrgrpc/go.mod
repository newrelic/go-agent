module github.com/newrelic/go-agent/v3/integrations/nrgrpc

go 1.25

require (
	// protobuf v1.3.0 is the earliest version using modules, we use v1.3.1
	// because all dependencies were removed in this version.
	github.com/golang/protobuf v1.5.4
	// v1.15.0 is the earliest version of grpc using modules.
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/newrelic/go-agent/v3 v3.43.0
	github.com/newrelic/go-agent/v3/integrations/nrsecurityagent v0.0.0-00010101000000-000000000000
)

require (
	github.com/adhocore/gronx v1.19.1 // indirect
	github.com/dlclark/regexp2 v1.9.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/k2io/hookingo v1.0.6 // indirect
	github.com/newrelic/csec-go-agent v1.6.0 // indirect
	golang.org/x/arch v0.4.0 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/newrelic/go-agent/v3/integrations/nrsecurityagent => ../../integrations/nrsecurityagent

replace github.com/newrelic/go-agent/v3 => ../..
