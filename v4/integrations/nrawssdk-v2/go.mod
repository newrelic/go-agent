module github.com/newrelic/go-agent/v4/integrations/nrawssdk-v2

// As of Dec 2019, the aws-sdk-go-v2 go.mod file uses 1.12:
// https://github.com/aws/aws-sdk-go-v2/blob/master/go.mod
go 1.12

require (
	// v0.8.0 is the earliest aws-sdk-go-v2 version where
	// dynamodb.DescribeTableRequest.Send takes a context.Context parameter.
	github.com/aws/aws-sdk-go-v2 v0.8.0
	github.com/newrelic/go-agent/v4 v4.0.0
)
