module github.com/newrelic/go-agent/v3/integrations/nrmssql

go 1.22

require (
	github.com/microsoft/go-mssqldb v0.19.0
	github.com/newrelic/go-agent/v3 v3.40.0
)


replace github.com/newrelic/go-agent/v3 => ../..
