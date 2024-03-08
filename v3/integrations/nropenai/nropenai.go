// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nropenai

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sashabaranov/go-openai"
)

var (
	errAIMonitoringDisabled = errors.New("AI Monitoring is set to disabled or High Security Mode is enabled. Please enable AI Monitoring and ensure High Security Mode is disabled")
)

type OpenAIClient interface {
	CreateChatCompletion(ctx context.Context, request openai.ChatCompletionRequest) (response openai.ChatCompletionResponse, err error)
	CreateEmbeddings(ctx context.Context, conv openai.EmbeddingRequestConverter) (res openai.EmbeddingResponse, err error)
}

// Wrapper for OpenAI Configuration
type ConfigWrapper struct {
	Config             *openai.ClientConfig
	LicenseKeyLastFour string
}

// Wrapper for OpenAI Client with Custom Attributes that can be set for all LLM Events
type ClientWrapper struct {
	Client             OpenAIClient
	LicenseKeyLastFour string
	// Set of Custom Attributes that get tied to all LLM Events
	CustomAttributes map[string]interface{}
}

// Adds Custom Attributes to the ClientWrapper
func (cw *ClientWrapper) AddCustomAttributes(attributes map[string]interface{}) {
	if cw.CustomAttributes == nil {
		cw.CustomAttributes = make(map[string]interface{})
	}

	for key, value := range attributes {
		if strings.HasPrefix(key, "llm.") {
			cw.CustomAttributes[key] = value
		}
	}
}

func AppendCustomAttributesToEvent(cw *ClientWrapper, data map[string]interface{}) map[string]interface{} {
	for k, v := range cw.CustomAttributes {
		data[k] = v
	}
	return data
}

// Wrapper for ChatCompletionResponse that is returned from NRCreateChatCompletion. It also includes the TraceID of the transaction for linking a chat response with it's feedback
type ChatCompletionResponseWrapper struct {
	ChatCompletionResponse openai.ChatCompletionResponse
	TraceID                string
}

func FormatAPIKey(apiKey string) string {
	return "sk-" + apiKey[len(apiKey)-4:]
}

// Default Config
func NRDefaultConfig(authToken string) *ConfigWrapper {
	cfg := openai.DefaultConfig(authToken)
	return &ConfigWrapper{
		Config:             &cfg,
		LicenseKeyLastFour: FormatAPIKey(authToken),
	}
}

// Azure Config
func NRDefaultAzureConfig(apiKey, baseURL string) *ConfigWrapper {
	cfg := openai.DefaultAzureConfig(apiKey, baseURL)
	return &ConfigWrapper{
		Config:             &cfg,
		LicenseKeyLastFour: FormatAPIKey(apiKey),
	}
}

// Call to Create Client Wrapper
func NRNewClient(authToken string) *ClientWrapper {
	client := openai.NewClient(authToken)
	return &ClientWrapper{
		Client:             client,
		LicenseKeyLastFour: FormatAPIKey(authToken),
	}
}

// NewClientWithConfig creates new OpenAI API client for specified config.
func NRNewClientWithConfig(config *ConfigWrapper) *ClientWrapper {
	client := openai.NewClientWithConfig(*config.Config)
	return &ClientWrapper{
		Client:             client,
		LicenseKeyLastFour: config.LicenseKeyLastFour,
	}
}

