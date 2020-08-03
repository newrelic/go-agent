module github.com/newrelic/go-agent/v4/integrations/nrecho-v4

// As of Dec 2019, the echo go.mod file uses 1.12:
// https://github.com/labstack/echo/blob/master/go.mod
go 1.12

require (
	github.com/labstack/echo/v4 v4.0.0
	github.com/newrelic/go-agent/v4 v4.0.0
)
