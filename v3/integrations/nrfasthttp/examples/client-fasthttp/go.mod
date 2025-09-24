module client-example

go 1.22

require (
	github.com/newrelic/go-agent/v3 v3.41.0
	github.com/newrelic/go-agent/v3/integrations/nrfasthttp v1.0.0
	github.com/valyala/fasthttp v1.49.0
)

replace github.com/newrelic/go-agent/v3/integrations/nrfasthttp v1.0.0 => ../../

replace github.com/newrelic/go-agent/v3 => ../../../..