func NRCreateChatCompletionSummary(txn *newrelic.Transaction, app *newrelic.Application, cw *ClientWrapper, req openai.ChatCompletionRequest) ChatCompletionResponseWrapper {
	// Get App Config for setting App Name Attribute
	appConfig, configErr := app.Config()
	if !configErr {
		txn.NoticeError(newrelic.Error{
			Class:   "OpenAIError",
			Message: "Error getting app config",
		})
	}
	uuid := uuid.New()
	spanID := txn.GetTraceMetadata().SpanID
	traceID := txn.GetTraceMetadata().TraceID
	transactionID := traceID[:16]

	ChatCompletionSummaryData := map[string]interface{}{}

	// Start span
	chatCompletionSpan := txn.StartSegment("Llm/completion/OpenAI/CreateChatCompletion")
	resp, err := cw.Client.CreateChatCompletion(
		context.Background(),
		req,
	)
	chatCompletionSpan.End()
	if err != nil {
		ChatCompletionSummaryData["error"] = true
		// notice error with custom attributes
		txn.NoticeError(newrelic.Error{
			Message: err.Error(),
			Class:   "OpenAIError",
			Attributes: map[string]interface{}{
				"http.status":   resp.Header().Get("Status"),
				"error.code":    resp.Header().Get("Error-Code"),
				"completion_id": uuid.String(),
			},
		})
	}

	// ratelimitLimitTokensUsageBased, ratelimitResetTokensUsageBased, and ratelimitRemainingTokensUsageBased are not in the response
	// Request Headers
	ChatCompletionSummaryData["request.temperature"] = req.Temperature
	ChatCompletionSummaryData["request.max_tokens"] = req.MaxTokens
	ChatCompletionSummaryData["request.model"] = req.Model
	ChatCompletionSummaryData["model"] = req.Model

	// Response Data
	ChatCompletionSummaryData["response.model"] = resp.Model
	ChatCompletionSummaryData["request_id"] = resp.ID
	ChatCompletionSummaryData["response.organization"] = resp.Header().Get("Openai-Organization")
	ChatCompletionSummaryData["response.number_of_messages"] = len(resp.Choices)
	ChatCompletionSummaryData["response.usage.total_tokens"] = resp.Usage.TotalTokens
	ChatCompletionSummaryData["response.usage.prompt_tokens"] = resp.Usage.PromptTokens
	ChatCompletionSummaryData["response.usage.completion_tokens"] = resp.Usage.CompletionTokens
	// TO:DO - Verify this is the correct method of getting FinishReason
	// ChatCompletionSummaryData["response.choices.finish_reason"] = resp.Choices[0].FinishReason

	// Response Headers
	ChatCompletionSummaryData["response.headers.llmVersion"] = resp.Header().Get("Openai-Version")
	ChatCompletionSummaryData["response.headers.ratelimitLimitRequests"] = resp.Header().Get("X-Ratelimit-Limit-Requests")
	ChatCompletionSummaryData["response.headers.ratelimitLimitTokens"] = resp.Header().Get("X-Ratelimit-Limit-Tokens")
	ChatCompletionSummaryData["response.headers.ratelimitResetTokens"] = resp.Header().Get("X-Ratelimit-Reset-Tokens")
	ChatCompletionSummaryData["response.headers.ratelimitResetRequests"] = resp.Header().Get("X-Ratelimit-Reset-Requests")
	ChatCompletionSummaryData["response.headers.ratelimitRemainingTokens"] = resp.Header().Get("X-Ratelimit-Remaining-Tokens")
	ChatCompletionSummaryData["response.headers.ratelimitRemainingRequests"] = resp.Header().Get("X-Ratelimit-Remaining-Requests")

	// New Relic Attributes
	ChatCompletionSummaryData["id"] = uuid.String()
	ChatCompletionSummaryData["span_id"] = spanID
	ChatCompletionSummaryData["transaction_id"] = transactionID
	ChatCompletionSummaryData["trace_id"] = traceID
	ChatCompletionSummaryData["api_key_last_four_digits"] = cw.LicenseKeyLastFour
	ChatCompletionSummaryData["vendor"] = "OpenAI"
	ChatCompletionSummaryData["ingest_source"] = "Go"
	ChatCompletionSummaryData["appName"] = appConfig.AppName

	// Record any custom attributes if they exist
	ChatCompletionSummaryData = AppendCustomAttributesToEvent(cw, ChatCompletionSummaryData)

	// Record Custom Event
	app.RecordCustomEvent("LlmChatCompletionSummary", ChatCompletionSummaryData)

	// Capture completion messages
	NRCreateChatCompletionMessage(txn, app, resp, uuid, cw)
	txn.End()

	return ChatCompletionResponseWrapper{
		ChatCompletionResponse: resp,
		TraceID:                traceID,
	}
}

func NRCreateChatCompletionMessage(txn *newrelic.Transaction, app *newrelic.Application, resp openai.ChatCompletionResponse, uuid uuid.UUID, cw *ClientWrapper) {
	spanID := txn.GetTraceMetadata().SpanID
	traceID := txn.GetTraceMetadata().TraceID
	transactionID := traceID[:16]
	appCfg, err := app.Config()
	if !err {
		txn.NoticeError(newrelic.Error{
			Class:   "OpenAIError",
			Message: "Error getting app config",
		})
	}

	chatCompletionMessageSpan := txn.StartSegment("Llm/completion/OpenAI/CreateChatCompletionMessage")
	for i, choice := range resp.Choices {
		ChatCompletionMessageData := map[string]interface{}{}
		// if the response doesn't have an ID, use the UUID from the summary
		if resp.ID == "" {
			ChatCompletionMessageData["id"] = uuid.String()
		} else {
			ChatCompletionMessageData["id"] = resp.ID
		}

		// Response Data
		ChatCompletionMessageData["response.model"] = resp.Model

		if appCfg.AIMonitoring.RecordContent.Enabled {
			ChatCompletionMessageData["content"] = choice.Message.Content
		}

		ChatCompletionMessageData["role"] = choice.Message.Role

		// Request Headers
		ChatCompletionMessageData["request_id"] = resp.Header().Get("X-Request-Id")

		// New Relic Attributes
		ChatCompletionMessageData["sequence"] = i
		ChatCompletionMessageData["vendor"] = "openai"
		ChatCompletionMessageData["ingest_source"] = "go"
		ChatCompletionMessageData["span_id"] = spanID
		ChatCompletionMessageData["trace_id"] = traceID
		ChatCompletionMessageData["transaction_id"] = transactionID
		// TO:DO completion_id set in CompletionSummary which is a UUID generated by the agent to identify the event
		// TO:DO - llm.conversation_id

		// If custom attributes are set, add them to the data
		ChatCompletionMessageData = AppendCustomAttributesToEvent(cw, ChatCompletionMessageData)

		// Record Custom Event for each message
		app.RecordCustomEvent("LlmChatCompletionMessage", ChatCompletionMessageData)

	}

	chatCompletionMessageSpan.End()
}

