module github.com/newrelic/go-agent/v3/integrations/nrgocqlx

go 1.25.0

replace github.com/gocql/gocql => github.com/scylladb/gocql v1.16.0

require (
	github.com/gocql/gocql v1.7.0
	github.com/newrelic/go-agent/v3 v3.45.0
	github.com/scylladb/gocqlx/v3 v3.0.4
)

require (
	github.com/google/pprof v0.0.0-20260802141513-ef3492d7dac3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.1 // indirect
	github.com/scylladb/go-reflectx v1.0.1 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/grpc v1.83.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
)

replace github.com/newrelic/go-agent/v3 => ../..
