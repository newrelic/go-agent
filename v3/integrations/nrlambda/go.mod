module github.com/newrelic/go-agent/v3/integrations/nrlambda

// As of Dec 2019, the aws-lambda-go go.mod uses 1.12:
// https://github.com/aws/aws-lambda-go/blob/master/go.mod
go 1.12

require (
	github.com/aws/aws-lambda-go v1.20.0
	github.com/newrelic/go-agent/v3 v3.4.0
)
