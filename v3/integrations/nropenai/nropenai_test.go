package nropenai

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sashabaranov/go-openai"
)

type MockOpenAIClient struct {
	MockCreateChatCompletionResp   openai.ChatCompletionResponse
	MockCreateEmbeddingsResp       openai.EmbeddingResponse
	MockCreateChatCompletionStream *openai.ChatCompletionStream
	MockCreateChatCompletionErr    error
}

// Mock CreateChatCompletion function that returns a mock response
func (m *MockOpenAIClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {

	MockResponse := openai.ChatCompletionResponse{
		ID:                "chatcmpl-123",
		Object:            "chat.completion",
		Created:           1677652288,
		Model:             openai.GPT3Dot5Turbo,
		SystemFingerprint: "fp_44709d6fcb",
		Usage: openai.Usage{
			PromptTokens:     9,
			CompletionTokens: 12,
			TotalTokens:      21,
		},
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "\n\nHello there, how may I assist you today?",
				},
			},
		},
	}
	hdrs := http.Header{}
	hdrs.Add("X-Request-Id", "chatcmpl-123")
	hdrs.Add("ratelimit-limit-tokens", "100")
	hdrs.Add("Openai-Version", "2020-10-01")
	hdrs.Add("X-Ratelimit-Limit-Requests", "10000")
	hdrs.Add("X-Ratelimit-Limit-Tokens", "100")
	hdrs.Add("X-Ratelimit-Reset-Tokens", "100")
	hdrs.Add("X-Ratelimit-Reset-Requests", "10000")
	hdrs.Add("X-Ratelimit-Remaining-Tokens", "100")
	hdrs.Add("X-Ratelimit-Remaining-Requests", "10000")
	hdrs.Add("Openai-Organization", "user-123")

	if req.Messages[0].Content == "testError" {
		mockRespErr := openai.ChatCompletionResponse{}
		hdrs.Add("Status", "404")
		hdrs.Add("Error-Code", "404")
		mockRespErr.SetHeader(hdrs)
		return mockRespErr, errors.New("test error")
	}
	MockResponse.SetHeader(hdrs)

	return MockResponse, m.MockCreateChatCompletionErr
}

func (m *MockOpenAIClient) CreateEmbeddings(ctx context.Context, conv openai.EmbeddingRequestConverter) (res openai.EmbeddingResponse, err error) {
	MockResponse := openai.EmbeddingResponse{
		Model: openai.AdaEmbeddingV2,
		Usage: openai.Usage{
			PromptTokens:     9,
			CompletionTokens: 12,
			TotalTokens:      21,
		},
		Data: []openai.Embedding{
			{
				Embedding: []float32{0.1, 0.2, 0.3},
			},
		},
	}
	hdrs := http.Header{}
	hdrs.Add("X-Request-Id", "chatcmpl-123")
	hdrs.Add("ratelimit-limit-tokens", "100")
	hdrs.Add("Openai-Version", "2020-10-01")
	hdrs.Add("X-Ratelimit-Limit-Requests", "10000")
	hdrs.Add("X-Ratelimit-Limit-Tokens", "100")
	hdrs.Add("X-Ratelimit-Reset-Tokens", "100")
	hdrs.Add("X-Ratelimit-Reset-Requests", "10000")
	hdrs.Add("X-Ratelimit-Remaining-Tokens", "100")
	hdrs.Add("X-Ratelimit-Remaining-Requests", "10000")
	hdrs.Add("Openai-Organization", "user-123")
	cv := conv.Convert()
	if cv.Input == "testError" {
		mockRespErr := openai.EmbeddingResponse{}
		hdrs.Add("Status", "404")
		hdrs.Add("Error-Code", "404")
		mockRespErr.SetHeader(hdrs)
		return mockRespErr, errors.New("test error")
	}

	MockResponse.SetHeader(hdrs)

	return MockResponse, m.MockCreateChatCompletionErr
}

