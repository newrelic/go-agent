module github.com/newrelic/go-agent/v3/integrations/nrmicro

// As of Dec 2019, the go-micro go.mod file uses 1.13:
// https://github.com/micro/go-micro/blob/master/go.mod
go 1.13

require (
	github.com/golang/protobuf v1.3.2
	github.com/micro/go-micro v1.8.0
	github.com/newrelic/go-agent/v3 v3.4.0
)
