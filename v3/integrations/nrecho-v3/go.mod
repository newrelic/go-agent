module github.com/newrelic/go-agent/v3/integrations/nrecho-v3

// 1.7 is the earliest version of Go tested by v3.1.0:
// https://github.com/labstack/echo/blob/v3.1.0/.travis.yml
go 1.7

require (
	// v3.1.0 is the earliest v3 version of Echo that works with modules due
	// to the github.com/rsc/letsencrypt import of v3.0.0.
	github.com/labstack/echo v3.1.0+incompatible
	github.com/labstack/gommon v0.3.0 // indirect
	github.com/newrelic/go-agent/v3 v3.17.0
)
