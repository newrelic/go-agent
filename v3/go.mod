module github.com/newrelic/go-agent/v3

go 1.19

require (
	github.com/golang/protobuf v1.5.3
	github.com/valyala/fasthttp v1.49.0
	google.golang.org/grpc v1.56.3
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/klauspost/compress v1.16.3 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

retract v3.22.0 // release process error corrected in v3.22.1

retract v3.25.0 // release process error corrected in v3.25.1