func (m *MockOpenAIClient) CreateChatCompletionStream(ctx context.Context, request openai.ChatCompletionRequest) (stream *openai.ChatCompletionStream, err error) {
	if request.Messages[0].Content == "testError" {
		return m.MockCreateChatCompletionStream, errors.New("test error")
	}
	return m.MockCreateChatCompletionStream, m.MockCreateChatCompletionErr
}

func TestFormatAPIKey(t *testing.T) {
	dummyAPIKey := "sk-12345678900abcdefghijklmnop"
	formattedKey := FormatAPIKey(dummyAPIKey)
	if formattedKey != "sk-mnop" {
		t.Errorf("Formatted API key is incorrect: expected: %s actual: %s", "sk-mnop", formattedKey)

	}
}
func TestDefaultConfig(t *testing.T) {
	dummyAPIKey := "sk-12345678900abcdefghijklmnop"
	cfg := NRDefaultConfig(dummyAPIKey)
	// Default Values
	if cfg.LicenseKeyLastFour != "sk-mnop" {
		t.Errorf("API Key is incorrect: expected: %s actual: %s", "sk-mnop", cfg.LicenseKeyLastFour)
	}
	if cfg.Config.OrgID != "" {
		t.Errorf("OrgID is incorrect: expected: %s actual: %s", "", cfg.Config.OrgID)
	}
	// Default Value set by openai package
	if cfg.Config.APIType != openai.APITypeOpenAI {
		t.Errorf("API Type is incorrect: expected: %s actual: %s", openai.APITypeOpenAI, cfg.Config.APIType)
	}
}

func TestDefaultConfigAzure(t *testing.T) {
	dummyAPIKey := "sk-12345678900abcdefghijklmnop"
	baseURL := "https://azure-base-url.com"
	cfg := NRDefaultAzureConfig(dummyAPIKey, baseURL)
	// Default Values
	if cfg.LicenseKeyLastFour != "sk-mnop" {
		t.Errorf("API Key is incorrect: expected: %s actual: %s", "sk-mnop", cfg.LicenseKeyLastFour)
	}
	if cfg.Config.BaseURL != baseURL {
		t.Errorf("baseURL is incorrect: expected: %s actual: %s", baseURL, cfg.Config.BaseURL)
	}
	// Default Value set by openai package
	if cfg.Config.APIType != openai.APITypeAzure {
		t.Errorf("API Type is incorrect: expected: %s actual: %s", openai.APITypeAzure, cfg.Config.APIType)
	}
}

func TestNRNewClient(t *testing.T) {
	dummyAPIKey := "sk-12345678900abcdefghijklmnop"
	client := NRNewClient(dummyAPIKey)
	if client.LicenseKeyLastFour != "sk-mnop" {
		t.Errorf("API Key is incorrect: expected: %s actual: %s", "sk-mnop", client.LicenseKeyLastFour)
	}
}

func TestNRNewClientWithConfigs(t *testing.T) {
	// Regular Config
	dummyAPIKey := "sk-12345678900abcdefghijklmnop"
	cfg := NRDefaultConfig(dummyAPIKey)
	client := NRNewClientWithConfig(cfg)
	if client.LicenseKeyLastFour != "sk-mnop" {
		t.Errorf("API Key is incorrect: expected: %s actual: %s", "sk-mnop", client.LicenseKeyLastFour)
	}
	// Azure Config
	baseURL := "https://azure-base-url.com"
	azureCfg := NRDefaultAzureConfig(dummyAPIKey, baseURL)
	azureClient := NRNewClientWithConfig(azureCfg)
	if azureClient.LicenseKeyLastFour != "sk-mnop" {
		t.Errorf("API Key is incorrect: expected: %s actual: %s", "sk-mnop", azureClient.LicenseKeyLastFour)
	}
	if azureCfg.Config.BaseURL != baseURL {
		t.Errorf("baseURL is incorrect: expected: %s actual: %s", baseURL, azureCfg.Config.BaseURL)
	}
	// Default Value set by openai package
	if azureCfg.Config.APIType != openai.APITypeAzure {
		t.Errorf("API Type is incorrect: expected: %s actual: %s", openai.APITypeAzure, azureCfg.Config.APIType)
	}
}

