module github.com/newrelic/go-agent/v3/integrations/nrmicro

// As of Dec 2019, the go-micro go.mod file uses 1.13:
// https://github.com/micro/go-micro/blob/master/go.mod
go 1.24

toolchain go1.24.2

require (
	github.com/golang/protobuf v1.5.4
	github.com/micro/go-micro v1.8.0
	github.com/newrelic/go-agent/v3 v3.41.0
	google.golang.org/protobuf v1.36.6
)


replace github.com/newrelic/go-agent/v3 => ../..
