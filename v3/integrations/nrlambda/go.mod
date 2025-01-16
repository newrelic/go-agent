module github.com/newrelic/go-agent/v3/integrations/nrlambda

go 1.21

require (
	github.com/aws/aws-lambda-go v1.41.0
	github.com/newrelic/go-agent/v3 v3.36.0
)


replace github.com/newrelic/go-agent/v3 => ../..
