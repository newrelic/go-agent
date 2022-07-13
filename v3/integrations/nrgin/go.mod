module github.com/newrelic/go-agent/v3/integrations/nrgin

// As of Jul 2022, the gin go.mod file uses 1.18:
// https://github.com/gin-gonic/gin/blob/master/go.mod
go 1.18

require (
	github.com/gin-gonic/gin v1.8.0
	github.com/newrelic/go-agent/v3 v3.17.0
)
