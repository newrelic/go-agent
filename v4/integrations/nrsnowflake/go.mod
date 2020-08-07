module github.com/newrelic/go-agent/v4/integrations/nrsnowflake

// snowflakedb/gosnowflake says it requires 1.12 but builds on 1.10
go 1.10

require (
	github.com/newrelic/go-agent/v4 v4.0.0
	github.com/snowflakedb/gosnowflake v1.3.4
)
