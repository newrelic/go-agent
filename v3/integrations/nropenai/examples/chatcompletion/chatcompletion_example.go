package main

import (
	"fmt"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/nropenai"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/pkoukk/tiktoken-go"
	openai "github.com/sashabaranov/go-openai"
)

func main() {
	// Start New Relic Application
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Basic OpenAI App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
		// Enable AI Monitoring
		// NOTE - If High Security Mode is enabled, AI Monitoring will always be disabled
		newrelic.ConfigAIMonitoringEnabled(true),
	)
	if nil != err {
		panic(err)
	}
	app.WaitForConnection(10 * time.Second)

	app.SetLLMTokenCountCallback(func(modelName string, content string) int {
		var tokensPerMessage, tokensPerName int
		switch modelName {
		case "gpt-3.5-turbo-0613",
			"gpt-3.5-turbo-16k-0613",
			"gpt-4-0314",
			"gpt-4-32k-0314",
			"gpt-4-0613",
			"gpt-4-32k-0613":
			tokensPerMessage = 3
			tokensPerName = 1
		case "gpt-3.5-turbo-0301":
			tokensPerMessage = 4
			tokensPerName = -1
		}

		tkm, err := tiktoken.EncodingForModel(modelName)
		if err != nil {
			fmt.Println("error getting tokens", err)
			return 0
		}
		token := tkm.Encode(content, nil, nil)
		totalTokens := len(token) + tokensPerMessage + tokensPerName
		return totalTokens
	})

	// OpenAI Config - Additionally, NRDefaultAzureConfig(apiKey, baseURL string) can be used for Azure
	cfg := nropenai.NRDefaultConfig(os.Getenv("OPEN_AI_API_KEY"))

	// Create OpenAI Client - Additionally, NRNewClient(authToken string) can be used
	client := nropenai.NRNewClientWithConfig(cfg)

	// Add any custom attributes
	// NOTE: Attributes must start with "llm.", otherwise they will be ignored
	client.AddCustomAttributes(map[string]interface{}{
		"llm.foo": "bar",
		"llm.pi":  3.14,
	})

	// GPT Request
	req := openai.ChatCompletionRequest{
		Model:       openai.GPT4,
		Temperature: 0.7,
		MaxTokens:   150,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Send example text",
			},
		},
	}
	// NRCreateChatCompletion returns a wrapped version of openai.ChatCompletionResponse
	resp, err := nropenai.NRCreateChatCompletion(client, req, app)

	if err != nil {
		panic(err)
	}

	fmt.Println(resp.ChatCompletionResponse.Choices[0].Message.Content)

	// Shutdown Application
	app.Shutdown(5 * time.Second)
}
