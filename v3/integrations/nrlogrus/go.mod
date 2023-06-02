module github.com/newrelic/go-agent/v3/integrations/nrlogrus

// As of Dec 2019, the logrus go.mod file uses 1.13:
// https://github.com/sirupsen/logrus/blob/master/go.mod
go 1.18

require (
	github.com/newrelic/go-agent/v3 v3.21.0
	// v1.1.0 is required for the Logger.GetLevel method, and is the earliest
	// version of logrus using modules.
	github.com/sirupsen/logrus v1.1.0
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/konsorten/go-windows-terminal-sequences v0.0.0-20180402223658-b729f2633dfe // indirect
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/grpc v1.49.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)