func TestAddCustomAttributes(t *testing.T) {
	client := NRNewClient("sk-12345678900abcdefghijklmnop")
	client.AddCustomAttributes(map[string]interface{}{
		"llm.foo": "bar",
	})
	if client.CustomAttributes["llm.foo"] != "bar" {
		t.Errorf("Custom attribute is incorrect: expected: %s actual: %s", "bar", client.CustomAttributes["llm.foo"])
	}
}
func TestAddCustomAttributesIncorrectPrefix(t *testing.T) {
	client := NRNewClient("sk-12345678900abcdefghijklmnop")
	client.AddCustomAttributes(map[string]interface{}{
		"msdwmdoawd.foo": "bar",
	})
	if len(client.CustomAttributes) != 0 {
		t.Errorf("Custom attribute is incorrect: expected: %d actual: %d", 0, len(client.CustomAttributes))
	}
}

func TestNRCreateChatCompletion(t *testing.T) {
	mockClient := &MockOpenAIClient{}
	cw := &ClientWrapper{
		Client:             mockClient,
		LicenseKeyLastFour: "sk-mnop",
	}
	req := openai.ChatCompletionRequest{
		Model:       openai.GPT3Dot5Turbo,
		Temperature: 0,
		MaxTokens:   150,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "What is 8*5",
			},
		},
	}
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true))
	resp, err := NRCreateChatCompletion(cw, req, app.Application)
	if err != nil {
		t.Error(err)
	}
	if resp.ChatCompletionResponse.Choices[0].Message.Content != "\n\nHello there, how may I assist you today?" {
		t.Errorf("Chat completion response is incorrect: expected: %s actual: %s", "\n\nHello there, how may I assist you today?", resp.ChatCompletionResponse.Choices[0].Message.Content)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionSummary",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"ingest_source":                    "Go",
				"vendor":                           "OpenAI",
				"model":                            "gpt-3.5-turbo",
				"id":                               internal.MatchAnything,
				"transaction_id":                   internal.MatchAnything,
				"trace_id":                         internal.MatchAnything,
				"span_id":                          internal.MatchAnything,
				"appName":                          "my app",
				"duration":                         0,
				"response.choices.finish_reason":   internal.MatchAnything,
				"request.temperature":              0,
				"api_key_last_four_digits":         "sk-mnop",
				"request_id":                       "chatcmpl-123",
				"request.model":                    "gpt-3.5-turbo",
				"request.max_tokens":               150,
				"response.number_of_messages":      1,
				"response.headers.llmVersion":      "2020-10-01",
				"response.organization":            "user-123",
				"response.usage.completion_tokens": 12,
				"response.model":                   "gpt-3.5-turbo",
				"response.usage.total_tokens":      21,
				"response.usage.prompt_tokens":     9,
				"response.headers.ratelimitRemainingTokens":   "100",
				"response.headers.ratelimitRemainingRequests": "10000",
				"response.headers.ratelimitResetTokens":       "100",
				"response.headers.ratelimitResetRequests":     "10000",
				"response.headers.ratelimitLimitTokens":       "100",
				"response.headers.ratelimitLimitRequests":     "10000",
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionMessage",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"trace_id":       internal.MatchAnything,
				"transaction_id": internal.MatchAnything,
				"span_id":        internal.MatchAnything,
				"id":             "chatcmpl-123",
				"sequence":       0,
				"role":           "assistant",
				"content":        "\n\nHello there, how may I assist you today?",
				"request_id":     "chatcmpl-123",
				"vendor":         "openai",
				"ingest_source":  "go",
				"response.model": "gpt-3.5-turbo",
			},
			AgentAttributes: map[string]interface{}{},
		},
	})

}

