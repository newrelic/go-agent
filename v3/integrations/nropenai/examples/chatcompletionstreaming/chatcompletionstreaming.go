package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/nropenai"
	"github.com/newrelic/go-agent/v3/newrelic"
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
		Model:       openai.GPT3Dot5Turbo,
		Temperature: 0.7,
		MaxTokens:   1500,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Say this is a test",
			},
		},
		Stream: true,
	}
	ctx := context.Background()

	stream, err := nropenai.NRCreateChatCompletionStream(client, ctx, req, app)

	if err != nil {
		panic(err)
	}
	fmt.Printf("Stream response: ")
	for {
		var response openai.ChatCompletionStreamResponse
		response, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println("\nStream finished")
			break
		}
		if err != nil {
			fmt.Printf("\nStream error: %v\n", err)
			return
		}

		fmt.Printf(response.Choices[0].Delta.Content)
	}
	stream.Close()
	// Shutdown Application
	app.Shutdown(5 * time.Second)
}
