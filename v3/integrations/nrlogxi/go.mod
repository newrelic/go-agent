module github.com/newrelic/go-agent/v3/integrations/nrlogxi

// As of Dec 2019, logxi requires 1.3+:
// https://github.com/mgutz/logxi#requirements
go 1.18

require (
	// 'v1', at commit aebf8a7d67ab, is the only logxi release.
	github.com/mgutz/logxi v0.0.0-20161027140823-aebf8a7d67ab
	github.com/newrelic/go-agent/v3 v3.21.1
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/grpc v1.54.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
