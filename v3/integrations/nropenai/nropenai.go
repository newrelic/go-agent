// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nropenai

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sashabaranov/go-openai"
)

var reportStreamingDisabled func()

func init() {
	reportStreamingDisabled = sync.OnceFunc(func() {
		internal.TrackUsage("Go", "ML", "Streaming", "Disabled")
	})
	// Get current go-openai version
	info, ok := debug.ReadBuildInfo()
	if info != nil && ok {
		for _, module := range info.Deps {
			if module != nil && strings.Contains(module.Path, "go-openai") {

				internal.TrackUsage("Go", "ML", "OpenAI", module.Version)

				return
			}
		}
	}
	internal.TrackUsage("Go", "ML", "OpenAI", "unknown")

}

var (
	errAIMonitoringDisabled = errors.New("AI Monitoring is set to disabled or High Security Mode is enabled. Please enable AI Monitoring and ensure High Security Mode is disabled")
)

type OpenAIClient interface {
	CreateChatCompletion(ctx context.Context, request openai.ChatCompletionRequest) (response openai.ChatCompletionResponse, err error)
	CreateChatCompletionStream(ctx context.Context, request openai.ChatCompletionRequest) (stream *openai.ChatCompletionStream, err error)
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

// If multiple messages are sent, only the first message is used for the "content" field
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

// Wrapper for ChatCompletionResponse that is returned from NRCreateChatCompletion. It also includes the TraceID of the transaction for linking a chat response with it's feedback
type ChatCompletionResponseWrapper struct {
	ChatCompletionResponse openai.ChatCompletionResponse
	TraceID                string
}

// Wrapper for ChatCompletionStream that is returned from NRCreateChatCompletionStream
type ChatCompletionStreamWrapper struct {
	stream *openai.ChatCompletionStream
	txn    *newrelic.Transaction
}

// Wrapper for Recv() method that calls the underlying stream's Recv() method
func (w *ChatCompletionStreamWrapper) Recv() (openai.ChatCompletionStreamResponse, error) {
	response, err := w.stream.Recv()

	if err != nil {
		return response, err
	}

	return response, nil

}

func (w *ChatCompletionStreamWrapper) Close() {
	w.stream.Close()
}

// NRCreateChatCompletionSummary captures the request and response data for a chat completion request and records a custom event in New Relic. It also captures the completion messages
// With a call to NRCreateChatCompletionMessage
func NRCreateChatCompletionSummary(txn *newrelic.Transaction, app *newrelic.Application, cw *ClientWrapper, req openai.ChatCompletionRequest) ChatCompletionResponseWrapper {
	// Get App Config for setting App Name Attribute
	appConfig, configErr := app.Config()
	if !configErr {
		appConfig.AppName = "Unknown"
	}
	uuid := uuid.New()
	spanID := txn.GetTraceMetadata().SpanID
	traceID := txn.GetTraceMetadata().TraceID

	ChatCompletionSummaryData := map[string]interface{}{}
	if !appConfig.AIMonitoring.Streaming.Enabled {
		if reportStreamingDisabled != nil {
			reportStreamingDisabled()
		}
	}
	// Start span
	integrationsupport.AddAgentAttribute(txn, "llm", "", true)
	chatCompletionSpan := txn.StartSegment("Llm/completion/OpenAI/CreateChatCompletion")
	// Track Total time taken for the chat completion or embedding call to complete in milliseconds
	start := time.Now()
	resp, err := cw.Client.CreateChatCompletion(
		context.Background(),
		req,
	)
	duration := time.Since(start).Milliseconds()
	chatCompletionSpan.End()
	if err != nil {
		ChatCompletionSummaryData["error"] = true
		// notice error with custom attributes
		txn.NoticeError(newrelic.Error{
			Message: err.Error(),
			Class:   "OpenAIError",
			Attributes: map[string]interface{}{
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
	ChatCompletionSummaryData["duration"] = duration

	// Response Data
	ChatCompletionSummaryData["response.number_of_messages"] = len(resp.Choices) + len(req.Messages)
	ChatCompletionSummaryData["response.model"] = resp.Model
	ChatCompletionSummaryData["request_id"] = resp.ID
	ChatCompletionSummaryData["response.organization"] = resp.Header().Get("Openai-Organization")

	if len(resp.Choices) > 0 {
		finishReason, err := resp.Choices[0].FinishReason.MarshalJSON()
		if err != nil {
			ChatCompletionSummaryData["error"] = true
			txn.NoticeError(newrelic.Error{
				Message: err.Error(),
				Class:   "OpenAIError",
			})
		} else {
			ChatCompletionSummaryData["response.choices.finish_reason"] = string(finishReason)
		}
	}

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
	ChatCompletionSummaryData["trace_id"] = traceID
	ChatCompletionSummaryData["api_key_last_four_digits"] = cw.LicenseKeyLastFour
	ChatCompletionSummaryData["vendor"] = "OpenAI"
	ChatCompletionSummaryData["ingest_source"] = "Go"
	ChatCompletionSummaryData["appName"] = appConfig.AppName

	// Record any custom attributes if they exist
	ChatCompletionSummaryData = AppendCustomAttributesToEvent(cw, ChatCompletionSummaryData)

	// Record Custom Event
	app.RecordCustomEvent("LlmChatCompletionSummary", ChatCompletionSummaryData)

	// Capture request message
	NRCreateChatCompletionMessageInput(txn, app, req, uuid, cw)
	// Capture completion messages
	NRCreateChatCompletionMessage(txn, app, resp, uuid, cw)
	txn.End()

	return ChatCompletionResponseWrapper{
		ChatCompletionResponse: resp,
		TraceID:                traceID,
	}
}
func NRCreateChatCompletionMessageInput(txn *newrelic.Transaction, app *newrelic.Application, req openai.ChatCompletionRequest, uuid uuid.UUID, cw *ClientWrapper) {
	spanID := txn.GetTraceMetadata().SpanID
	traceID := txn.GetTraceMetadata().TraceID
	appCfg, configErr := app.Config()
	if !configErr {
		appCfg.AppName = "Unknown"
	}
	integrationsupport.AddAgentAttribute(txn, "llm", "", true)
	chatCompletionMessageSpan := txn.StartSegment("Llm/completion/OpenAI/CreateChatCompletionMessage")

	ChatCompletionMessageData := map[string]interface{}{}
	// if the response doesn't have an ID, use the UUID from the summary
	ChatCompletionMessageData["id"] = uuid.String() + "-0"

	// Response Data
	ChatCompletionMessageData["response.model"] = req.Model

	if appCfg.AIMonitoring.RecordContent.Enabled {
		ChatCompletionMessageData["content"] = req.Messages[0].Content
	}

	ChatCompletionMessageData["role"] = req.Messages[0].Role

	// New Relic Attributes
	ChatCompletionMessageData["sequence"] = 0
	ChatCompletionMessageData["vendor"] = "openai"
	ChatCompletionMessageData["ingest_source"] = "go"
	ChatCompletionMessageData["span_id"] = spanID
	ChatCompletionMessageData["trace_id"] = traceID
	contentTokens, contentCounted := app.InvokeLLMTokenCountCallback(req.Model, req.Messages[0].Content)

	if contentCounted {
		ChatCompletionMessageData["token_count"] = contentTokens
	}

	// If custom attributes are set, add them to the data
	ChatCompletionMessageData = AppendCustomAttributesToEvent(cw, ChatCompletionMessageData)
	chatCompletionMessageSpan.End()
	// Record Custom Event for each message
	app.RecordCustomEvent("LlmChatCompletionMessage", ChatCompletionMessageData)

}

// NRCreateChatCompletionMessage captures the completion messages and records a custom event in New Relic for each message
func NRCreateChatCompletionMessage(txn *newrelic.Transaction, app *newrelic.Application, resp openai.ChatCompletionResponse, uuid uuid.UUID, cw *ClientWrapper) {
	spanID := txn.GetTraceMetadata().SpanID
	traceID := txn.GetTraceMetadata().TraceID
	appCfg, configErr := app.Config()
	if !configErr {
		appCfg.AppName = "Unknown"
	}
	integrationsupport.AddAgentAttribute(txn, "llm", "", true)
	chatCompletionMessageSpan := txn.StartSegment("Llm/completion/OpenAI/CreateChatCompletionMessage")
	for i, choice := range resp.Choices {
		ChatCompletionMessageData := map[string]interface{}{}
		// if the response doesn't have an ID, use the UUID from the summary
		if resp.ID == "" {
			ChatCompletionMessageData["id"] = uuid.String() + "-" + fmt.Sprint(i+1)
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
		ChatCompletionMessageData["sequence"] = i + 1
		ChatCompletionMessageData["vendor"] = "openai"
		ChatCompletionMessageData["ingest_source"] = "go"
		ChatCompletionMessageData["span_id"] = spanID
		ChatCompletionMessageData["trace_id"] = traceID
		tokenCount, tokensCounted := TokenCountingHelper(app, choice.Message, resp.Model)
		if tokensCounted {
			ChatCompletionMessageData["token_count"] = tokenCount
		}

		// If custom attributes are set, add them to the data
		ChatCompletionMessageData = AppendCustomAttributesToEvent(cw, ChatCompletionMessageData)

		// Record Custom Event for each message
		app.RecordCustomEvent("LlmChatCompletionMessage", ChatCompletionMessageData)

	}

	chatCompletionMessageSpan.End()
}

func TokenCountingHelper(app *newrelic.Application, message openai.ChatCompletionMessage, model string) (numTokens int, tokensCounted bool) {

	contentTokens, contentCounted := app.InvokeLLMTokenCountCallback(model, message.Content)
	roleTokens, roleCounted := app.InvokeLLMTokenCountCallback(model, message.Role)
	messageTokens, messageCounted := app.InvokeLLMTokenCountCallback(model, message.Name)
	numTokens += contentTokens + roleTokens + messageTokens

	return numTokens, (contentCounted && roleCounted && messageCounted)
}

// NRCreateChatCompletion is a wrapper for the OpenAI CreateChatCompletion method.
// If AI Monitoring is disabled, the wrapped function will still call the OpenAI CreateChatCompletion method and return the response with no New Relic instrumentation
func NRCreateChatCompletion(cw *ClientWrapper, req openai.ChatCompletionRequest, app *newrelic.Application) (ChatCompletionResponseWrapper, error) {
	config, cfgErr := app.Config()
	if !cfgErr {
		config.AppName = "Unknown"
	}

	resp := ChatCompletionResponseWrapper{}
	// If AI Monitoring is disabled, do not start a transaction but still perform the request
	if !config.AIMonitoring.Enabled {
		chatresp, err := cw.Client.CreateChatCompletion(context.Background(), req)
		resp.ChatCompletionResponse = chatresp
		if err != nil {

			return resp, err
		}
		return resp, errAIMonitoringDisabled
	}
	// Start NR Transaction
	txn := app.StartTransaction("OpenAIChatCompletion")
	resp = NRCreateChatCompletionSummary(txn, app, cw, req)

	return resp, nil
}

// NRCreateEmbedding is a wrapper for the OpenAI CreateEmbedding method.
// If AI Monitoring is disabled, the wrapped function will still call the OpenAI CreateEmbedding method and return the response with no New Relic instrumentation
func NRCreateEmbedding(cw *ClientWrapper, req openai.EmbeddingRequest, app *newrelic.Application) (openai.EmbeddingResponse, error) {
	config, cfgErr := app.Config()
	if !cfgErr {
		config.AppName = "Unknown"
	}

	resp := openai.EmbeddingResponse{}

	// If AI Monitoring is disabled, do not start a transaction but still perform the request
	if !config.AIMonitoring.Enabled {
		resp, err := cw.Client.CreateEmbeddings(context.Background(), req)
		if err != nil {

			return resp, err
		}
		return resp, errAIMonitoringDisabled
	}

	// Start NR Transaction
	txn := app.StartTransaction("OpenAIEmbedding")

	spanID := txn.GetTraceMetadata().SpanID
	traceID := txn.GetTraceMetadata().TraceID
	EmbeddingsData := map[string]interface{}{}
	uuid := uuid.New()
	integrationsupport.AddAgentAttribute(txn, "llm", "", true)

	embeddingSpan := txn.StartSegment("Llm/embedding/OpenAI/CreateEmbedding")
	start := time.Now()
	resp, err := cw.Client.CreateEmbeddings(context.Background(), req)
	duration := time.Since(start).Milliseconds()
	embeddingSpan.End()

	if err != nil {
		EmbeddingsData["error"] = true
		txn.NoticeError(newrelic.Error{
			Message: err.Error(),
			Class:   "OpenAIError",
			Attributes: map[string]interface{}{
				"embedding_id": uuid.String(),
			},
		})
	}

	// Request Data
	if config.AIMonitoring.RecordContent.Enabled {
		EmbeddingsData["input"] = GetInput(req.Input)
	}

	EmbeddingsData["request_id"] = resp.Header().Get("X-Request-Id")
	EmbeddingsData["api_key_last_four_digits"] = cw.LicenseKeyLastFour
	EmbeddingsData["request.model"] = string(req.Model)
	EmbeddingsData["duration"] = duration

	// Response Data
	EmbeddingsData["response.model"] = string(resp.Model)
	// cast input as string
	input := GetInput(req.Input).(string)
	tokenCount, tokensCounted := app.InvokeLLMTokenCountCallback(string(resp.Model), input)
	if tokensCounted {
		EmbeddingsData["token_count"] = tokenCount
	}

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
	EmbeddingsData["trace_id"] = traceID

	app.RecordCustomEvent("LlmEmbedding", EmbeddingsData)
	txn.End()
	return resp, nil
}

func NRCreateChatCompletionStream(cw *ClientWrapper, ctx context.Context, req openai.ChatCompletionRequest, app *newrelic.Application) (*ChatCompletionStreamWrapper, error) {
	config, cfgErr := app.Config()
	if !cfgErr {
		config.AppName = "Unknown"
	}
	if !config.AIMonitoring.Streaming.Enabled {
		if reportStreamingDisabled != nil {
			reportStreamingDisabled()
		}
	}
	// If AI Monitoring OR AIMonitoring.Streaming is disabled, do not start a transaction but still perform the request
	if !config.AIMonitoring.Enabled || !config.AIMonitoring.Streaming.Enabled {
		stream, err := cw.Client.CreateChatCompletionStream(ctx, req)
		if err != nil {

			return &ChatCompletionStreamWrapper{stream: stream}, err
		}
		return &ChatCompletionStreamWrapper{stream: stream}, errAIMonitoringDisabled
	}

	txn := app.StartTransaction("OpenAIChatCompletionStream")
	spanID := txn.GetTraceMetadata().SpanID
	traceID := txn.GetTraceMetadata().TraceID
	StreamingData := map[string]interface{}{}
	uuid := uuid.New()
	integrationsupport.AddAgentAttribute(txn, "llm", "", true)
	streamSpan := txn.StartSegment("Llm/completion/OpenAI/stream")
	start := time.Now()
	stream, err := cw.Client.CreateChatCompletionStream(ctx, req)
	duration := time.Since(start).Milliseconds()
	streamSpan.End()

	if err != nil {
		StreamingData["error"] = true
		txn.NoticeError(newrelic.Error{
			Message: err.Error(),
			Class:   "OpenAIError",
		})
		txn.End()
		return nil, err
	}

	// Request Data
	StreamingData["api_key_last_four_digits"] = cw.LicenseKeyLastFour
	StreamingData["request.model"] = string(req.Model)
	StreamingData["request.temperature"] = req.Temperature
	StreamingData["request.max_tokens"] = req.MaxTokens
	StreamingData["model"] = req.Model

	StreamingData["duration"] = duration

	// New Relic Attributes
	StreamingData["id"] = uuid.String()
	StreamingData["span_id"] = spanID
	StreamingData["trace_id"] = traceID
	StreamingData["api_key_last_four_digits"] = cw.LicenseKeyLastFour
	StreamingData["vendor"] = "OpenAI"
	StreamingData["ingest_source"] = "Go"
	StreamingData["appName"] = config.AppName
	app.RecordCustomEvent("LlmChatCompletionSummary", StreamingData)
	txn.End()
	return &ChatCompletionStreamWrapper{stream: stream, txn: txn}, nil

}
