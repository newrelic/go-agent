module github.com/newrelic/go-agent/v3/integrations/nrlogxi

// As of Dec 2019, logxi requires 1.3+:
// https://github.com/mgutz/logxi#requirements
go 1.7

require (
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	// 'v1', at commit aebf8a7d67ab, is the only logxi release.
	github.com/mgutz/logxi v0.0.0-20161027140823-aebf8a7d67ab
	github.com/newrelic/go-agent/v3 v3.0.0
)
