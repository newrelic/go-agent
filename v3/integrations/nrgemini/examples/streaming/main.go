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

	// Two ways to consume a stream. Each is scoped to its own function so that the
	// deferred Close runs when that stream is done, not at the end of main.
	streamManually(ctx, nrClient, prompt)
	streamWithCallback(ctx, nrClient, prompt)
}

// printChunk writes whatever text a chunk carried.
func printChunk(chunk *genai.GenerateContentResponse) {
	if len(chunk.Candidates) == 0 || chunk.Candidates[0].Content == nil {
		return
	}
	for _, part := range chunk.Candidates[0].Content.Parts {
		if part != nil && part.Text != "" {
			fmt.Print(part.Text)
		}
	}
}

// streamManually drives the SDK iterator itself, reporting each chunk to New
// Relic. Close records the events, so defer it to cover an early return.
func streamManually(ctx context.Context, nrClient *nrgemini.NRClient, prompt string) {
	fmt.Println("=== Streaming Message Example ===")
	fmt.Printf("Prompt: %s\n\n", prompt)
	fmt.Print("Response: ")

	stream := nrClient.Models.GenerateContentStream(ctx, "gemini-2.5-flash", genai.Text(prompt), nil)
	defer stream.Close()

	for chunk, err := range stream.Stream {
		stream.RecordEvent(chunk, err)
		if err != nil {
			log.Printf("Stream error: %v", err)
			break
		}
		printChunk(chunk)
	}
	fmt.Println()
}

// streamWithCallback lets ProcessContentStream run the loop above for you.
func streamWithCallback(ctx context.Context, nrClient *nrgemini.NRClient, prompt string) {
	fmt.Println("\n=== Callback Example ===")
	fmt.Print("Response: ")

	err := nrClient.Models.ProcessContentStream(ctx, "gemini-2.5-flash", genai.Text(prompt), nil,
		func(chunk *genai.GenerateContentResponse) error {
			printChunk(chunk)
			return nil
		})
	if err != nil {
		log.Printf("Stream error: %v", err)
	}
	fmt.Println()
}
