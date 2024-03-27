module github.com/newrelic/go-agent/v3/integrations/nrpgx

go 1.19

require (
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jackc/pgx/v4 v4.18.2
	github.com/newrelic/go-agent/v3 v3.30.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgconn v1.14.3 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.3 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgtype v1.14.0 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	golang.org/x/crypto v0.20.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.56.3 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace github.com/newrelic/go-agent/v3 => ../..
