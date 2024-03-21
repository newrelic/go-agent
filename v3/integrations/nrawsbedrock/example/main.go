//
// Example Bedrock client application with New Relic instrumentation
//
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/newrelic/go-agent/v3/integrations/nrawsbedrock"
	"github.com/newrelic/go-agent/v3/newrelic"
)

const region = "us-east-2"

func main() {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		panic(err)
	}
	bedrockClient := bedrock.NewFromConfig(sdkConfig)
	result, err := bedrockClient.ListFoundationModels(context.TODO(), &bedrock.ListFoundationModelsInput{})
	if err != nil {
		panic(err)
	}
	if len(result.ModelSummaries) == 0 {
		fmt.Println("no models found")
	}
	for _, modelSummary := range result.ModelSummaries {
		fmt.Printf("Name: %30s | Provider: %20s | ID: %s\n", *modelSummary.ModelName, *modelSummary.ProviderName, *modelSummary.ModelId)
	}

	// Create a New Relic application. This will look for your license key in an
	// environment variable called NEW_RELIC_LICENSE_KEY. This example turns on
	// Distributed Tracing, but that's not required.
	app, err := newrelic.NewApplication(
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigAppName("Example App"),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigAIMonitoringEnabled(true),
	)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	// For demo purposes only. Don't use the app.WaitForConnection call in
	// production unless this is a very short-lived process and the caller
	// doesn't block or exit if there's an error.
	app.WaitForConnection(5 * time.Second)

	// Start recording a New Relic transaction
	txn := app.StartTransaction("My sample transaction")

	model := "amazon.titan-text-lite-v1"
	//model := "amazon.titan-embed-g1-text-02"
	//model := "amazon.titan-text-express-v1"
	brc := bedrockruntime.NewFromConfig(sdkConfig)
	//output, err := brc.InvokeModel(context.Background(), &bedrockruntime.InvokeModelInput{
	output, err := nrawsbedrock.InvokeModel(app, brc, context.Background(), &bedrockruntime.InvokeModelInput{
		Body: []byte(`{
			"inputText": "What is your quest?",
			"textGenerationConfig": {
				"temperature": 0.5,
				"maxTokenCount": 100
			}
		}`),
		ModelId: &model,
	})
	if err != nil {
		fmt.Printf("error: %v\n", err)
	} else {
		fmt.Printf("%v\n", output)
	}

	// End the New Relic transaction
	txn.End()

	// Force all the harvests and shutdown. Like the app.WaitForConnection call
	// above, this is for the purposes of this demo only and can be safely
	// removed for longer-running processes.
	app.Shutdown(10 * time.Second)
}
