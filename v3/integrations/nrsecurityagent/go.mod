module github.com/newrelic/go-agent/v3/integrations/nrsecurityagent

go 1.19

require (
	github.com/newrelic/csec-go-agent v0.2.1
	github.com/newrelic/go-agent/v3 v3.23.0
	github.com/newrelic/go-agent/v3/integrations/nrsqlite3 v1.1.1
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/newrelic/go-agent/v3 => ../..

require (
	github.com/dlclark/regexp2 v1.9.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/juju/fslock v0.0.0-20160525022230-4d5c94c67b4b // indirect
	github.com/k2io/hookingo v1.0.3 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mackerelio/go-osstat v0.2.4 // indirect
	github.com/mattn/go-sqlite3 v1.0.0 // indirect
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/struCoder/pidusage v0.2.1 // indirect
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/grpc v1.54.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
