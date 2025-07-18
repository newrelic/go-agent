module github.com/newrelic/go-agent/v3/integrations/logcontext/nrlogrusplugin

// As of Dec 2019, the logrus go.mod file uses 1.13:
// https://github.com/sirupsen/logrus/blob/master/go.mod
go 1.22

require (
	github.com/newrelic/go-agent/v3 v3.40.1
	// v1.4.0 is required for for the log.WithContext.
	github.com/sirupsen/logrus v1.4.0
)


replace github.com/newrelic/go-agent/v3 => ../../..
