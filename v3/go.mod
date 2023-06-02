module github.com/newrelic/go-agent/v3

go 1.18

require (
	github.com/golang/protobuf v1.5.3
	google.golang.org/grpc v1.54.0
)

require (
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)

retract v3.22.0 // release process error corrected in v3.22.1
