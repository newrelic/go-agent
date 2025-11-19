module github.com/newrelic/go-agent/v3/integrations/logcontext-v2/logWriter

go 1.24

require (
	github.com/newrelic/go-agent/v3 v3.42.0
	github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrwriter v1.0.0
)


replace github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrwriter => ../nrwriter
replace github.com/newrelic/go-agent/v3 => ../../..
