package nranthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
)

const (
	testModel     = "claude-3-5-sonnet-20241022"
	testPrompt    = "What is 8*5"
	testResponse  = "Hello there, how may I assist you today?"
	testMessageID = "msg_abc123"
)

// noCodeLevelMetrics disables CLM so code.* agent attributes don't appear in
// test assertions and cause spurious length mismatches.
func noCodeLevelMetrics(cfg *newrelic.Config) {
	cfg.CodeLevelMetrics.Enabled = false
}

// mockAnthropicServer returns a test server and an Anthropic client pointing at it.
// The handler is called for every request.
func mockAnthropicServer(t *testing.T, handler http.HandlerFunc) *anthropic.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := anthropic.NewClient(
		option.WithAPIKey("test-key"),
		option.WithBaseURL(srv.URL),
	)
	return &client
}

func successHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":            testMessageID,
		"type":          "message",
		"role":          "assistant",
		"model":         testModel,
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"content": []map[string]interface{}{
			{"type": "text", "text": testResponse},
		},
		"usage": map[string]interface{}{
			"input_tokens":  9,
			"output_tokens": 12,
		},
	})
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"type": "error",
		"error": map[string]interface{}{
			"type":    "invalid_request_error",
			"message": "test error",
		},
	})
}

func TestAddCustomAttributes(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	client := mockAnthropicServer(t, successHandler)
	nrClient := NewClient(app.Application, client)

	nrClient.AddCustomAttributes(map[string]interface{}{
		"llm.foo": "bar",
	})
	if nrClient.customAttributes["llm.foo"] != "bar" {
		t.Errorf("expected llm.foo=bar, got %v", nrClient.customAttributes["llm.foo"])
	}
}

func TestAddCustomAttributesIncorrectPrefix(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	client := mockAnthropicServer(t, successHandler)
	nrClient := NewClient(app.Application, client)

	nrClient.AddCustomAttributes(map[string]interface{}{
		"notllm.foo": "bar",
	})
	if len(nrClient.customAttributes) != 0 {
		t.Errorf("expected no custom attributes, got %d", len(nrClient.customAttributes))
	}
}

func TestNRMessagesNew(t *testing.T) {
	client := mockAnthropicServer(t, successHandler)
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := NewClient(app.Application, client)

	resp, err := nrClient.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model(testModel),
		MaxTokens: 150,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(testPrompt)),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Content) == 0 || resp.Content[0].Text != testResponse {
		t.Errorf("unexpected response content: %v", resp.Content)
	}

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionSummary",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"id":                             internal.MatchAnything,
				"span_id":                        internal.MatchAnything,
				"trace_id":                       internal.MatchAnything,
				"request.model":                  testModel,
				"request.max_tokens":             int64(150),
				"vendor":                         "anthropic",
				"ingest_source":                  "Go",
				"duration":                       internal.MatchAnything,
				"response.model":                 testModel,
				"response.choices.finish_reason": "end_turn",
				"response.number_of_messages":    2,
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionMessage",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"id":             internal.MatchAnything,
				"span_id":        internal.MatchAnything,
				"trace_id":       internal.MatchAnything,
				"completion_id":  internal.MatchAnything,
				"sequence":       0,
				"role":           "user",
				"content":        testPrompt,
				"vendor":         "anthropic",
				"ingest_source":  "Go",
				"response.model": testModel,
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionMessage",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"id":             internal.MatchAnything,
				"span_id":        internal.MatchAnything,
				"trace_id":       internal.MatchAnything,
				"completion_id":  internal.MatchAnything,
				"sequence":       1,
				"role":           "assistant",
				"content":        testResponse,
				"vendor":         "anthropic",
				"ingest_source":  "Go",
				"response.model": testModel,
				"is_response":    true,
			},
		},
	})
}

func TestNRMessagesNewAIMonitoringNotEnabled(t *testing.T) {
	client := mockAnthropicServer(t, successHandler)
	app := integrationsupport.NewTestApp(nil) // AI monitoring NOT enabled
	nrClient := NewClient(app.Application, client)

	resp, err := nrClient.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model(testModel),
		MaxTokens: 150,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(testPrompt)),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Call still succeeds, just with no instrumentation
	if len(resp.Content) == 0 || resp.Content[0].Text != testResponse {
		t.Errorf("unexpected response content: %v", resp.Content)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{})
}

func TestNRMessagesNewError(t *testing.T) {
	client := mockAnthropicServer(t, errorHandler)
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := NewClient(app.Application, client)

	_, err := nrClient.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model(testModel),
		MaxTokens: 150,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(testPrompt)),
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionSummary",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"id":                          internal.MatchAnything,
				"span_id":                     internal.MatchAnything,
				"trace_id":                    internal.MatchAnything,
				"request.model":               testModel,
				"request.max_tokens":          int64(150),
				"vendor":                      "anthropic",
				"ingest_source":               "Go",
				"duration":                    internal.MatchAnything,
				"error":                       true,
				"response.number_of_messages": 1,
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionMessage",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"id":             internal.MatchAnything,
				"span_id":        internal.MatchAnything,
				"trace_id":       internal.MatchAnything,
				"completion_id":  internal.MatchAnything,
				"sequence":       0,
				"role":           "user",
				"content":        testPrompt,
				"vendor":         "anthropic",
				"ingest_source":  "Go",
				"response.model": testModel,
			},
		},
	})

	app.ExpectErrorEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":            "TransactionError",
				"transactionName": "OtherTransaction/Go/AnthropicMessageNew",
				"guid":            internal.MatchAnything,
				"priority":        internal.MatchAnything,
				"sampled":         internal.MatchAnything,
				"traceId":         internal.MatchAnything,
				"error.class":     "AnthropicError",
				"error.message":   internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"completion_id": internal.MatchAnything,
			},
			AgentAttributes: map[string]interface{}{
				"llm": true,
			},
		},
	})
}

func TestNRMessagesNewWithExistingTxn(t *testing.T) {
	client := mockAnthropicServer(t, successHandler)
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := NewClient(app.Application, client)

	txn := app.StartTransaction("my-existing-txn")
	ctx := newrelic.NewContext(context.Background(), txn)

	resp, err := nrClient.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(testModel),
		MaxTokens: 150,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(testPrompt)),
		},
	})
	txn.End()

	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Content) == 0 || resp.Content[0].Text != testResponse {
		t.Errorf("unexpected response content: %v", resp.Content)
	}

	// Events should be recorded under the caller's transaction
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "Transaction",
				"name":      "OtherTransaction/Go/my-existing-txn",
				"guid":      internal.MatchAnything,
				"priority":  internal.MatchAnything,
				"sampled":   internal.MatchAnything,
				"traceId":   internal.MatchAnything,
				"timestamp": internal.MatchAnything,
				"duration":  internal.MatchAnything,
				"totalTime": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"llm": true,
			},
		},
	})
}