func TestNRCreateChatCompletionAIMonitoringNotEnabled(t *testing.T) {
	mockClient := &MockOpenAIClient{}
	cw := &ClientWrapper{
		Client:             mockClient,
		LicenseKeyLastFour: "sk-mnop",
	}
	req := openai.ChatCompletionRequest{
		Model:       openai.GPT3Dot5Turbo,
		Temperature: 0,
		MaxTokens:   150,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "What is 8*5",
			},
		},
	}
	app := integrationsupport.NewTestApp(nil)
	resp, err := NRCreateChatCompletion(cw, req, app.Application)
	if err != errAIMonitoringDisabled {
		t.Error(err)
	}
	// If AI Monitoring is disabled, no events should be sent, but a response from OpenAI should still be returned
	if resp.ChatCompletionResponse.Choices[0].Message.Content != "\n\nHello there, how may I assist you today?" {
		t.Errorf("Chat completion response is incorrect: expected: %s actual: %s", "\n\nHello there, how may I assist you today?", resp.ChatCompletionResponse.Choices[0].Message.Content)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{})

}

func TestNRCreateChatCompletionError(t *testing.T) {
	mockClient := &MockOpenAIClient{}
	cw := &ClientWrapper{
		Client:             mockClient,
		LicenseKeyLastFour: "sk-mnop",
	}
	req := openai.ChatCompletionRequest{
		Model:       openai.GPT3Dot5Turbo,
		Temperature: 0,
		MaxTokens:   150,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "testError",
			},
		},
	}
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true))
	_, err := NRCreateChatCompletion(cw, req, app.Application)
	if err != nil {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionSummary",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"error":                            true,
				"ingest_source":                    "Go",
				"vendor":                           "OpenAI",
				"model":                            "gpt-3.5-turbo",
				"id":                               internal.MatchAnything,
				"transaction_id":                   internal.MatchAnything,
				"trace_id":                         internal.MatchAnything,
				"span_id":                          internal.MatchAnything,
				"appName":                          "my app",
				"duration":                         0,
				"request.temperature":              0,
				"api_key_last_four_digits":         "sk-mnop",
				"request_id":                       "",
				"request.model":                    "gpt-3.5-turbo",
				"request.max_tokens":               150,
				"response.number_of_messages":      0,
				"response.headers.llmVersion":      "2020-10-01",
				"response.organization":            "user-123",
				"response.usage.completion_tokens": 0,
				"response.model":                   "",
				"response.usage.total_tokens":      0,
				"response.usage.prompt_tokens":     0,
				"response.headers.ratelimitRemainingTokens":   "100",
				"response.headers.ratelimitRemainingRequests": "10000",
				"response.headers.ratelimitResetTokens":       "100",
				"response.headers.ratelimitResetRequests":     "10000",
				"response.headers.ratelimitLimitTokens":       "100",
				"response.headers.ratelimitLimitRequests":     "10000",
			},
		},
	})
	app.ExpectErrorEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":            "TransactionError",
				"transactionName": "OtherTransaction/Go/OpenAIChatCompletion",
				"guid":            internal.MatchAnything,
				"priority":        internal.MatchAnything,
				"sampled":         internal.MatchAnything,
				"traceId":         internal.MatchAnything,
				"error.class":     "OpenAIError",
				"error.message":   "test error",
			},
			UserAttributes: map[string]interface{}{
				"error.code":    "404",
				"http.status":   "404",
				"completion_id": internal.MatchAnything,
			},
		}})
}
func TestNRCreateEmbedding(t *testing.T) {
	mockClient := &MockOpenAIClient{}
	cw := &ClientWrapper{
		Client:             mockClient,
		LicenseKeyLastFour: "sk-mnop",
	}
	embeddingReq := openai.EmbeddingRequest{
		Input: []string{
			"The food was delicious and the waiter",
			"Other examples of embedding request",
		},
		Model:          openai.AdaEmbeddingV2,
		EncodingFormat: openai.EmbeddingEncodingFormatFloat,
	}

	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true))

	_, err := NRCreateEmbedding(cw, embeddingReq, app.Application)
	if err != nil {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmEmbedding",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"ingest_source":                "Go",
				"vendor":                       "OpenAI",
				"id":                           internal.MatchAnything,
				"transaction_id":               internal.MatchAnything,
				"trace_id":                     internal.MatchAnything,
				"span_id":                      internal.MatchAnything,
				"duration":                     0,
				"api_key_last_four_digits":     "sk-mnop",
				"request.model":                "text-embedding-ada-002",
				"response.headers.llmVersion":  "2020-10-01",
				"response.organization":        "user-123",
				"response.model":               "text-embedding-ada-002",
				"response.usage.total_tokens":  21,
				"response.usage.prompt_tokens": 9,
				"input":                        "The food was delicious and the waiter",
				"response.headers.ratelimitRemainingTokens":   "100",
				"response.headers.ratelimitRemainingRequests": "10000",
				"response.headers.ratelimitResetTokens":       "100",
				"response.headers.ratelimitResetRequests":     "10000",
				"response.headers.ratelimitLimitTokens":       "100",
				"response.headers.ratelimitLimitRequests":     "10000",
			},
		},
	})

}

