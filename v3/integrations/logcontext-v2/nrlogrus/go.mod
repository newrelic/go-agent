module github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrlogrus

go 1.25

require (
	github.com/newrelic/go-agent/v3 v3.43.1
	github.com/sirupsen/logrus v1.8.3
)


replace github.com/newrelic/go-agent/v3 => ../../..
