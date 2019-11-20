module github.com/newrelic/go-agent/v3/integrations/nrpkgerrors

go 1.13

require (
	github.com/newrelic/go-agent/v3 v3.0.0
	// v0.8.0 was the last release in 2016, and when
	// major development on pkg/errors stopped.
	github.com/pkg/errors v0.8.0
)
