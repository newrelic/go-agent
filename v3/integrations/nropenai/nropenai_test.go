package nropenai

import (
	"context"
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sashabaranov/go-openai"
)

type MockOpenAIClient struct {
	MockCreateChatCompletionResp openai.ChatCompletionResponse
	MockCreateEmbeddingsResp     openai.EmbeddingResponse
	MockCreateChatCompletionErr  error
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

	MockResponse.SetHeader(hdrs)

	return MockResponse, m.MockCreateChatCompletionErr
}

func (m *MockOpenAIClient) CreateEmbeddings(ctx context.Context, conv openai.EmbeddingRequestConverter) (res openai.EmbeddingResponse, err error) {
	return m.MockCreateEmbeddingsResp, m.MockCreateChatCompletionErr
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
		"ll.pi":   3.14,
	})
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
		},
	})

}
