module github.com/newrelic/go-agent/v3/integrations/nrmysql

// As of Dec 2019, 1.9 is the Go version in mysql's go.mod:
// https://github.com/go-sql-driver/mysql/blob/master/go.mod
go 1.9

require (
	// As of Nov 2019, the latest go-sql-driver/mysql release (v1.4.1) does
	// not support modules, though there is an unreleased go.mod on master.
	// v1.3.0 is required for ParseDSN.
	github.com/go-sql-driver/mysql v1.3.0
	github.com/newrelic/go-agent/v3 v3.0.0
)
