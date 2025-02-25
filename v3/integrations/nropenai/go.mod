module github.com/newrelic/go-agent/v3/integrations/nropenai

go 1.22

require (
	github.com/google/uuid v1.6.0
	github.com/newrelic/go-agent/v3 v3.37.0
	github.com/pkoukk/tiktoken-go v0.1.6
	github.com/sashabaranov/go-openai v1.20.2
)


replace github.com/newrelic/go-agent/v3 => ../..
