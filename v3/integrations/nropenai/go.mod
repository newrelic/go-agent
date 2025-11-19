module github.com/newrelic/go-agent/v3/integrations/nropenai

go 1.24

require (
	github.com/google/uuid v1.6.0
	github.com/newrelic/go-agent/v3 v3.42.0
	github.com/pkoukk/tiktoken-go v0.1.6
	github.com/sashabaranov/go-openai v1.20.2
)


replace github.com/newrelic/go-agent/v3 => ../..