func NRCreateChatCompletion(cw *ClientWrapper, req openai.ChatCompletionRequest, app *newrelic.Application) (ChatCompletionResponseWrapper, error) {
	config, _ := app.Config()
	resp := ChatCompletionResponseWrapper{}
	// If AI Monitoring is disabled, do not start a transaction but still perform the request
	if !config.AIMonitoring.Enabled {
		chatresp, err := cw.Client.CreateChatCompletion(context.Background(), req)
		resp.ChatCompletionResponse = chatresp
		return resp, err
	}
	// Start NR Transaction
	txn := app.StartTransaction("OpenAIChatCompletion")
	resp = NRCreateChatCompletionSummary(txn, app, cw, req)

	return resp, nil
}

// If multiple messages are sent, only the first message is used
func GetInput(any interface{}) any {
	v := reflect.ValueOf(any)
	if v.Kind() == reflect.Array || v.Kind() == reflect.Slice {
		if v.Len() > 0 {
			// Return the first element
			return v.Index(0).Interface()
		}
		// Input passed in is empty
		return ""
	}
	return any

}
func NRCreateEmbedding(cw *ClientWrapper, req openai.EmbeddingRequest, app *newrelic.Application) (openai.EmbeddingResponse, error) {
	config, _ := app.Config()
	resp := openai.EmbeddingResponse{}

	// If AI Monitoring is disabled, do not start a transaction but still perform the request
	if !config.AIMonitoring.Enabled {
		resp, err := cw.Client.CreateEmbeddings(context.Background(), req)
		return resp, err
	}

	// Start NR Transaction
	txn := app.StartTransaction("OpenAIEmbedding")

	spanID := txn.GetTraceMetadata().SpanID
	traceID := txn.GetTraceMetadata().TraceID
	transactionID := traceID[:16]
	EmbeddingsData := map[string]interface{}{}
	uuid := uuid.New()

	embeddingSpan := txn.StartSegment("Llm/completion/OpenAI/CreateEmbedding")
	resp, err := cw.Client.CreateEmbeddings(context.Background(), req)
	embeddingSpan.End()

	if err != nil {
		EmbeddingsData["error"] = true
		txn.NoticeError(newrelic.Error{
			Message: err.Error(),
			Class:   "OpenAIError",
			Attributes: map[string]interface{}{
				"http.status":  resp.Header().Get("Status"),
				"error.code":   resp.Header().Get("Error-Code"),
				"embedding_id": uuid.String(),
			},
		})
	}

	// Request Data
	if config.AIMonitoring.RecordContent.Enabled {
		EmbeddingsData["input"] = GetInput(req.Input)
	}
	EmbeddingsData["api_key_last_four_digits"] = cw.LicenseKeyLastFour
	EmbeddingsData["request.model"] = string(req.Model)

	// Response Data
	EmbeddingsData["response.model"] = string(resp.Model)
	EmbeddingsData["response.usage.total_tokens"] = resp.Usage.TotalTokens
	EmbeddingsData["response.usage.prompt_tokens"] = resp.Usage.PromptTokens

	// Response Headers
	EmbeddingsData["response.organization"] = resp.Header().Get("Openai-Organization")
	EmbeddingsData["response.headers.llmVersion"] = resp.Header().Get("Openai-Version")
	EmbeddingsData["response.headers.ratelimitLimitRequests"] = resp.Header().Get("X-Ratelimit-Limit-Requests")
	EmbeddingsData["response.headers.ratelimitLimitTokens"] = resp.Header().Get("X-Ratelimit-Limit-Tokens")
	EmbeddingsData["response.headers.ratelimitResetTokens"] = resp.Header().Get("X-Ratelimit-Reset-Tokens")
	EmbeddingsData["response.headers.ratelimitResetRequests"] = resp.Header().Get("X-Ratelimit-Reset-Requests")
	EmbeddingsData["response.headers.ratelimitRemainingTokens"] = resp.Header().Get("X-Ratelimit-Remaining-Tokens")
	EmbeddingsData["response.headers.ratelimitRemainingRequests"] = resp.Header().Get("X-Ratelimit-Remaining-Requests")

	EmbeddingsData = AppendCustomAttributesToEvent(cw, EmbeddingsData)

	// New Relic Attributes
	EmbeddingsData["id"] = uuid.String()
	EmbeddingsData["vendor"] = "OpenAI"
	EmbeddingsData["ingest_source"] = "Go"
	EmbeddingsData["span_id"] = spanID
	EmbeddingsData["transaction_id"] = transactionID
	EmbeddingsData["trace_id"] = traceID

	app.RecordCustomEvent("LlmEmbedding", EmbeddingsData)
	txn.End()
	return resp, nil
}
