module github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrzap

go 1.20

require (
	github.com/newrelic/go-agent/v3 v3.32.0
	go.uber.org/zap v1.24.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.56.3 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace github.com/newrelic/go-agent/v3 => ../../..
