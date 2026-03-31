module github.com/newrelic/go-agent/v3/integrations/nrgocql

go 1.25.0

replace github.com/newrelic/go-agent/v3 => ../..

replace github.com/gocql/gocql => github.com/scylladb/gocql v1.16.0

require (
	github.com/apache/cassandra-gocql-driver/v2 v2.0.0
	github.com/newrelic/go-agent/v3 v3.0.0-00010101000000-000000000000
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/grpc v1.65.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
)
