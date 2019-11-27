module github.com/newrelic/go-agent/v3/integrations/logcontext/nrlogrusplugin

go 1.13

require (
	github.com/newrelic/go-agent/v3 v3.0.0
	// v1.4.0 is required for for the log.WithContext.
	github.com/sirupsen/logrus v1.4.0
)
