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
		newrelic.ConfigAppName("Anthropic Simple Example"),
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

	// Set NR_ANTHROPIC_BASE_URL_NR to override (defaults to the NR staging proxy).
	baseURL := os.Getenv("NR_ANTHROPIC_BASE_URL_NR")
	// The NR proxy uses its own model slugs (see GET /v1/models).
	// Set NR_ANTHROPIC_MODEL to override.
	model := os.Getenv("NR_ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	client := anthropic.NewClient(
		option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY_AIR")),
		option.WithBaseURL(baseURL),
	)

	nrClient := nranthropic.NewClient(app, &client)

	// Send a message
	prompt := "Write a haiku about programming in Go"
	message, err := nrClient.Messages.New(context.TODO(), anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		log.Fatalf("Error creating message: %v", err)
	}

	fmt.Println("=== Simple Message Example ===")
	fmt.Printf("Prompt: %s\n\n", prompt)
	if len(message.Content) > 0 {
		fmt.Printf("Response: %s\n", message.Content[0].Text)
	}
	fmt.Printf("\nInput tokens:  %d\n", message.Usage.InputTokens)
	fmt.Printf("Output tokens: %d\n", message.Usage.OutputTokens)
}
