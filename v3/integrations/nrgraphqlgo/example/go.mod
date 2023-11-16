module github.com/newrelic/go-agent/v3/integrations/nrgraphqlgo/example

go 1.19

require (
	github.com/graphql-go/graphql v0.8.1
	github.com/graphql-go/graphql-go-handler v0.2.3
	github.com/newrelic/go-agent/v3 v3.26.0
	github.com/newrelic/go-agent/v3/integrations/nrgraphqlgo v1.0.0
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/graphql-go/handler v0.2.3 // indirect
	github.com/klauspost/compress v1.16.3 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.49.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.56.3 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace github.com/newrelic/go-agent/v3/integrations/nrgraphqlgo => ../

replace github.com/newrelic/go-agent/v3 => ../../..
