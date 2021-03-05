// This sqlx example is a separate module to avoid adding sqlx dependency to the
// nrpq go.mod file.

module github.com/newrelic/go-agent/v3/integrations/nrcloudsqlpostgres/example/sqlx

go 1.13

require (
	github.com/jmoiron/sqlx v1.2.0
	github.com/GoogleCloudPlatform/cloudsql-proxy v1.19.2
	github.com/newrelic/go-agent/v3 v3.3.0
	github.com/newrelic/go-agent/v3/integrations/nrcloudsqlpostgres v0.0.0
)

replace github.com/newrelic/go-agent/v3 => ../../../../

replace github.com/newrelic/go-agent/v3/integrations/nrcloudsqlpostgres => ../../
