module github.com/newrelic/go-agent/v3/integrations/nrmicro

// As of Dec 2019, the go-micro go.mod file uses 1.13:
// https://github.com/micro/go-micro/blob/master/go.mod
go 1.13

require (
	github.com/golang/protobuf v1.4.2
	github.com/micro/go-micro v1.8.0
	github.com/nats-io/nats-server/v2 v2.9.16 // indirect
	github.com/newrelic/go-agent/v3 v3.4.0
)
