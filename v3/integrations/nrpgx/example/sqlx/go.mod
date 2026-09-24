// This sqlx example is a separate module to avoid adding sqlx dependency to the
// nrpgx go.mod file.
module github.com/newrelic/go-agent/v3/integrations/nrpgx/example/sqlx

go 1.25.0

require (
	github.com/jmoiron/sqlx v1.2.0
	github.com/newrelic/go-agent/v3 v3.45.0
	github.com/newrelic/go-agent/v3/integrations/nrpgx v0.0.0
)

require (
	github.com/google/pprof v0.0.0-20260802141513-ef3492d7dac3 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.14.3 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.3 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgtype v1.14.0 // indirect
	github.com/jackc/pgx v3.6.2+incompatible // indirect
	github.com/jackc/pgx/v4 v4.18.2 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/grpc v1.83.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/newrelic/go-agent/v3/integrations/nrpgx => ../../

replace github.com/newrelic/go-agent/v3 => ../../../..
