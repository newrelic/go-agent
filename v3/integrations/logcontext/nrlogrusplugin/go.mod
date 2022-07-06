module github.com/newrelic/go-agent/v3/integrations/logcontext/nrlogrusplugin

// As of Dec 2019, the logrus go.mod file uses 1.13:
// https://github.com/sirupsen/logrus/blob/master/go.mod
go 1.13

require (
	github.com/newrelic/go-agent/v3 v3.17.0
	// v1.4.0 is required for for the log.WithContext.
	github.com/sirupsen/logrus v1.4.0
	golang.org/x/sys v0.0.0-20220704084225-05e143d24a9e // indirect
)
