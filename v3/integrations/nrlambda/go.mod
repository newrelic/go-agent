module github.com/newrelic/go-agent/v3/integrations/nrlambda

go 1.22

require (
	github.com/aws/aws-lambda-go v1.41.0
	github.com/newrelic/go-agent/v3 v3.38.0
)


replace github.com/newrelic/go-agent/v3 => ../..
