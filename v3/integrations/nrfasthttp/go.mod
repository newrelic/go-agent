module github.com/newrelic/go-agent/v3/integrations/nrfasthttp

go 1.19

require (
	github.com/newrelic/go-agent/v3 v3.26.0
	github.com/valyala/fasthttp v1.49.0
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/klauspost/compress v1.16.3 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/grpc v1.54.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)

replace github.com/newrelic/go-agent/v3 => ../..
