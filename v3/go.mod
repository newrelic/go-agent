module github.com/newrelic/go-agent/v3

go 1.25.0

require (
	google.golang.org/grpc v1.82.1
	google.golang.org/protobuf v1.36.11
)

require (
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
)

retract (
	v3.44.0 // retracting release due to a typo in tagging
	v3.40.0 // this release erronously had deadlocks in utilization.go and incorrectly added aws-sdk-go to the go.mod file
	v3.34.0 // this release erronously referred to and invalid protobuf dependency
	v3.25.0 // release process error corrected in v3.25.1
	v3.22.0 // release process error corrected in v3.22.1
)
