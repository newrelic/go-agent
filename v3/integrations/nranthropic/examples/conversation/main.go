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
		newrelic.ConfigAppName("Anthropic Conversation Example"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Start a transaction
	txn := app.StartTransaction("multi-turn-conversation")
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

	fmt.Println("=== Multi-turn Conversation Example ===")
	fmt.Println()

	// Build conversation history across turns
	var history []anthropic.MessageParam

	turns := []string{
		"What is a goroutine in Go?",
		"Can you give me a simple example?",
		"What are the main differences between goroutines and threads?",
	}

	for _, userText := range turns {
		fmt.Printf("User: %s\n", userText)

		// Append the new user message to history
		history = append(history, anthropic.NewUserMessage(anthropic.NewTextBlock(userText)))

		// Send the full conversation history
		message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(model),
			MaxTokens: 1024,
			Messages:  history,
		})
		if err != nil {
			log.Fatalf("Error creating message: %v", err)
		}

		// Extract and print the assistant reply
		var replyText string
		if len(message.Content) > 0 {
			replyText = message.Content[0].Text
		}
		fmt.Printf("Assistant: %s\n\n", replyText)

		// Add the assistant reply to history for the next turn
		history = append(history, anthropic.NewAssistantMessage(anthropic.NewTextBlock(replyText)))
	}

	fmt.Printf("Total conversation turns: %d\n", len(history))
}
