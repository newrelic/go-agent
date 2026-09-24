module github.com/newrelic/go-agent/v3/integrations/nrlogrus

// As of Dec 2019, the logrus go.mod file uses 1.13:
// https://github.com/sirupsen/logrus/blob/master/go.mod
go 1.25.0

require (
	github.com/newrelic/go-agent/v3 v3.45.0
	github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrlogrus v1.0.0
	// v1.1.0 is required for the Logger.GetLevel method, and is the earliest
	// version of logrus using modules.
	github.com/sirupsen/logrus v1.8.3
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

replace github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrlogrus => ../logcontext-v2/nrlogrus

replace github.com/newrelic/go-agent/v3 => ../..
