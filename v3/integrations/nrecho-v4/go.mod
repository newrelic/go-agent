module github.com/newrelic/go-agent/v3/integrations/nrecho-v4

// As of Jun 2022, the echo go.mod file uses 1.17:
// https://github.com/labstack/echo/blob/master/go.mod
go 1.17

require (
	github.com/labstack/echo/v4 v4.9.0
	github.com/newrelic/go-agent/v3 v3.18.2
)

require (
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/labstack/gommon v0.3.1 // indirect
	github.com/mattn/go-colorable v0.1.11 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.1 // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f // indirect
	golang.org/x/sys v0.0.0-20211103235746-7861aae1554b // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/grpc v1.39.0 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)
