module github.com/newrelic/go-agent/v3/integrations/nrecho-v4

// As of Jun 2022, the echo go.mod file uses 1.17:
// https://github.com/labstack/echo/blob/master/go.mod
go 1.17


require (
	github.com/labstack/echo/v4 v4.5.0
	github.com/newrelic/go-agent/v3 v3.16.1
)

require (
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/labstack/gommon v0.3.0 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.1 // indirect
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/sys v0.0.0-20210403161142-5e06dd20ab57 // indirect
	golang.org/x/text v0.3.6 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/grpc v1.39.0 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)
