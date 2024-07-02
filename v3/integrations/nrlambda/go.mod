module github.com/newrelic/go-agent/v3/integrations/nrlambda

go 1.20

require (
	github.com/aws/aws-lambda-go v1.41.0
	github.com/newrelic/go-agent/v3 v3.33.1
)


replace github.com/newrelic/go-agent/v3 => ../..
