module github.com/newrelic/go-agent/v4/integrations/nrmicro

// As of Dec 2019, the go-micro go.mod file uses 1.13:
// https://github.com/micro/go-micro/blob/master/go.mod
go 1.13

replace github.com/newrelic/go-agent/v4 => ../../

require (
	github.com/golang/protobuf v1.4.2
	github.com/micro/go-micro v1.8.0
	github.com/nats-io/nats-server/v2 v2.1.9 // indirect
	github.com/newrelic/go-agent/v4 v4.0.0
	google.golang.org/grpc/examples v0.0.0-20210223174733-dabedfb38b74 // indirect
)
