module github.com/newrelic/go-agent/v3/integrations/nrgrpc

go 1.19

require (
	// protobuf v1.3.0 is the earliest version using modules, we use v1.3.1
	// because all dependencies were removed in this version.
	github.com/golang/protobuf v1.5.3
	github.com/newrelic/go-agent/v3 v3.30.0
	github.com/newrelic/go-agent/v3/integrations/nrsecurityagent v1.1.0
	// v1.15.0 is the earliest version of grpc using modules.
	google.golang.org/grpc v1.56.3
	google.golang.org/protobuf v1.33.0
)

require (
	github.com/dlclark/regexp2 v1.9.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/k2io/hookingo v1.0.5 // indirect
	github.com/newrelic/csec-go-agent v1.0.0 // indirect
	golang.org/x/arch v0.4.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/newrelic/go-agent/v3/integrations/nrsecurityagent => ../../integrations/nrsecurityagent

replace github.com/newrelic/go-agent/v3 => ../..
