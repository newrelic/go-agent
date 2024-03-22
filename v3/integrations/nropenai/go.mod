module github.com/newrelic/go-agent/v3/integrations/nropenai

go 1.21.0

require (
	github.com/google/uuid v1.6.0
	github.com/newrelic/go-agent/v3 v3.30.0
	github.com/pkoukk/tiktoken-go v0.1.6
	github.com/sashabaranov/go-openai v1.20.2
)

require (
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.56.3 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace github.com/newrelic/go-agent/v3 => ../..
