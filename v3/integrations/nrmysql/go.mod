module github.com/newrelic/go-agent/v3/integrations/nrmysql

// 1.10 is the Go version in mysql's go.mod
go 1.17

require (
	// v1.5.0 is the first mysql version to support gomod
	github.com/go-sql-driver/mysql v1.6.0
	// v3.3.0 includes the new location of ParseQuery
	github.com/newrelic/go-agent/v3 v3.18.2
)
