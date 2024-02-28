package main

import (
	"fmt"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/nropenai"
	"github.com/newrelic/go-agent/v3/newrelic"
	openai "github.com/sashabaranov/go-openai"
)

// Simulates feedback being sent to New Relic. Feedback on a chat completion requires
// having access to the ChatCompletionResponseWrapper which is returned by the NRCreateChatCompletion function.
func SendFeedback(app *newrelic.Application, resp nropenai.ChatCompletionResponseWrapper) {
	trace_id := resp.TraceID
	rating := "5"
	category := "informative"
	message := "The response was concise yet thorough."
	customMetadata := map[string]interface{}{
		"foo": "bar",
		"pi":  3.14,
	}

	app.RecordLlmFeedbackEvent(trace_id, rating, category, message, customMetadata)
}

func main() {
	// Start New Relic Application
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Basic OpenAI App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigAIMonitoringEnabled(true),
	)
	if nil != err {
		panic(err)
	}
	app.WaitForConnection(10 * time.Second)

	// OpenAI Config - Additionally, NRDefaultAzureConfig(apiKey, baseURL string) can be used for Azure
	cfg := nropenai.NRDefaultConfig(os.Getenv("OPEN_AI_API_KEY"))
	client := nropenai.NRNewClientWithConfig(cfg)
	// GPT Request
	req := openai.ChatCompletionRequest{
		Model:       openai.GPT3Dot5Turbo,
		Temperature: 0.7,
		MaxTokens:   150,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "What is 8*5",
			},
		},
	}
	// Create Chat Completion
	resp, err := nropenai.NRCreateChatCompletion(client, req, app)

	if err != nil {
		fmt.Println("Unable to create chat completion: ", err)
	} else {
		// Print the contents of the message
		fmt.Println("Message Response: ", resp.ChatCompletionResponse.Choices[0].Message.Content)
		SendFeedback(app, resp)
	}

	// Shutdown Application
	app.Shutdown(5 * time.Second)
}
