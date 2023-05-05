module github.com/newrelic/go-agent/v3/integrations/nrpgx

// As of Dec 2019, go 1.11 is the earliest version of Go tested by lib/pq:
// https://github.com/lib/pq/blob/master/.travis.yml
go 1.11

require (
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jackc/pgx/v4 v4.13.0
	github.com/newrelic/go-agent/v3 v3.3.0
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/text v0.3.8 // indirect
)
