module github.com/newrelic/go-agent/v3/integrations/nrawssdk-v2

// As of May 2021, the aws-sdk-go-v2 go.mod file uses 1.15:
// https://github.com/aws/aws-sdk-go-v2/blob/master/go.mod
go 1.17

require (
	github.com/aws/aws-sdk-go-v2 v1.16.15
	github.com/aws/aws-sdk-go-v2/config v1.17.6
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.17.0
	github.com/aws/aws-sdk-go-v2/service/lambda v1.24.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.27.10
	github.com/aws/smithy-go v1.13.3
	github.com/newrelic/go-agent/v3 v3.18.2
)
