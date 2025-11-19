module github.com/newrelic/go-agent/v3/integrations/nrfiber

go 1.24

require (
	github.com/gofiber/fiber/v2 v2.52.9
	github.com/newrelic/go-agent/v3 v3.42.0
	github.com/stretchr/testify v1.10.0
	github.com/valyala/fasthttp v1.51.0
)


replace github.com/newrelic/go-agent/v3 => ../..
