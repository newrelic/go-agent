module github.com/newrelic/go-agent/v3/integrations/logcontext-v2/logWriter

go 1.22

require (
	github.com/newrelic/go-agent/v3 v3.39.0
	github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrwriter v1.0.0
)

replace github.com/newrelic/go-agent/v3 => ../../..

replace github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrwriter => ../nrwriter
