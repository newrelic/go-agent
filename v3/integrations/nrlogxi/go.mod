module github.com/newrelic/go-agent/v3/integrations/nrlogxi

// As of Dec 2019, logxi requires 1.3+:
// https://github.com/mgutz/logxi#requirements
go 1.22

require (
	// 'v1', at commit aebf8a7d67ab, is the only logxi release.
	github.com/mgutz/logxi v0.0.0-20161027140823-aebf8a7d67ab
	github.com/newrelic/go-agent/v3 v3.37.0
)


replace github.com/newrelic/go-agent/v3 => ../..
