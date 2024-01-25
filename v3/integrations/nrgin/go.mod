module github.com/newrelic/go-agent/v3/integrations/nrgin

// As of Dec 2019, the gin go.mod file uses 1.12:
// https://github.com/gin-gonic/gin/blob/master/go.mod
go 1.19

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/newrelic/go-agent/v3 v3.29.1
)


replace github.com/newrelic/go-agent/v3 => ../..
