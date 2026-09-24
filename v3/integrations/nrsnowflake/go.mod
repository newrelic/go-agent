module github.com/newrelic/go-agent/v3/integrations/nrsnowflake

go 1.25

require (
	github.com/newrelic/go-agent/v3 v3.45.0
	github.com/snowflakedb/gosnowflake v1.14.0
)


replace github.com/newrelic/go-agent/v3 => ../..
