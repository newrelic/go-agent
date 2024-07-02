module github.com/newrelic/go-agent/v3/integrations/nrfasthttp

go 1.20

require (
	github.com/newrelic/go-agent/v3 v3.33.1
	github.com/valyala/fasthttp v1.49.0
)


replace github.com/newrelic/go-agent/v3 => ../..
