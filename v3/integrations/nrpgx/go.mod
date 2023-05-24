module github.com/newrelic/go-agent/v3/integrations/nrpgx

// As of Dec 2019, go 1.11 is the earliest version of Go tested by lib/pq:
// https://github.com/lib/pq/blob/master/.travis.yml
go 1.18

require (
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jackc/pgx/v4 v4.13.0
	github.com/newrelic/go-agent/v3 v3.3.0
)
