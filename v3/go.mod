module github.com/newrelic/go-agent/v3

go 1.20

require (
	github.com/golang/protobuf v1.5.3
	google.golang.org/grpc v1.56.3
)


retract v3.22.0 // release process error corrected in v3.22.1

retract v3.25.0 // release process error corrected in v3.25.1
