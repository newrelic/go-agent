// This sqlx example is a separate module to avoid adding sqlx dependency to the
// nrpgx go.mod file.
module github.com/newrelic/go-agent/v3/integrations/nrpgx/example/sqlx
go 1.25
require (
	github.com/jmoiron/sqlx v1.2.0
	github.com/newrelic/go-agent/v3 v3.42.0
	github.com/newrelic/go-agent/v3/integrations/nrpgx v0.0.0
)
replace github.com/newrelic/go-agent/v3/integrations/nrpgx => ../../
replace github.com/newrelic/go-agent/v3 => ../../../..
