module github.com/newrelic/go-agent/v3/integrations/nrpgx

go 1.25

require (
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jackc/pgx/v4 v4.18.2
	github.com/newrelic/go-agent/v3 v3.42.0
)


replace github.com/newrelic/go-agent/v3 => ../..
