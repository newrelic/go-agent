module github.com/newrelic/go-agent/v3/integrations/nrgin

// As of Dec 2019, the gin go.mod file uses 1.12:
// https://github.com/gin-gonic/gin/blob/master/go.mod
go 1.12

require (
	github.com/gin-gonic/gin v1.4.0
	github.com/newrelic/go-agent/v3 v3.0.0
)
