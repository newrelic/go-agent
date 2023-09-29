module github.com/newrelic/go-agent/v3/integrations/nrfasthttp

go 1.19

require (
	github.com/newrelic/go-agent/v3 v3.26.0
	github.com/stretchr/testify v1.8.4
	github.com/valyala/fasthttp v1.48.0
)
replace github.com/newrelic/go-agent/v3 => ../..
