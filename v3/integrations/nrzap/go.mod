module github.com/newrelic/go-agent/v3/integrations/nzap

go 1.13

require (
	github.com/newrelic/go-agent/v3 v3.0.0
	// v1.12.0 is the earliest version of zap using modules.
	go.uber.org/zap v1.12.0
)
