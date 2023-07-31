module github.com/newrelic/go-agent/v3/integrations/nrpgx

// As of Dec 2019, go 1.11 is the earliest version of Go tested by lib/pq:
// https://github.com/lib/pq/blob/master/.travis.yml
go 1.18

require (
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jackc/pgx/v4 v4.13.0
	github.com/newrelic/go-agent/v3 v3.3.0
)

require (
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgconn v1.10.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.1.1 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/pgtype v1.8.1 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97 // indirect
	golang.org/x/text v0.3.6 // indirect
)
