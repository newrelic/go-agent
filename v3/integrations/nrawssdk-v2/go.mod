module github.com/newrelic/go-agent/v3/integrations/nrawssdk-v2

// As of May 2021, the aws-sdk-go-v2 go.mod file uses 1.15:
// https://github.com/aws/aws-sdk-go-v2/blob/master/go.mod
go 1.21

toolchain go1.21.0

require (
	github.com/aws/aws-sdk-go-v2 v1.30.4
	github.com/aws/aws-sdk-go-v2/config v1.27.31
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.34.6
	github.com/aws/aws-sdk-go-v2/service/lambda v1.58.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.61.0
	github.com/aws/aws-sdk-go-v2/service/sqs v1.34.6
	github.com/aws/smithy-go v1.20.4
	github.com/newrelic/go-agent/v3 v3.35.0
)


replace github.com/newrelic/go-agent/v3 => ../..
