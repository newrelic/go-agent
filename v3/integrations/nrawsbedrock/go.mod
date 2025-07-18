module github.com/newrelic/go-agent/v3/integrations/nrawsbedrock

go 1.22

require (
	github.com/aws/aws-sdk-go-v2 v1.26.0
	github.com/aws/aws-sdk-go-v2/config v1.27.4
	github.com/aws/aws-sdk-go-v2/service/bedrock v1.7.3
	github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.7.1
	github.com/google/uuid v1.6.0
	github.com/newrelic/go-agent/v3 v3.40.1
)


replace github.com/newrelic/go-agent/v3 => ../..
