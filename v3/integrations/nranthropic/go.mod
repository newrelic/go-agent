module github.com/newrelic/go-agent/v3/integrations/nranthropic

go 1.25

require (
	github.com/anthropics/anthropic-sdk-go v1.2.0
	github.com/google/uuid v1.6.0
	github.com/newrelic/go-agent/v3 v3.44.1
)


replace github.com/newrelic/go-agent/v3 => ../..
