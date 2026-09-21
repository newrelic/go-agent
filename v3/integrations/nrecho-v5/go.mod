module github.com/newrelic/go-agent/v3/integrations/nrecho-v5

// echo v5 requires Go 1.25:
// https://github.com/labstack/echo/blob/v5/go.mod
go 1.25.0

require (
	github.com/labstack/echo/v5 v5.3.1
	github.com/newrelic/go-agent/v3 v3.44.1
)

require (
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
	google.golang.org/grpc v1.80.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/newrelic/go-agent/v3 => ../..
