module github.com/newrelic/go-agent/v3/integrations/nrpkgerrors

// As of Dec 2019, 1.11 is the earliest version of Go tested by pkg/errors:
// https://github.com/pkg/errors/blob/master/.travis.yml
go 1.11

require (
	github.com/newrelic/go-agent/v3 v3.0.0
	// v0.8.0 was the last release in 2016, and when
	// major development on pkg/errors stopped.
	github.com/pkg/errors v0.8.0
)
