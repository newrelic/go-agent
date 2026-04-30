module github.com/newrelic/go-agent/v3/integrations/nrpgx5

go 1.25

require (
	github.com/jackc/pgx/v5 v5.9.2
	github.com/newrelic/go-agent/v3 v3.43.3
	github.com/stretchr/testify v1.11.1
)


replace github.com/newrelic/go-agent/v3 => ../..
