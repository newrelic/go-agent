module github.com/newrelic/go-agent/v3/integrations/nrzap

// As of Dec 2019, zap has 1.13 in their go.mod file:
// https://github.com/uber-go/zap/blob/master/go.mod
go 1.19

require (
	github.com/newrelic/go-agent/v3 v3.28.1
	// v1.12.0 is the earliest version of zap using modules.
	go.uber.org/zap v1.12.0
)


replace github.com/newrelic/go-agent/v3 => ../..
