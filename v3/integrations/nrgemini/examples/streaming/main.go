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
		newrelic.ConfigAppName("Gemini Streaming Example"),
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
	nrClient, err := nrgemini.NewClient(app, ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatalf("Error creating Gemini client: %v", err)
	}

	prompt := "Explain the benefits of using Go for backend services in 3 points"
	fmt.Println("=== Streaming Message Example ===")
	fmt.Printf("Prompt: %s\n\n", prompt)
	fmt.Print("Response: ")

	for chunk, err := range nrClient.Models.GenerateContentStream(ctx, "gemini-2.5-flash", genai.Text(prompt), nil) {
		if err != nil {
			log.Fatalf("Stream error: %v", err)
		}
		if len(chunk.Candidates) == 0 || chunk.Candidates[0].Content == nil {
			continue
		}
		for _, part := range chunk.Candidates[0].Content.Parts {
			if part != nil && part.Text != "" {
				fmt.Print(part.Text)
			}
		}
	}
	fmt.Println()
}
