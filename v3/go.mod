module github.com/newrelic/go-agent/v3

go 1.19

require (
	github.com/golang/protobuf v1.5.3
	github.com/valyala/fasthttp v1.49.0
	google.golang.org/grpc v1.54.0
)

retract v3.22.0 // release process error corrected in v3.22.1
