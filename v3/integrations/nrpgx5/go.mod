module github.com/newrelic/go-agent/v3/integrations/nrpgx5

go 1.22

toolchain go1.24.2

require (
	github.com/egon12/pgsnap v0.0.0-20221022154027-2847f0124ed8
	github.com/jackc/pgx/v5 v5.5.4
	github.com/newrelic/go-agent/v3 v3.41.0
	github.com/stretchr/testify v1.8.1
)


replace github.com/newrelic/go-agent/v3 => ../..
