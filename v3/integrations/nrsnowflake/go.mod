module github.com/newrelic/go-agent/v3/integrations/nrsnowflake

// snowflakedb/gosnowflake says it requires 1.12 but builds on 1.10
go 1.10

require (
	// v3.3.0 includes the new location of ParseQuery
	github.com/newrelic/go-agent/v3 v3.3.0
	github.com/snowflakedb/gosnowflake v1.3.4
)
