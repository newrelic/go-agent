module github.com/newrelic/go-agent/v3/integrations/nrawssdk-v2

// As of May 2021, the aws-sdk-go-v2 go.mod file uses 1.15:
// https://github.com/aws/aws-sdk-go-v2/blob/master/go.mod
go 1.15

replace github.com/newrelic/go-agent/v3 => ../../

require (
	github.com/aws/aws-sdk-go-v2 v1.4.0
	github.com/aws/aws-sdk-go-v2/config v1.1.7
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.2.3
	github.com/aws/aws-sdk-go-v2/service/lambda v1.2.3
	github.com/aws/aws-sdk-go-v2/service/s3 v1.6.0
	github.com/aws/smithy-go v1.4.0
	github.com/newrelic/go-agent/v3 v3.0.0
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/sys v0.0.0-20210119212857-b64e53b001e4 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)
