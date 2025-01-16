module github.com/newrelic/go-agent/v3/integrations/nrsnowflake

go 1.21

require (
	github.com/newrelic/go-agent/v3 v3.36.0
	github.com/snowflakedb/gosnowflake v1.6.19
)


replace github.com/newrelic/go-agent/v3 => ../..
