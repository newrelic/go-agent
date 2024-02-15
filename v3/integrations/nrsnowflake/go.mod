module github.com/newrelic/go-agent/v3/integrations/nrsnowflake

go 1.19

require (
	github.com/newrelic/go-agent/v3 v3.30.0
	github.com/snowflakedb/gosnowflake v1.6.19
)


replace github.com/newrelic/go-agent/v3 => ../..
