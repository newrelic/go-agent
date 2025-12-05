module github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrlogrus

go 1.24

require (
	github.com/newrelic/go-agent/v3 v3.42.0
	github.com/sirupsen/logrus v1.8.3
)

require (
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/grpc v1.65.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace github.com/newrelic/go-agent/v3 => ../../..
