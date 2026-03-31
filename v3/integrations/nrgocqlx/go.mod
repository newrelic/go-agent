module github.com/newrelic/go-agent/v3/integrations/nrgocqlx

go 1.25.0

replace github.com/newrelic/go-agent/v3 => ../..

replace github.com/gocql/gocql => github.com/scylladb/gocql v1.16.0

require (
	github.com/gocql/gocql v1.7.0
	github.com/newrelic/go-agent/v3 v3.0.0-00010101000000-000000000000
	github.com/scylladb/gocqlx/v3 v3.0.4
)

require github.com/scylladb/go-reflectx v1.0.1 // indirect

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.1 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/grpc v1.65.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
)
