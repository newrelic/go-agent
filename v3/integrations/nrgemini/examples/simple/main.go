package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/nrgemini"
	"github.com/newrelic/go-agent/v3/newrelic"
	"google.golang.org/genai"
)

func main() {
	// Initialize New Relic
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Gemini Simple Example"),
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

	ctx := context.Background()
	nrClient, err := nrgemini.NewClient(app, ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatalf("Error creating Gemini client: %v", err)
	}

	// Send a message
	prompt := "Write a haiku about programming in Go"
	resp, err := nrClient.Models.GenerateContent(ctx, "gemini-2.5-flash", genai.Text(prompt), nil)
	if err != nil {
		log.Fatalf("Error generating content: %v", err)
	}

	fmt.Println("=== Simple Message Example ===")
	fmt.Printf("Prompt: %s\n\n", prompt)
	fmt.Printf("Response: %s\n", resp.Text())
	if resp.UsageMetadata != nil {
		fmt.Printf("\nInput tokens:  %d\n", resp.UsageMetadata.PromptTokenCount)
		fmt.Printf("Output tokens: %d\n", resp.UsageMetadata.CandidatesTokenCount)
	}
}
