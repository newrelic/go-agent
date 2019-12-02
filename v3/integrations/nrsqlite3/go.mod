module github.com/newrelic/go-agent/v3/integrations/nrsqlite3

// As of Dec 2019, 1.9 is the oldest version of Go tested by go-sqlite3:
// https://github.com/mattn/go-sqlite3/blob/master/.travis.yml
go 1.9

require (
	github.com/mattn/go-sqlite3 v1.0.0
	github.com/newrelic/go-agent/v3 v3.0.0
)
