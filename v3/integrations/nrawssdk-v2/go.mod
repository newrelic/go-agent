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

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.8 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.12.19 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.0.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.7.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.13.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.18 // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	golang.org/x/net v0.0.0-20200822124328-c89045814202 // indirect
	golang.org/x/sys v0.0.0-20200323222414-85ca7c5b95cd // indirect
	golang.org/x/text v0.3.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/grpc v1.39.0 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)
