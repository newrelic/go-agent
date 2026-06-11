package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/newrelic/go-agent/v3/integrations/nranthropic"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	// Initialize New Relic
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Anthropic Streaming Example"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigAIMonitoringEnabled(true),
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := app.WaitForConnection(5 * time.Second); err != nil {
		log.Fatalf("New Relic failed to connect: %v", err)
	}
	defer app.Shutdown(10 * time.Second)

	// Start a transaction
	txn := app.StartTransaction("streaming-message")
	defer txn.End()

	ctx := newrelic.NewContext(context.Background(), txn)
	// Set NR_ANTHROPIC_BASE_URL_NR to override (defaults to the NR staging proxy).
	baseURL := os.Getenv("NR_ANTHROPIC_BASE_URL_NR")
	// The NR proxy uses its own model slugs (see GET /v1/models).
	// Set NR_ANTHROPIC_MODEL to override.
	model := os.Getenv("NR_ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	nrClient := nranthropic.NewClient(app,
		option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY_AIR")),
		option.WithBaseURL(baseURL),
	)

	prompt := "Explain the benefits of using Go for backend services in 3 points"
	fmt.Println("=== Streaming Message Example ===")
	fmt.Printf("Prompt: %s\n\n", prompt)
	fmt.Print("Response: ")

	// Create a streaming message
	stream := nrClient.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
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

	if err := stream.Close(); err != nil {
		log.Fatalf("Stream close error: %v", err)
	}

	fmt.Println()
}