func TestNRCreateEmbeddingAIMonitoringNotEnabled(t *testing.T) {
	mockClient := &MockOpenAIClient{}
	cw := &ClientWrapper{
		Client:             mockClient,
		LicenseKeyLastFour: "sk-mnop",
	}
	embeddingReq := openai.EmbeddingRequest{
		Input: []string{
			"The food was delicious and the waiter",
			"Other examples of embedding request",
		},
		Model:          openai.AdaEmbeddingV2,
		EncodingFormat: openai.EmbeddingEncodingFormatFloat,
	}

	app := integrationsupport.NewTestApp(nil)

	resp, err := NRCreateEmbedding(cw, embeddingReq, app.Application)
	if err != errAIMonitoringDisabled {
		t.Error(err)
	}
	// If AI Monitoring is disabled, no events should be sent, but a response from OpenAI should still be returned
	app.ExpectCustomEvents(t, []internal.WantEvent{})
	if resp.Data[0].Embedding[0] != 0.1 {
		t.Errorf("Embedding response is incorrect: expected: %f actual: %f", 0.1, resp.Data[0].Embedding[0])
	}

}
func TestNRCreateEmbeddingError(t *testing.T) {
	mockClient := &MockOpenAIClient{}
	cw := &ClientWrapper{
		Client:             mockClient,
		LicenseKeyLastFour: "sk-mnop",
	}
	embeddingReq := openai.EmbeddingRequest{
		Input:          "testError",
		Model:          openai.AdaEmbeddingV2,
		EncodingFormat: openai.EmbeddingEncodingFormatFloat,
	}

	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true))

	_, err := NRCreateEmbedding(cw, embeddingReq, app.Application)
	if err != nil {
		t.Error(err)
	}

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmEmbedding",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"ingest_source":                "Go",
				"vendor":                       "OpenAI",
				"id":                           internal.MatchAnything,
				"transaction_id":               internal.MatchAnything,
				"trace_id":                     internal.MatchAnything,
				"span_id":                      internal.MatchAnything,
				"duration":                     0,
				"api_key_last_four_digits":     "sk-mnop",
				"request.model":                "text-embedding-ada-002",
				"response.headers.llmVersion":  "2020-10-01",
				"response.organization":        "user-123",
				"error":                        true,
				"response.model":               "",
				"response.usage.total_tokens":  0,
				"response.usage.prompt_tokens": 0,
				"input":                        "testError",
				"response.headers.ratelimitRemainingTokens":   "100",
				"response.headers.ratelimitRemainingRequests": "10000",
				"response.headers.ratelimitResetTokens":       "100",
				"response.headers.ratelimitResetRequests":     "10000",
				"response.headers.ratelimitLimitTokens":       "100",
				"response.headers.ratelimitLimitRequests":     "10000",
			},
		},
	})

	app.ExpectErrorEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":            "TransactionError",
				"transactionName": "OtherTransaction/Go/OpenAIEmbedding",
				"guid":            internal.MatchAnything,
				"priority":        internal.MatchAnything,
				"sampled":         internal.MatchAnything,
				"traceId":         internal.MatchAnything,
				"error.class":     "OpenAIError",
				"error.message":   "test error",
			},
			UserAttributes: map[string]interface{}{
				"error.code":   "404",
				"http.status":  "404",
				"embedding_id": internal.MatchAnything,
			},
		}})
}

