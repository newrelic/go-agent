module github.com/newrelic/go-agent/v3

go 1.19

require (
	github.com/golang/protobuf v1.5.3
	golang.org/x/exp v0.0.0-20240318143956-a85f2c67cd81
	google.golang.org/grpc v1.56.3
)


retract v3.22.0 // release process error corrected in v3.22.1

retract v3.25.0 // release process error corrected in v3.25.1
