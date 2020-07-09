module github.com/newrelic/go-agent/v3/integrations/nrlogrus

// As of Dec 2019, the logrus go.mod file uses 1.13:
// https://github.com/sirupsen/logrus/blob/master/go.mod
go 1.13

require (
	github.com/newrelic/go-agent/v3 v3.0.0
	// v1.1.0 is required for the Logger.GetLevel method, and is the earliest
	// version of logrus using modules.
	github.com/sirupsen/logrus v1.1.0
)