func TestNRCreateStream(t *testing.T) {
	mockClient := &MockOpenAIClient{}
	cw := &ClientWrapper{
		Client:             mockClient,
		LicenseKeyLastFour: "sk-mnop",
	}
	req := openai.ChatCompletionRequest{
		Model:       openai.GPT3Dot5Turbo,
		Temperature: 0,
		MaxTokens:   1500,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Say this is a test",
			},
		},
		Stream: true,
	}
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true))
	_, err := NRCreateChatCompletionStream(cw, context.Background(), req, app.Application)
	if err != nil {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionSummary",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"ingest_source":            "Go",
				"vendor":                   "OpenAI",
				"model":                    "gpt-3.5-turbo",
				"id":                       internal.MatchAnything,
				"transaction_id":           internal.MatchAnything,
				"trace_id":                 internal.MatchAnything,
				"span_id":                  internal.MatchAnything,
				"appName":                  "my app",
				"duration":                 0,
				"request.temperature":      0,
				"api_key_last_four_digits": "sk-mnop",
				"request.max_tokens":       1500,
				"request.model":            "gpt-3.5-turbo",
			},
		},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "Transaction",
				"name":      "OtherTransaction/Go/OpenAIChatCompletionStream",
				"timestamp": internal.MatchAnything,
				"traceId":   internal.MatchAnything,
				"priority":  internal.MatchAnything,
				"sampled":   internal.MatchAnything,
				"guid":      internal.MatchAnything,
			},
		},
	})
}

func TestNRCreateStreamAIMonitoringNotEnabled(t *testing.T) {
	mockClient := &MockOpenAIClient{}
	cw := &ClientWrapper{
		Client:             mockClient,
		LicenseKeyLastFour: "sk-mnop",
	}
	req := openai.ChatCompletionRequest{
		Model:       openai.GPT3Dot5Turbo,
		Temperature: 0,
		MaxTokens:   1500,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Say this is a test",
			},
		},
		Stream: true,
	}
	app := integrationsupport.NewTestApp(nil)
	_, err := NRCreateChatCompletionStream(cw, context.Background(), req, app.Application)
	if err != errAIMonitoringDisabled {
		t.Error(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{})
	app.ExpectTxnEvents(t, []internal.WantEvent{})
}

func TestNRCreateStreamError(t *testing.T) {
	mockClient := &MockOpenAIClient{}
	cw := &ClientWrapper{
		Client:             mockClient,
		LicenseKeyLastFour: "sk-mnop",
	}
	req := openai.ChatCompletionRequest{
		Model:       openai.GPT3Dot5Turbo,
		Temperature: 0,
		MaxTokens:   1500,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "testError",
			},
		},
		Stream: true,
	}
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true))
	_, err := NRCreateChatCompletionStream(cw, context.Background(), req, app.Application)
	if err.Error() != "test error" {
		t.Error(err)
	}

	app.ExpectErrorEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":            "TransactionError",
				"transactionName": "OtherTransaction/Go/OpenAIChatCompletionStream",
				"guid":            internal.MatchAnything,
				"priority":        internal.MatchAnything,
				"sampled":         internal.MatchAnything,
				"traceId":         internal.MatchAnything,
				"error.class":     "OpenAIError",
				"error.message":   "test error",
			},
		}})

}
