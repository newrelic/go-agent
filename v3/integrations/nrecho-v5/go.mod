module github.com/newrelic/go-agent/v3/integrations/nrecho-v5

// echo v5 requires Go 1.25:
// https://github.com/labstack/echo/blob/v5/go.mod
go 1.25

require (
	github.com/labstack/echo/v5 v5.3.1
	github.com/newrelic/go-agent/v3 v3.45.0
)


replace github.com/newrelic/go-agent/v3 => ../..
