module github.com/newrelic/go-agent/v3/integrations/nrgorilla

// As of Dec 2019, the gorilla/mux go.mod file uses 1.12:
// https://github.com/gorilla/mux/blob/master/go.mod
go 1.25.0

require (
	// v1.7.0 is the earliest version of Gorilla using modules.
	github.com/gorilla/mux v1.7.0
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
