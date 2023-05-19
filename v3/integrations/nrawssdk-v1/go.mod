module github.com/newrelic/go-agent/v3/integrations/nrawssdk-v1

// As of Dec 2019, aws-sdk-go's go.mod does not specify a Go version.  1.6 is
// the earliest version of Go tested by aws-sdk-go's CI:
// https://github.com/aws/aws-sdk-go/blob/master/.travis.yml
go 1.7

require (
	// v1.15.0 is the first aws-sdk-go version with module support.
	github.com/aws/aws-sdk-go v1.33.0
	github.com/newrelic/go-agent/v3 v3.16.0
)
