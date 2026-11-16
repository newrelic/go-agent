module github.com/newrelic/go-agent/v3/integrations/nrgemini

go 1.25

require (
	github.com/google/uuid v1.6.0
	github.com/newrelic/go-agent/v3 v3.45.0
	google.golang.org/genai v1.64.0
)


replace github.com/newrelic/go-agent/v3 => ../..
