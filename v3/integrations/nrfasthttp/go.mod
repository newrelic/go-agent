module github.com/newrelic/go-agent/v3/integrations/nrfasthttp

go 1.24

require (
	github.com/newrelic/go-agent/v3 v3.41.0
	github.com/valyala/fasthttp v1.49.0
)

replace github.com/newrelic/go-agent/v3 => ../..
