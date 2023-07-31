module github.com/newrelic/go-agent/v3/integrations/nrstan

// As of Dec 2019, 1.11 is the earliest Go version tested by Stan:
// https://github.com/nats-io/stan.go/blob/master/.travis.yml
go 1.18

require (
	github.com/nats-io/stan.go v0.10.4
	github.com/newrelic/go-agent/v3 v3.21.1
)

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/klauspost/compress v1.16.5 // indirect
	github.com/nats-io/nats-server/v2 v2.9.20 // indirect
	github.com/nats-io/nats-streaming-server v0.25.5 // indirect
	github.com/nats-io/nats.go v1.27.0 // indirect
	github.com/nats-io/nkeys v0.4.4 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.10.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sys v0.9.0 // indirect
	golang.org/x/text v0.10.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/grpc v1.54.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
