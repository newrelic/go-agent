module github.com/newrelic/go-agent/v3/integrations/nrzap

// As of Jun 2022, zap has 1.18 in their go.mod file:
// https://github.com/uber-go/zap/blob/master/go.mod
go 1.18

require (
	github.com/newrelic/go-agent/v3 v3.16.1
	go.uber.org/zap v1.21.0
)

require (
	github.com/golang/protobuf v1.4.3 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007 // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/grpc v1.39.0 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)
