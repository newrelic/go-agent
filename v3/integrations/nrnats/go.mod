module github.com/newrelic/go-agent/v3/integrations/nrnats

// As of Jun 2023, 1.19 is the earliest version of Go tested by nats:
// https://github.com/nats-io/nats.go/blob/master/.travis.yml
go 1.25.0

require (
	github.com/nats-io/nats-server/v2 v2.11.15
	github.com/nats-io/nats.go v1.49.0
	github.com/newrelic/go-agent/v3 v3.45.0
)

require (
	github.com/google/pprof v0.0.0-20260802141513-ef3492d7dac3 // indirect
	github.com/klauspost/compress v1.18.4 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/grpc v1.83.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/newrelic/go-agent/v3 => ../..
