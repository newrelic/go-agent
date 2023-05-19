module github.com/newrelic/go-agent/v3/integrations/nrnats

// As of Dec 2019, 1.11 is the earliest version of Go tested by nats:
// https://github.com/nats-io/nats.go/blob/master/.travis.yml
go 1.18

require (
	github.com/nats-io/nats-server v1.4.1
	github.com/nats-io/nats.go v1.25.0
	github.com/newrelic/go-agent/v3 v3.21.0
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/nats-io/gnatsd v1.4.1 // indirect
	github.com/nats-io/go-nats v1.7.2 // indirect
	github.com/nats-io/nats-server/v2 v2.9.17 // indirect
	github.com/nats-io/nkeys v0.4.4 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.8.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/grpc v1.49.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)
