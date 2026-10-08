// This sqlx example is a separate module to avoid adding sqlx dependency to the
// nrpq go.mod file.
module github.com/newrelic/go-agent/v3/integrations/nrpq/example/sqlx

go 1.25.0

require (
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.1.0
	github.com/newrelic/go-agent/v3 v3.45.0
	github.com/newrelic/go-agent/v3/integrations/nrpq v0.0.0
)

require (
	github.com/google/pprof v0.0.0-20260802141513-ef3492d7dac3 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/grpc v1.83.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/newrelic/go-agent/v3/integrations/nrpq => ../../

replace github.com/newrelic/go-agent/v3 => ../../../..
