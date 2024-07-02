// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nropenai

import (
	"context"
	"errors"
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

// OpenAIClient is any type that can invoke OpenAI model with a request.
type OpenAIClient interface {
	CreateChatCompletion(ctx context.Context, request openai.ChatCompletionRequest) (response openai.ChatCompletionResponse, err error)
	CreateChatCompletionStream(ctx context.Context, request openai.ChatCompletionRequest) (stream *openai.ChatCompletionStream, err error)
	CreateEmbeddings(ctx context.Context, conv openai.EmbeddingRequestConverter) (res openai.EmbeddingResponse, err error)
}

// Wrapper for OpenAI Configuration
type ConfigWrapper struct {
	Config *openai.ClientConfig
}

// Wrapper for OpenAI Client with Custom Attributes that can be set for all LLM Events
type ClientWrapper struct {
	Client OpenAIClient
	// Set of Custom Attributes that get tied to all LLM Events
	CustomAttributes map[string]interface{}
}

// Wrapper for ChatCompletionResponse that is returned from NRCreateChatCompletion. It also includes the TraceID of the transaction for linking a chat response with it's feedback
type ChatCompletionResponseWrapper struct {
	ChatCompletionResponse openai.ChatCompletionResponse
	TraceID                string
}

// Wrapper for ChatCompletionStream that is returned from NRCreateChatCompletionStream
// Contains attributes that get populated during the streaming process
type ChatCompletionStreamWrapper struct {
	app           *newrelic.Application
	span          *newrelic.Segment // active span
	stream        *openai.ChatCompletionStream
	streamResp    openai.ChatCompletionResponse
	txn           *newrelic.Transaction
	cw            *ClientWrapper
	role          string
	model         string
	responseStr   string
	uuid          string
	finishReason  string
	StreamingData map[string]interface{}
	isRoleAdded   bool
	TraceID       string
	isError       bool
	sequence      int
}

// Default Config
func NRDefaultConfig(authToken string) *ConfigWrapper {
	cfg := openai.DefaultConfig(authToken)
	return &ConfigWrapper{
		Config: &cfg,
	}
}

// Azure Config
func NRDefaultAzureConfig(apiKey, baseURL string) *ConfigWrapper {
	cfg := openai.DefaultAzureConfig(apiKey, baseURL)
	return &ConfigWrapper{
		Config: &cfg,
	}
}

// Call to Create Client Wrapper
func NRNewClient(authToken string) *ClientWrapper {
	client := openai.NewClient(authToken)
	return &ClientWrapper{
		Client: client,
	}
}

// NewClientWithConfig creates new OpenAI API client for specified config.
func NRNewClientWithConfig(config *ConfigWrapper) *ClientWrapper {
	client := openai.NewClientWithConfig(*config.Config)
	return &ClientWrapper{
		Client: client,
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

// Wrapper for OpenAI Streaming Recv() method
// Captures the response messages as they are received in the wrapper
// Once the stream is closed, the Close() method is called and sends the captured
// data to New Relic
func (w *ChatCompletionStreamWrapper) Recv() (openai.ChatCompletionStreamResponse, error) {
	response, err := w.stream.Recv()
	if err != nil {
		return response, err
	}
	if !w.isRoleAdded && (response.Choices[0].Delta.Role == "assistant" || response.Choices[0].Delta.Role == "user" || response.Choices[0].Delta.Role == "system") {
		w.isRoleAdded = true
		w.role = response.Choices[0].Delta.Role

	}
	if response.Choices[0].FinishReason != "stop" {
		w.responseStr += response.Choices[0].Delta.Content
		w.streamResp.ID = response.ID
		w.streamResp.Model = response.Model
		w.model = response.Model
	}
	finishReason, finishReasonErr := response.Choices[0].FinishReason.MarshalJSON()
	if finishReasonErr != nil {
		w.isError = true
	}
	w.finishReason = string(finishReason)

	return response, nil

}

// Close the stream and send the event to New Relic
func (w *ChatCompletionStreamWrapper) Close() {
	w.StreamingData["response.model"] = w.model
	NRCreateChatCompletionMessageStream(w.app, uuid.MustParse(w.uuid), w, w.cw, w.sequence)
	if w.isError {
		w.StreamingData["error"] = true
	} else {
		w.StreamingData["response.choices.finish_reason"] = w.finishReason
	}

	w.span.End()
	w.app.RecordCustomEvent("LlmChatCompletionSummary", w.StreamingData)

	w.txn.End()
	w.stream.Close()
}

// NRCreateChatCompletionSummary captures the request data for a chat completion request
// A new segment is created for the chat completion request, and the response data is timed and captured
// Custom attributes are added to the event if they exist from client.AddCustomAttributes()
// After closing out the custom event for the chat completion summary, the function then calls
// NRCreateChatCompletionMessageInput/NRCreateChatCompletionMessage to capture the request messages
func NRCreateChatCompletionSummary(txn *newrelic.Transaction, app *newrelic.Application, cw *ClientWrapper, req openai.ChatCompletionRequest) ChatCompletionResponseWrapper {
	// Start span
	txn.AddAttribute("llm", true)

	chatCompletionSpan := txn.StartSegment("Llm/completion/OpenAI/CreateChatCompletion")
	// Track Total time taken for the chat completion or embedding call to complete in milliseconds

	// Get App Config for setting App Name Attribute
	appConfig, _ := app.Config()

	uuid := uuid.New()
	spanID := txn.GetTraceMetadata().SpanID
	traceID := txn.GetTraceMetadata().TraceID

	ChatCompletionSummaryData := map[string]interface{}{}
	if !appConfig.AIMonitoring.Streaming.Enabled {
		if reportStreamingDisabled != nil {
			reportStreamingDisabled()
		}
	}
	start := time.Now()
	resp, err := cw.Client.CreateChatCompletion(
		context.Background(),
		req,
	)
	duration := time.Since(start).Milliseconds()
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
			s := string(finishReason)
			if len(s) > 0 && s[0] == '"' {
				s = s[1:]
			}
			if len(s) > 0 && s[len(s)-1] == '"' {
				s = s[:len(s)-1]
			}

			// strip quotes from the finish reason before setting it
			ChatCompletionSummaryData["response.choices.finish_reason"] = s
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
	ChatCompletionSummaryData["vendor"] = "openai"
	ChatCompletionSummaryData["ingest_source"] = "Go"
	// Record any custom attributes if they exist
	ChatCompletionSummaryData = AppendCustomAttributesToEvent(cw, ChatCompletionSummaryData)

	// Record Custom Event
	app.RecordCustomEvent("LlmChatCompletionSummary", ChatCompletionSummaryData)
	// Capture request message, returns a sequence of the messages already sent in the request. We will use that during the response message counting
	sequence := NRCreateChatCompletionMessageInput(txn, app, req, uuid, cw)
	// Capture completion messages
	NRCreateChatCompletionMessage(txn, app, resp, uuid, cw, sequence, req)
	chatCompletionSpan.End()

	txn.End()

	return ChatCompletionResponseWrapper{
		ChatCompletionResponse: resp,
		TraceID:                traceID,
	}
}

// Captures initial request messages and records a custom event in New Relic for each message
// similarly to NRCreateChatCompletionMessage, but only for the request messages
// Returns the sequence of the messages sent in the request
// which is used to calculate the sequence in the response messages
func NRCreateChatCompletionMessageInput(txn *newrelic.Transaction, app *newrelic.Application, req openai.ChatCompletionRequest, inputuuid uuid.UUID, cw *ClientWrapper) int {
	sequence := 0
	for i, message := range req.Messages {
		spanID := txn.GetTraceMetadata().SpanID
		traceID := txn.GetTraceMetadata().TraceID

		appCfg, _ := app.Config()
		newUUID := uuid.New()
		newID := newUUID.String()
		integrationsupport.AddAgentAttribute(txn, "llm", "", true)

		ChatCompletionMessageData := map[string]interface{}{}
		// if the response doesn't have an ID, use the UUID from the summary
		ChatCompletionMessageData["id"] = newID

		// Response Data
		ChatCompletionMessageData["response.model"] = req.Model

		if appCfg.AIMonitoring.RecordContent.Enabled {
			ChatCompletionMessageData["content"] = message.Content
		}

		ChatCompletionMessageData["role"] = message.Role
		ChatCompletionMessageData["completion_id"] = inputuuid.String()

		// New Relic Attributes
		ChatCompletionMessageData["sequence"] = i
		ChatCompletionMessageData["vendor"] = "openai"
		ChatCompletionMessageData["ingest_source"] = "Go"
		ChatCompletionMessageData["span_id"] = spanID
		ChatCompletionMessageData["trace_id"] = traceID
		contentTokens, contentCounted := app.InvokeLLMTokenCountCallback(req.Model, message.Content)

		if contentCounted && app.HasLLMTokenCountCallback() {
			ChatCompletionMessageData["token_count"] = contentTokens
		}

		// If custom attributes are set, add them to the data
		ChatCompletionMessageData = AppendCustomAttributesToEvent(cw, ChatCompletionMessageData)
		// Record Custom Event for each message
		app.RecordCustomEvent("LlmChatCompletionMessage", ChatCompletionMessageData)
		sequence = i
	}
	return sequence

}

// NRCreateChatCompletionMessage captures the completion response messages and records a custom event
// in New Relic for each message. The completion response messages are the responses from the model
// after the request messages have been sent and logged in NRCreateChatCompletionMessageInput.
// The sequence of the messages is calculated by logging each of the request messages first, then
// incrementing the sequence for each response message.
// The token count is calculated for each message and added to the custom event if the token count callback is set
// If not, no token count is added to the custom event
func NRCreateChatCompletionMessage(txn *newrelic.Transaction, app *newrelic.Application, resp openai.ChatCompletionResponse, uuid uuid.UUID, cw *ClientWrapper, sequence int, req openai.ChatCompletionRequest) {
	spanID := txn.GetTraceMetadata().SpanID
	traceID := txn.GetTraceMetadata().TraceID
	appCfg, _ := app.Config()

	integrationsupport.AddAgentAttribute(txn, "llm", "", true)
	sequence += 1
	for i, choice := range resp.Choices {
		ChatCompletionMessageData := map[string]interface{}{}
		// if the response doesn't have an ID, use the UUID from the summary
		if resp.ID == "" {
			ChatCompletionMessageData["id"] = uuid.String()
		} else {
			ChatCompletionMessageData["id"] = resp.ID
		}

		// Request Data
		ChatCompletionMessageData["request.model"] = req.Model

		// Response Data
		ChatCompletionMessageData["response.model"] = resp.Model

		if appCfg.AIMonitoring.RecordContent.Enabled {
			ChatCompletionMessageData["content"] = choice.Message.Content
		}

		ChatCompletionMessageData["completion_id"] = uuid.String()
		ChatCompletionMessageData["role"] = choice.Message.Role

		// Request Headers
		ChatCompletionMessageData["request_id"] = resp.Header().Get("X-Request-Id")

		// New Relic Attributes
		ChatCompletionMessageData["is_response"] = true
		ChatCompletionMessageData["sequence"] = sequence + i
		ChatCompletionMessageData["vendor"] = "openai"
		ChatCompletionMessageData["ingest_source"] = "Go"
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
}

// NRCreateChatCompletionMessageStream is identical to NRCreateChatCompletionMessage, but for streaming responses.
// Gets invoked only when the stream is closed
func NRCreateChatCompletionMessageStream(app *newrelic.Application, uuid uuid.UUID, sw *ChatCompletionStreamWrapper, cw *ClientWrapper, sequence int) {

	spanID := sw.txn.GetTraceMetadata().SpanID
	traceID := sw.txn.GetTraceMetadata().TraceID

	appCfg, _ := app.Config()

	integrationsupport.AddAgentAttribute(sw.txn, "llm", "", true)

	ChatCompletionMessageData := map[string]interface{}{}
	// if the response doesn't have an ID, use the UUID from the summary

	ChatCompletionMessageData["id"] = sw.streamResp.ID

	// Response Data
	ChatCompletionMessageData["request.model"] = sw.model

	if appCfg.AIMonitoring.RecordContent.Enabled {
		ChatCompletionMessageData["content"] = sw.responseStr
	}

	ChatCompletionMessageData["role"] = sw.role
	ChatCompletionMessageData["is_response"] = true

	// New Relic Attributes
	ChatCompletionMessageData["sequence"] = sequence + 1
	ChatCompletionMessageData["vendor"] = "openai"
	ChatCompletionMessageData["ingest_source"] = "Go"
	ChatCompletionMessageData["completion_id"] = uuid.String()
	ChatCompletionMessageData["span_id"] = spanID
	ChatCompletionMessageData["trace_id"] = traceID
	tmpMessage := openai.ChatCompletionMessage{
		Content: sw.responseStr,
		Role:    sw.role,
		// Name is not provided in the stream response, so we don't include it in token counting
		Name: "",
	}
	tokenCount, tokensCounted := TokenCountingHelper(app, tmpMessage, sw.model)
	if tokensCounted {
		ChatCompletionMessageData["token_count"] = tokenCount
	}

	// If custom attributes are set, add them to the data
	ChatCompletionMessageData = AppendCustomAttributesToEvent(cw, ChatCompletionMessageData)
	// Record Custom Event for each message
	app.RecordCustomEvent("LlmChatCompletionMessage", ChatCompletionMessageData)

}

// Calculates tokens using the LLmTokenCountCallback
// In order to calculate total tokens of a message, we need to factor in the Content, Role, and Name (if it exists)
func TokenCountingHelper(app *newrelic.Application, message openai.ChatCompletionMessage, model string) (numTokens int, tokensCounted bool) {
	contentTokens, contentCounted := app.InvokeLLMTokenCountCallback(model, message.Content)
	roleTokens, roleCounted := app.InvokeLLMTokenCountCallback(model, message.Role)
	var messageTokens int
	if message.Name != "" {
		messageTokens, _ = app.InvokeLLMTokenCountCallback(model, message.Name)

	}
	numTokens += contentTokens + roleTokens + messageTokens

	return numTokens, (contentCounted && roleCounted)
}

// Similar to NRCreateChatCompletionSummary, but for streaming responses
// Returns a custom wrapper with a stream that can be used to receive messages
// Example Usage:
/*
	ctx := context.Background()
	stream, err := nropenai.NRCreateChatCompletionStream(client, ctx, req, app)
	if err != nil {
		panic(err)
	}
	for {
		var response openai.ChatCompletionStreamResponse
		response, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println("\nStream finished")
			break
		}
		if err != nil {
			fmt.Printf("\nStream error: %v\n", err)
			return
		}
		fmt.Printf(response.Choices[0].Delta.Content)
	}
	stream.Close()
*/
// It is important to call stream.Close() after the stream has been used, as it will close the stream and send the event to New Relic.
// Additionally, custom attributes can be added to the client using client.AddCustomAttributes(map[string]interface{}) just like in NRCreateChatCompletionSummary
func NRCreateChatCompletionStream(cw *ClientWrapper, ctx context.Context, req openai.ChatCompletionRequest, app *newrelic.Application) (*ChatCompletionStreamWrapper, error) {
	txn := app.StartTransaction("OpenAIChatCompletionStream")

	config, _ := app.Config()

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

	streamSpan := txn.StartSegment("Llm/completion/OpenAI/CreateChatCompletion")

	spanID := txn.GetTraceMetadata().SpanID
	traceID := txn.GetTraceMetadata().TraceID
	StreamingData := map[string]interface{}{}
	uuid := uuid.New()
	integrationsupport.AddAgentAttribute(txn, "llm", "", true)
	start := time.Now()
	stream, err := cw.Client.CreateChatCompletionStream(ctx, req)
	duration := time.Since(start).Milliseconds()

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
	StreamingData["request.model"] = string(req.Model)
	StreamingData["request.temperature"] = req.Temperature
	StreamingData["request.max_tokens"] = req.MaxTokens
	StreamingData["model"] = req.Model

	StreamingData["duration"] = duration

	// New Relic Attributes
	StreamingData["id"] = uuid.String()
	StreamingData["span_id"] = spanID
	StreamingData["trace_id"] = traceID
	StreamingData["vendor"] = "openai"
	StreamingData["ingest_source"] = "Go"

	sequence := NRCreateChatCompletionMessageInput(txn, app, req, uuid, cw)
	return &ChatCompletionStreamWrapper{
		app:           app,
		stream:        stream,
		txn:           txn,
		span:          streamSpan,
		uuid:          uuid.String(),
		cw:            cw,
		StreamingData: StreamingData,
		TraceID:       traceID,
		sequence:      sequence}, nil

}

// NRCreateChatCompletion is a wrapper for the OpenAI CreateChatCompletion method.
// If AI Monitoring is disabled, the wrapped function will still call the OpenAI CreateChatCompletion method
// and return the response with no New Relic instrumentation
// Calls NRCreateChatCompletionSummary to capture the request data and response data
// Returns a ChatCompletionResponseWrapper with the response and the TraceID of the transaction
// The trace ID is used to link the chat response with its feedback, with a call to SendFeedback()
// Otherwise, the response is the same as the OpenAI CreateChatCompletion method. It can be accessed
// by calling resp.ChatCompletionResponse
func NRCreateChatCompletion(cw *ClientWrapper, req openai.ChatCompletionRequest, app *newrelic.Application) (ChatCompletionResponseWrapper, error) {
	config, _ := app.Config()

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
	config, _ := app.Config()

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
	embeddingSpan := txn.StartSegment("Llm/embedding/OpenAI/CreateEmbedding")

	spanID := txn.GetTraceMetadata().SpanID
	traceID := txn.GetTraceMetadata().TraceID
	EmbeddingsData := map[string]interface{}{}
	uuid := uuid.New()
	integrationsupport.AddAgentAttribute(txn, "llm", "", true)

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
	EmbeddingsData["request.model"] = string(req.Model)
	EmbeddingsData["duration"] = duration

	// Response Data
	EmbeddingsData["response.model"] = string(resp.Model)
	// cast input as string
	input := GetInput(req.Input).(string)
	tokenCount, tokensCounted := app.InvokeLLMTokenCountCallback(string(resp.Model), input)

	if tokensCounted && app.HasLLMTokenCountCallback() {
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
	EmbeddingsData["vendor"] = "openai"
	EmbeddingsData["ingest_source"] = "Go"
	EmbeddingsData["span_id"] = spanID
	EmbeddingsData["trace_id"] = traceID

	app.RecordCustomEvent("LlmEmbedding", EmbeddingsData)
	txn.End()
	return resp, nil
}
