module github.com/newrelic/go-agent/integrations/logcontext-v2/nrlogrus

go 1.18

replace github.com/newrelic/go-agent/v3 v3.17.0 => ../../../

require (
	github.com/newrelic/go-agent/v3 v3.17.0
	github.com/sirupsen/logrus v1.8.1
)

require (
	github.com/golang/protobuf v1.4.3 // indirect
	golang.org/x/net v0.0.0-20200822124328-c89045814202 // indirect
	golang.org/x/sys v0.0.0-20200323222414-85ca7c5b95cd // indirect
	golang.org/x/text v0.3.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/grpc v1.39.0 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)
