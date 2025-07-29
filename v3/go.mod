module github.com/newrelic/go-agent/v3

go 1.23.0

toolchain go1.24.2

require (
	github.com/google/pprof v0.0.0-20250630185457-6e76a2b096b5
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
)

require (
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
)

retract v3.22.0 // release process error corrected in v3.22.1

retract v3.25.0 // release process error corrected in v3.25.1

retract v3.34.0 // this release erronously referred to and invalid protobuf dependency

retract v3.40.0 // this release erronously had deadlocks in utilization.go and incorrectly added aws-sdk-go to the go.mod file
