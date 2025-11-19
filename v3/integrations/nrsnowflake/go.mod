module github.com/newrelic/go-agent/v3/integrations/nrsnowflake

go 1.24

toolchain go1.23.2

require (
	github.com/newrelic/go-agent/v3 v3.42.0
	github.com/snowflakedb/gosnowflake v1.14.0
)

replace github.com/newrelic/go-agent/v3 => ../..
