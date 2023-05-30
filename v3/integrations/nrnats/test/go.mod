module github.com/newrelic/go-agent/v3/integrations/test

// This module exists to avoid having extra nrnats module dependencies.

go 1.17

replace github.com/newrelic/go-agent/v3/integrations/nrnats v1.0.0 => ../

replace github.com/newrelic/go-agent/v3 v3.18.2 => ../../../

require (
	github.com/nats-io/nats-server v1.4.1
	github.com/nats-io/nats.go v1.17.0
	github.com/newrelic/go-agent/v3 v3.18.2
	github.com/newrelic/go-agent/v3/integrations/nrnats v1.0.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/nats-io/gnatsd v1.4.1 // indirect
	github.com/nats-io/go-nats v1.7.2 // indirect
	github.com/nats-io/nats-server/v2 v2.9.0 // indirect
	github.com/nats-io/nkeys v0.3.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.0.0-20220919173607-35f4265a4bc0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/grpc v1.54.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
