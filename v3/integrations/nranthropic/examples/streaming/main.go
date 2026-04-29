package main

import (
	"context"
	"fmt"
	"log"
	"os"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	// Initialize New Relic
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Anthropic Streaming Example"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Start a transaction
	txn := app.StartTransaction("streaming-message")
	defer txn.End()

	ctx := newrelic.NewContext(context.Background(), txn)

	// Create Anthropic client pointed at the NR proxy.
	// Set NR_ANTHROPIC_BASE_URL to override (defaults to the NR staging proxy).
	baseURL := os.Getenv("NR_ANTHROPIC_BASE_URL_NR")
	// The NR proxy uses its own model slugs (see GET /v1/models).
	// Set NR_ANTHROPIC_MODEL to override.
	model := os.Getenv("NR_ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-3-5-sonnet"
	}
	client := anthropic.NewClient(
		option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
		option.WithBaseURL(baseURL),
	)

	prompt := "Explain the benefits of using Go for backend services in 3 points"
	fmt.Println("=== Streaming Message Example ===")
	fmt.Printf("Prompt: %s\n\n", prompt)
	fmt.Print("Response: ")

	// Create a streaming message
	stream := client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})

	// Process stream events as they arrive
	for stream.Next() {
		event := stream.Current()
		switch v := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			switch delta := v.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				fmt.Print(delta.Text)
			}
		}
	}

	if err := stream.Err(); err != nil {
		log.Fatalf("Stream error: %v", err)
	}

}
