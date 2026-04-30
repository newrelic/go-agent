module github.com/newrelic/go-agent/v3/integrations/nrrueidis

// Rueidis requires 1.23 as per https://github.com/redis/rueidis/blob/45dbab6deb4481a5873712b18fe7e3927d9e5066/go.mod#L3
go 1.24

require (
	github.com/newrelic/go-agent/v3 v3.41.0
	github.com/redis/rueidis v1.0.66
	github.com/redis/rueidis/rueidishook v1.0.66
	github.com/stretchr/testify v1.8.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/grpc v1.65.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/newrelic/go-agent/v3 => /usr/src/app/go-agent/./v3
