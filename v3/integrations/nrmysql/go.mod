module github.com/newrelic/go-agent/v3/integrations/nrmysql

// As of Dec 2019, 1.10 is the Go version in mysql's go.mod:
// https://github.com/go-sql-driver/mysql/blob/master/go.mod
go 1.10

require (
	github.com/go-sql-driver/mysql v1.5.0
	github.com/newrelic/go-agent/v3 v3.4.0
)
