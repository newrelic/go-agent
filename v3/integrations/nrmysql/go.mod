module github.com/lamalex/go-agent/v3/integrations/nrmysql

// 1.10 is the Go version in mysql's go.mod
go 1.20

require (
	// v1.5.0 is the first mysql version to support gomod
	github.com/go-sql-driver/mysql v1.6.0
	// v3.3.0 includes the new location of ParseQuery
	github.com/newrelic/go-agent/v3 v3.33.1
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.56.3 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace github.com/newrelic/go-agent/v3 => ../..
