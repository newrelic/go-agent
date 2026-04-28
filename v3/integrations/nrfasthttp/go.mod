module github.com/newrelic/go-agent/v3/integrations/nrfasthttp

go 1.25

require (
	github.com/newrelic/go-agent/v3 v3.43.3
	github.com/valyala/fasthttp v1.49.0
)


replace github.com/newrelic/go-agent/v3 => ../..
