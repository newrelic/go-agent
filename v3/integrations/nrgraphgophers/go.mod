module github.com/newrelic/go-agent/v3/integrations/nrgraphgophers

// As of Jan 2020, the graphql-go go.mod file uses 1.13:
// https://github.com/graph-gophers/graphql-go/blob/master/go.mod
go 1.25.0

require (
	// graphql-go has no tagged releases as of Jan 2020.
	github.com/graph-gophers/graphql-go v1.3.0
	github.com/newrelic/go-agent/v3 v3.45.0
)

require (
	github.com/google/pprof v0.0.0-20260802141513-ef3492d7dac3 // indirect
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/grpc v1.83.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/newrelic/go-agent/v3 => ../..
