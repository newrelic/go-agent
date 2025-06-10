// Example Bedrock client application with New Relic instrumentation
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/newrelic/go-agent/v3/integrations/nrawsbedrock"
	"github.com/newrelic/go-agent/v3/newrelic"
)

const region = "us-east-1"

func main() {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		panic(err)
	}

	// Create a New Relic application. This will look for your license key in an
	// environment variable called NEW_RELIC_LICENSE_KEY. This example turns on
	// Distributed Tracing, but that's not required.
	app, err := newrelic.NewApplication(
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigAppName("Example Bedrock App"),
		newrelic.ConfigDebugLogger(os.Stdout),
		//newrelic.ConfigInfoLogger(os.Stdout),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigAIMonitoringEnabled(true),
		newrelic.ConfigAIMonitoringRecordContentEnabled(true),
	)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	// For demo purposes only. Don't use the app.WaitForConnection call in
	// production unless this is a very short-lived process and the caller
	// doesn't block or exit if there's an error.
	app.WaitForConnection(5 * time.Second)

	listModels(sdkConfig)

	brc := bedrockruntime.NewFromConfig(sdkConfig)
	simpleEmbedding(app, brc)
	simpleChatCompletionError(app, brc)
	simpleChatCompletion(app, brc)
	processedChatCompletionStream(app, brc)
	manualChatCompletionStream(app, brc)

	app.Shutdown(10 * time.Second)
}

func listModels(sdkConfig aws.Config) {
	fmt.Println("================================================== MODELS")
	bedrockClient := bedrock.NewFromConfig(sdkConfig)
	result, err := bedrockClient.ListFoundationModels(context.TODO(), &bedrock.ListFoundationModelsInput{})
	if err != nil {
		panic(err)
	}
	if len(result.ModelSummaries) == 0 {
		fmt.Println("no models found")
	}
	for _, modelSummary := range result.ModelSummaries {
		fmt.Printf("Name: %-30s | Provider: %-20s | ID: %s\n", *modelSummary.ModelName, *modelSummary.ProviderName, *modelSummary.ModelId)
	}
}

func simpleChatCompletionError(app *newrelic.Application, brc *bedrockruntime.Client) {
	fmt.Println("================================================== CHAT COMPLETION WITH ERROR")
	// Start recording a New Relic transaction
	txn := app.StartTransaction("demo-chat-completion-error")

	contentType := "application/json"
	model := "amazon.titan-text-lite-v1"
	//
	// without nrawsbedrock instrumentation, the call to invoke the model would be:
	//    output, err := brc.InvokeModel(context.Background(), &bedrockruntime.InvokeModelInput{
	//       ...
	//    })
	//
	_, err := nrawsbedrock.InvokeModel(app, brc, newrelic.NewContext(context.Background(), txn), &bedrockruntime.InvokeModelInput{
		ContentType: &contentType,
		Accept:      &contentType,
		Body: []byte(`{
			"inputTexxt": "What is your quest?",
			"textGenerationConfig": {
				"temperature": 0.5,
				"maxTokenCount": 100,
				"stopSequences": [],
				"topP": 1
			}
		}`),
		ModelId: &model,
	})

	txn.End()

	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
}

func simpleEmbedding(app *newrelic.Application, brc *bedrockruntime.Client) {
	fmt.Println("================================================== EMBEDDING")
	// Start recording a New Relic transaction
	contentType := "application/json"
	model := "amazon.titan-embed-text-v1"
	//
	// without nrawsbedrock instrumentation, the call to invoke the model would be:
	//    output, err := brc.InvokeModel(context.Background(), &bedrockruntime.InvokeModelInput{
	//       ...
	//    })
	//
	output, err := nrawsbedrock.InvokeModel(app, brc, context.Background(), &bedrockruntime.InvokeModelInput{
		ContentType: &contentType,
		Accept:      &contentType,
		Body: []byte(`{
			"inputText": "What is your quest?"
		}`),
		ModelId: &model,
	})

	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	if output != nil {
		fmt.Printf("Result: %v\n", string(output.Body))
	}
}

func simpleChatCompletion(app *newrelic.Application, brc *bedrockruntime.Client) {
	fmt.Println("================================================== COMPLETION")
	// Start recording a New Relic transaction
	txn := app.StartTransaction("demo-chat-completion")

	contentType := "application/json"
	model := "amazon.titan-text-lite-v1"
	//
	// without nrawsbedrock instrumentation, the call to invoke the model would be:
	//    output, err := brc.InvokeModel(context.Background(), &bedrockruntime.InvokeModelInput{
	//       ...
	//    })
	//
	app.SetLLMTokenCountCallback(func(model, data string) int { return 42 })
	output, err := nrawsbedrock.InvokeModel(app, brc, newrelic.NewContext(context.Background(), txn), &bedrockruntime.InvokeModelInput{
		ContentType: &contentType,
		Accept:      &contentType,
		Body: []byte(`{
			"inputText": "What is your quest?",
			"textGenerationConfig": {
				"temperature": 0.5,
				"maxTokenCount": 100,
				"stopSequences": [],
				"topP": 1
			}
		}`),
		ModelId: &model,
	})

	txn.End()
	app.SetLLMTokenCountCallback(nil)

	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

	if output != nil {
		fmt.Printf("Result: %v\n", string(output.Body))
	}
}

// This example shows a stream invocation where we let the nrawsbedrock integration retrieve
// all the stream output for us.
func processedChatCompletionStream(app *newrelic.Application, brc *bedrockruntime.Client) {
	fmt.Println("================================================== STREAM (PROCESSED)")
	contentType := "application/json"
	model := "anthropic.claude-v2"

	err := nrawsbedrock.ProcessModelWithResponseStreamAttributes(app, brc, context.Background(), func(data []byte) error {
		fmt.Printf(">>> Received %s\n", string(data))
		return nil
	}, &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     &model,
		ContentType: &contentType,
		Accept:      &contentType,
		Body: []byte(`{
			"prompt": "Human: Tell me a story.\n\nAssistant:",
			"max_tokens_to_sample": 200,
			"temperature": 0.5
		}`),
	}, map[string]any{
		"llm.what_is_this": "processed stream invocation",
	})

	if err != nil {
		fmt.Printf("ERROR processing model: %v\n", err)
	}
}

// This example shows a stream invocation where we manually process the retrieval
// of the stream output.
func manualChatCompletionStream(app *newrelic.Application, brc *bedrockruntime.Client) {
	fmt.Println("================================================== STREAM (MANUAL)")
	contentType := "application/json"
	model := "anthropic.claude-v2"

	output, err := nrawsbedrock.InvokeModelWithResponseStreamAttributes(app, brc, context.Background(), &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     &model,
		ContentType: &contentType,
		Accept:      &contentType,
		Body: []byte(`{
			"prompt": "Human: Tell me a story.\n\nAssistant:",
			"max_tokens_to_sample": 200,
			"temperature": 0.5
		}`)},
		map[string]any{
			"llm.what_is_this": "manual chat completion stream",
		},
	)

	if err != nil {
		fmt.Printf("ERROR processing model: %v\n", err)
		return
	}

	stream := output.Response.GetStream()
	for event := range stream.Events() {
		switch v := event.(type) {
		case *types.ResponseStreamMemberChunk:
			fmt.Println("=====[event received]=====")
			fmt.Println(string(v.Value.Bytes))
			output.RecordEvent(v.Value.Bytes)
		default:
			fmt.Println("=====[unknown value received]=====")
		}
	}
	output.Close()
	stream.Close()
}
