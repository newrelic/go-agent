module github.com/newrelic/go-agent/v3/integrations/nrmysql

// 1.10 is the Go version in mysql's go.mod
go 1.17

require (
	// v1.5.0 is the first mysql version to support gomod
	github.com/go-sql-driver/mysql v1.6.0
	// v3.3.0 includes the new location of ParseQuery
	github.com/newrelic/go-agent/v3 v3.18.2
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
