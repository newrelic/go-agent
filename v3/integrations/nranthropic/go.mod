module github.com/newrelic/go-agent/v3/integrations/nranthropic

go 1.25

require (
	github.com/anthropics/anthropic-sdk-go v1.2.0
	github.com/google/uuid v1.6.0
	github.com/newrelic/go-agent/v3 v3.43.3
)

replace github.com/newrelic/go-agent/v3 => ../..

require (
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
	google.golang.org/grpc v1.80.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
