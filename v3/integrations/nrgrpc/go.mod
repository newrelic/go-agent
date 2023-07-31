module github.com/newrelic/go-agent/v3/integrations/nrgrpc

go 1.19

require (
	// protobuf v1.3.0 is the earliest version using modules, we use v1.3.1
	// because all dependencies were removed in this version.
	github.com/golang/protobuf v1.5.3
	github.com/newrelic/go-agent/v3 v3.23.0
	github.com/newrelic/go-agent/v3/integrations/nrsecurityagent v1.0.2
	// v1.15.0 is the earliest version of grpc using modules.
	google.golang.org/grpc v1.54.0
	google.golang.org/protobuf v1.28.1
)

require (
	github.com/dlclark/regexp2 v1.9.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/juju/fslock v0.0.0-20160525022230-4d5c94c67b4b // indirect
	github.com/k2io/hookingo v1.0.3 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mackerelio/go-osstat v0.2.4 // indirect
	github.com/newrelic/csec-go-agent v0.2.1 // indirect
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/struCoder/pidusage v0.2.1 // indirect
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/newrelic/go-agent/v3 => ../..
