module server-example

go 1.21

require (
	github.com/newrelic/go-agent/v3 v3.35.0
	github.com/newrelic/go-agent/v3/integrations/nrfasthttp v1.0.0
	github.com/valyala/fasthttp v1.49.0
)

replace github.com/newrelic/go-agent/v3/integrations/nrfasthttp v1.0.0 => ../../

replace github.com/newrelic/go-agent/v3 => ../../../..
