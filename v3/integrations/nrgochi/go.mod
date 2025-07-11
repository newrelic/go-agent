module github.com/newrelic/go-agent/v3/integrations/nrgochi

go 1.22

require (
	github.com/go-chi/chi/v5 v5.2.2
	github.com/newrelic/go-agent/v3 v3.39.0
)

require (
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/grpc v1.65.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace github.com/newrelic/go-agent/v3 => ../..
