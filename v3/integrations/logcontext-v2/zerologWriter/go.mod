module github.com/newrelic/go-agent/v3/integrations/logcontext-v2/zerologWriter

go 1.24

require (
	github.com/newrelic/go-agent/v3 v3.41.0
	github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrwriter v1.0.0
	github.com/rs/zerolog v1.27.0
)

replace github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrwriter => ../nrwriter

replace github.com/newrelic/go-agent/v3 => ../../..
