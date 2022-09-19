module github.com/newrelic/go-agent/v3/integrations/nrnats

// As of Dec 2019, 1.11 is the earliest version of Go tested by nats:
// https://github.com/nats-io/nats.go/blob/master/.travis.yml
go 1.17

require (
	github.com/nats-io/nats.go v1.17.0
	github.com/newrelic/go-agent/v3 v3.18.2
)

require (
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/nats-io/nats-server/v2 v2.9.0 // indirect
	github.com/nats-io/nkeys v0.3.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.0.0-20220829220503-c86fa9a7ed90 // indirect
	golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2 // indirect
	golang.org/x/sys v0.0.0-20220906135438-9e1f76180b77 // indirect
	golang.org/x/text v0.3.6 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/grpc v1.39.0 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)
