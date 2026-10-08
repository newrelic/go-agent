module github.com/newrelic/go-agent/v3/integrations/nrsqlite3

// As of Dec 2019, 1.9 is the oldest version of Go tested by go-sqlite3:
// https://github.com/mattn/go-sqlite3/blob/master/.travis.yml
go 1.25.0

require (
	github.com/mattn/go-sqlite3 v1.0.0
	// v3.3.0 includes the new location of ParseQuery
	github.com/newrelic/go-agent/v3 v3.45.0
)

require (
	github.com/google/pprof v0.0.0-20260802141513-ef3492d7dac3 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/grpc v1.83.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/newrelic/go-agent/v3 => ../..
