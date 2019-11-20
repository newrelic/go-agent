module github.com/newrelic/go-agent/v3/integrations/nrawssdk/v2

go 1.13

require (
	// v0.8.0 is the earliest aws-sdk-go-v2 version where
	// dynamodb.DescribeTableRequest.Send takes a context.Context parameter.
	github.com/aws/aws-sdk-go-v2 v0.8.0
	github.com/newrelic/go-agent/v3 v3.0.0
)

