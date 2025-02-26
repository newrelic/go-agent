module github.com/newrelic/go-agent/v3

go 1.22

require (
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
)


retract v3.22.0 // release process error corrected in v3.22.1

retract v3.25.0 // release process error corrected in v3.25.1

retract v3.34.0 // this release erronously referred to and invalid protobuf dependency
