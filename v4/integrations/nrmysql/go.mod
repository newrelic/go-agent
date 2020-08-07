module github.com/newrelic/go-agent/v4/integrations/nrmysql

// 1.10 is the Go version in mysql's go.mod
go 1.10

require (
	// v1.5.0 is the first mysql version to support gomod
	github.com/go-sql-driver/mysql v1.5.0
	github.com/newrelic/go-agent/v4 v4.0.0
)
