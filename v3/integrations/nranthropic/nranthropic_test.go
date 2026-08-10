package nranthropic

import (
	"context"
	"encoding/json"
	"fmt"
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

// mockAnthropicServer returns request options pointing at a test server.
// The handler is called for every request.
func mockAnthropicServer(t *testing.T, handler http.HandlerFunc) []option.RequestOption {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return []option.RequestOption{
		option.WithAPIKey("test-key"),
		option.WithBaseURL(srv.URL),
	}
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
	nrClient := NewClient(app.Application, mockAnthropicServer(t, successHandler)...)

	nrClient.AddCustomAttributes(map[string]interface{}{
		"llm.foo": "bar",
	})
	if nrClient.Messages.customAttributes["llm.foo"] != "bar" {
		t.Errorf("expected llm.foo=bar, got %v", nrClient.Messages.customAttributes["llm.foo"])
	}
}

func TestAddCustomAttributesIncorrectPrefix(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := NewClient(app.Application, mockAnthropicServer(t, successHandler)...)

	nrClient.AddCustomAttributes(map[string]interface{}{
		"notllm.foo": "bar",
	})
	if len(nrClient.Messages.customAttributes) != 0 {
		t.Errorf("expected no custom attributes, got %d", len(nrClient.Messages.customAttributes))
	}
}

func TestNRMessagesNew(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := NewClient(app.Application, mockAnthropicServer(t, successHandler)...)

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
	app := integrationsupport.NewTestApp(nil) // AI monitoring NOT enabled
	nrClient := NewClient(app.Application, mockAnthropicServer(t, successHandler)...)

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
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := NewClient(app.Application, mockAnthropicServer(t, errorHandler)...)

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

func streamingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	writeSSE := func(eventType, data string) {
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data)
	}

	startMsg, _ := json.Marshal(map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":    testMessageID,
			"type":  "message",
			"role":  "assistant",
			"model": testModel,
			"usage": map[string]interface{}{"input_tokens": 9, "output_tokens": 0},
		},
	})
	writeSSE("message_start", string(startMsg))

	delta1, _ := json.Marshal(map[string]interface{}{
		"type":  "content_block_delta",
		"index": 0,
		"delta": map[string]interface{}{"type": "text_delta", "text": testResponse},
	})
	writeSSE("content_block_delta", string(delta1))

	msgDelta, _ := json.Marshal(map[string]interface{}{
		"type":  "message_delta",
		"delta": map[string]interface{}{"stop_reason": "end_turn", "stop_sequence": nil},
		"usage": map[string]interface{}{"output_tokens": 12},
	})
	writeSSE("message_delta", string(msgDelta))

	writeSSE("message_stop", `{"type":"message_stop"}`)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func TestNRMessagesNewWithExistingTxn(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := NewClient(app.Application, mockAnthropicServer(t, successHandler)...)

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

func TestNRMessagesNewStreaming(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := NewClient(app.Application, mockAnthropicServer(t, streamingHandler)...)

	stream := nrClient.Messages.NewStreaming(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model(testModel),
		MaxTokens: 150,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(testPrompt)),
		},
	})

	for stream.Next() {
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
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

func TestNRMessagesNewStreamingAIMonitoringNotEnabled(t *testing.T) {
	app := integrationsupport.NewTestApp(nil) // AI monitoring NOT enabled
	nrClient := NewClient(app.Application, mockAnthropicServer(t, streamingHandler)...)

	stream := nrClient.Messages.NewStreaming(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model(testModel),
		MaxTokens: 150,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(testPrompt)),
		},
	})
	for stream.Next() {
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{})
	app.ExpectTxnEvents(t, []internal.WantEvent{})
}

func TestNRMessagesNewStreamingDisabled(t *testing.T) {
	// AI monitoring enabled but streaming specifically disabled — no txn should be started.
	app := integrationsupport.NewTestApp(nil,
		newrelic.ConfigAIMonitoringEnabled(true),
		func(cfg *newrelic.Config) { cfg.AIMonitoring.Streaming.Enabled = false },
		noCodeLevelMetrics,
	)
	nrClient := NewClient(app.Application, mockAnthropicServer(t, streamingHandler)...)

	stream := nrClient.Messages.NewStreaming(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model(testModel),
		MaxTokens: 150,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(testPrompt)),
		},
	})

	if stream.txn != nil {
		t.Error("expected txn to be nil when streaming is disabled, but it was set")
	}
	if stream.txnOwned {
		t.Error("expected txnOwned=false when streaming is disabled")
	}

	for stream.Next() {
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{})
	app.ExpectTxnEvents(t, []internal.WantEvent{})
}

func TestNRMessagesNewStreamingWithExistingTxn(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := NewClient(app.Application, mockAnthropicServer(t, streamingHandler)...)

	txn := app.StartTransaction("my-existing-streaming-txn")
	ctx := newrelic.NewContext(context.Background(), txn)

	stream := nrClient.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(testModel),
		MaxTokens: 150,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(testPrompt)),
		},
	})
	for stream.Next() {
	}
	stream.Close()
	txn.End()

	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "Transaction",
				"name":      "OtherTransaction/Go/my-existing-streaming-txn",
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

// --- Pure helper function tests ---

func TestExtractParamText(t *testing.T) {
	tests := []struct {
		name   string
		blocks []anthropic.ContentBlockParamUnion
		want   string
	}{
		{name: "nil", blocks: nil, want: ""},
		{name: "empty", blocks: []anthropic.ContentBlockParamUnion{}, want: ""},
		{
			name:   "single text",
			blocks: []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock("hello")},
			want:   "hello",
		},
		{
			name: "multiple text",
			blocks: []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock("hello"),
				anthropic.NewTextBlock("world"),
			},
			want: "hello world",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractParamText(tc.blocks); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractResponseText(t *testing.T) {
	tests := []struct {
		name   string
		blocks []anthropic.ContentBlockUnion
		want   string
	}{
		{name: "nil", blocks: nil, want: ""},
		{name: "empty", blocks: []anthropic.ContentBlockUnion{}, want: ""},
		{
			name:   "single text",
			blocks: []anthropic.ContentBlockUnion{{Type: "text", Text: "hello"}},
			want:   "hello",
		},
		{
			name: "multiple text",
			blocks: []anthropic.ContentBlockUnion{
				{Type: "text", Text: "hello"},
				{Type: "text", Text: "world"},
			},
			want: "hello world",
		},
		{
			name:   "non-text block ignored",
			blocks: []anthropic.ContentBlockUnion{{Type: "tool_use", Text: "ignored"}},
			want:   "",
		},
		{
			name: "mixed text and non-text",
			blocks: []anthropic.ContentBlockUnion{
				{Type: "text", Text: "hello"},
				{Type: "tool_use", Text: "ignored"},
				{Type: "text", Text: "world"},
			},
			want: "hello world",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractResponseText(tc.blocks); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// --- NRMessageStreamWrapper.appendCustomAttrs tests ---

func TestStreamWrapperAppendCustomAttrs(t *testing.T) {
	w := &NRMessageStreamWrapper{
		customAttributes: map[string]interface{}{
			"llm.key1": "val1",
			"llm.key2": 42,
		},
	}
	data := map[string]interface{}{"existing": "preserved"}
	w.appendCustomAttrs(data)

	if data["llm.key1"] != "val1" {
		t.Errorf("llm.key1: got %v, want val1", data["llm.key1"])
	}
	if data["llm.key2"] != 42 {
		t.Errorf("llm.key2: got %v, want 42", data["llm.key2"])
	}
	if data["existing"] != "preserved" {
		t.Errorf("existing key was overwritten, got %v", data["existing"])
	}
}

func TestStreamWrapperAppendCustomAttrsEmpty(t *testing.T) {
	w := &NRMessageStreamWrapper{customAttributes: map[string]interface{}{}}
	data := map[string]interface{}{"k": "v"}
	w.appendCustomAttrs(data)
	if len(data) != 1 || data["k"] != "v" {
		t.Errorf("data should be unchanged, got %v", data)
	}
}

// --- NRClient.AddCustomAttributes additional tests ---

func TestAddCustomAttributesMixed(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := NewClient(app.Application)

	nrClient.AddCustomAttributes(map[string]interface{}{
		"llm.valid":   "yes",
		"notllm.skip": "no",
		"also.skip":   "no",
	})

	if len(nrClient.Messages.customAttributes) != 1 {
		t.Fatalf("expected 1 attribute, got %d: %v", len(nrClient.Messages.customAttributes), nrClient.Messages.customAttributes)
	}
	if nrClient.Messages.customAttributes["llm.valid"] != "yes" {
		t.Errorf("llm.valid: got %v, want yes", nrClient.Messages.customAttributes["llm.valid"])
	}
}

func TestAddCustomAttributesEmpty(t *testing.T) {
	app := integrationsupport.NewTestApp(nil)
	nrClient := NewClient(app.Application)
	nrClient.AddCustomAttributes(map[string]interface{}{})
	if len(nrClient.Messages.customAttributes) != 0 {
		t.Errorf("expected no attributes, got %d", len(nrClient.Messages.customAttributes))
	}
}

// --- NRMessageService.recordSummary tests ---

func newTestService(t *testing.T) (*NRMessageService, integrationsupport.ExpectApp) {
	t.Helper()
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	svc := &NRMessageService{
		app:              app.Application,
		customAttributes: make(map[string]interface{}),
	}
	return svc, app
}

func baseParams() anthropic.MessageNewParams {
	return anthropic.MessageNewParams{
		Model:     anthropic.Model(testModel),
		MaxTokens: 150,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(testPrompt)),
		},
	}
}

func TestRecordSummarySuccess(t *testing.T) {
	svc, app := newTestService(t)
	resp := &anthropic.Message{
		ID:         testMessageID,
		Model:      anthropic.Model(testModel),
		StopReason: anthropic.StopReasonEndTurn,
	}

	svc.recordSummary("cid", "sid", "tid", baseParams(), resp, 100, false)

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionSummary",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"id":                             "cid",
				"span_id":                        "sid",
				"trace_id":                       "tid",
				"request.model":                  testModel,
				"request.max_tokens":             int64(150),
				"vendor":                         "anthropic",
				"ingest_source":                  "Go",
				"duration":                       int64(100),
				"response.model":                 testModel,
				"response.choices.finish_reason": "end_turn",
				"response.number_of_messages":    2,
			},
		},
	})
}

func TestRecordSummaryError(t *testing.T) {
	svc, app := newTestService(t)

	svc.recordSummary("cid", "sid", "tid", baseParams(), nil, 50, true)

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionSummary",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"id":                          "cid",
				"span_id":                     "sid",
				"trace_id":                    "tid",
				"request.model":               testModel,
				"request.max_tokens":          int64(150),
				"vendor":                      "anthropic",
				"ingest_source":               "Go",
				"duration":                    int64(50),
				"error":                       true,
				"response.number_of_messages": 1,
			},
		},
	})
}

func TestRecordSummaryWithTemperature(t *testing.T) {
	svc, app := newTestService(t)
	params := baseParams()
	params.Temperature = anthropic.Float(0.7)
	resp := &anthropic.Message{
		ID:         testMessageID,
		Model:      anthropic.Model(testModel),
		StopReason: anthropic.StopReasonEndTurn,
	}

	svc.recordSummary("cid", "sid", "tid", params, resp, 10, false)

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionSummary",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"id":                             "cid",
				"span_id":                        "sid",
				"trace_id":                       "tid",
				"request.model":                  testModel,
				"request.max_tokens":             int64(150),
				"request.temperature":            0.7,
				"vendor":                         "anthropic",
				"ingest_source":                  "Go",
				"duration":                       int64(10),
				"response.model":                 testModel,
				"response.choices.finish_reason": "end_turn",
				"response.number_of_messages":    2,
			},
		},
	})
}

func TestRecordSummaryCustomAttrs(t *testing.T) {
	svc, app := newTestService(t)
	svc.customAttributes["llm.env"] = "prod"
	resp := &anthropic.Message{
		Model:      anthropic.Model(testModel),
		StopReason: anthropic.StopReasonEndTurn,
	}

	svc.recordSummary("cid", "sid", "tid", baseParams(), resp, 10, false)

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionSummary",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"id":                             "cid",
				"span_id":                        "sid",
				"trace_id":                       "tid",
				"request.model":                  testModel,
				"request.max_tokens":             int64(150),
				"vendor":                         "anthropic",
				"ingest_source":                  "Go",
				"duration":                       int64(10),
				"response.model":                 testModel,
				"response.choices.finish_reason": "end_turn",
				"response.number_of_messages":    2,
				"llm.env":                        "prod",
			},
		},
	})
}

// --- NRMessageService.recordMessages tests ---

func TestRecordMessagesWithResp(t *testing.T) {
	svc, app := newTestService(t)
	resp := &anthropic.Message{
		ID:    testMessageID,
		Model: anthropic.Model(testModel),
		Content: []anthropic.ContentBlockUnion{
			{Type: "text", Text: testResponse},
		},
	}

	svc.recordMessages("cid", "sid", "tid", baseParams(), resp)

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionMessage",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"id":             internal.MatchAnything,
				"span_id":        "sid",
				"trace_id":       "tid",
				"role":           "user",
				"completion_id":  "cid",
				"sequence":       0,
				"response.model": testModel,
				"vendor":         "anthropic",
				"ingest_source":  "Go",
				"content":        testPrompt,
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionMessage",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"id":             fmt.Sprintf("%s-%d", testMessageID, 1),
				"span_id":        "sid",
				"trace_id":       "tid",
				"role":           "assistant",
				"completion_id":  "cid",
				"sequence":       1,
				"response.model": testModel,
				"vendor":         "anthropic",
				"ingest_source":  "Go",
				"content":        testResponse,
				"is_response":    true,
			},
		},
	})
}

func TestRecordMessagesNilResp(t *testing.T) {
	svc, app := newTestService(t)

	svc.recordMessages("cid", "sid", "tid", baseParams(), nil)

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionMessage",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"id":             internal.MatchAnything,
				"span_id":        "sid",
				"trace_id":       "tid",
				"role":           "user",
				"completion_id":  "cid",
				"sequence":       0,
				"response.model": testModel,
				"vendor":         "anthropic",
				"ingest_source":  "Go",
				"content":        testPrompt,
			},
		},
	})
}

func TestRecordMessagesContentDisabled(t *testing.T) {
	app := integrationsupport.NewTestApp(nil,
		newrelic.ConfigAIMonitoringEnabled(true),
		func(cfg *newrelic.Config) { cfg.AIMonitoring.RecordContent.Enabled = false },
		noCodeLevelMetrics,
	)
	svc := &NRMessageService{
		app:              app.Application,
		customAttributes: make(map[string]interface{}),
	}
	resp := &anthropic.Message{
		ID:    testMessageID,
		Model: anthropic.Model(testModel),
		Content: []anthropic.ContentBlockUnion{
			{Type: "text", Text: testResponse},
		},
	}

	svc.recordMessages("cid", "sid", "tid", baseParams(), resp)

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionMessage",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"id":             internal.MatchAnything,
				"span_id":        "sid",
				"trace_id":       "tid",
				"role":           "user",
				"completion_id":  "cid",
				"sequence":       0,
				"response.model": testModel,
				"vendor":         "anthropic",
				"ingest_source":  "Go",
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"type":      "LlmChatCompletionMessage",
				"timestamp": internal.MatchAnything,
			},
			UserAttributes: map[string]interface{}{
				"id":             fmt.Sprintf("%s-%d", testMessageID, 1),
				"span_id":        "sid",
				"trace_id":       "tid",
				"role":           "assistant",
				"completion_id":  "cid",
				"sequence":       1,
				"response.model": testModel,
				"vendor":         "anthropic",
				"ingest_source":  "Go",
				"is_response":    true,
			},
		},
	})
}

// --- NRMessageStreamWrapper.Next state accumulation ---

func TestStreamWrapperNextAccumulatesState(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := NewClient(app.Application, mockAnthropicServer(t, streamingHandler)...)

	stream := nrClient.Messages.NewStreaming(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model(testModel),
		MaxTokens: 150,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(testPrompt)),
		},
	})

	for stream.Next() {
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}

	if stream.responseID != testMessageID {
		t.Errorf("responseID: got %q, want %q", stream.responseID, testMessageID)
	}
	if stream.responseModel != testModel {
		t.Errorf("responseModel: got %q, want %q", stream.responseModel, testModel)
	}
	if got := stream.responseText.String(); got != testResponse {
		t.Errorf("responseText: got %q, want %q", got, testResponse)
	}
	if stream.stopReason != "end_turn" {
		t.Errorf("stopReason: got %q, want %q", stream.stopReason, "end_turn")
	}

	stream.Close()
}

// --- NRMessageStreamWrapper.Close tests (post-refactor) ---

func TestStreamWrapperCloseReturnsNilOnSuccess(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := NewClient(app.Application, mockAnthropicServer(t, streamingHandler)...)

	stream := nrClient.Messages.NewStreaming(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model(testModel),
		MaxTokens: 150,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(testPrompt)),
		},
	})
	for stream.Next() {
	}

	if err := stream.Close(); err != nil {
		t.Errorf("Close() returned unexpected error: %v", err)
	}
}

func TestStreamWrapperCloseEndsTxnWhenOwned(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := NewClient(app.Application, mockAnthropicServer(t, streamingHandler)...)

	// No txn in context — wrapper creates and owns the transaction.
	stream := nrClient.Messages.NewStreaming(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model(testModel),
		MaxTokens: 150,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(testPrompt)),
		},
	})
	for stream.Next() {
	}
	stream.Close()

	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "Transaction",
				"name":      "OtherTransaction/Go/AnthropicMessageNewStreaming",
				"guid":      internal.MatchAnything,
				"priority":  internal.MatchAnything,
				"sampled":   internal.MatchAnything,
				"traceId":   internal.MatchAnything,
				"timestamp": internal.MatchAnything,
				"duration":  internal.MatchAnything,
				"totalTime": internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{"llm": true},
		},
	})
}

func TestStreamWrapperCloseNoTxnEndWhenNotOwned(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := NewClient(app.Application, mockAnthropicServer(t, streamingHandler)...)

	// Inject an existing txn — wrapper must NOT end it.
	txn := app.StartTransaction("caller-txn")
	ctx := newrelic.NewContext(context.Background(), txn)

	stream := nrClient.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(testModel),
		MaxTokens: 150,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(testPrompt)),
		},
	})
	for stream.Next() {
	}
	stream.Close()

	// Txn is still open; end it explicitly and verify the name is caller-txn, not the wrapper name.
	txn.End()

	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "Transaction",
				"name":      "OtherTransaction/Go/caller-txn",
				"guid":      internal.MatchAnything,
				"priority":  internal.MatchAnything,
				"sampled":   internal.MatchAnything,
				"traceId":   internal.MatchAnything,
				"timestamp": internal.MatchAnything,
				"duration":  internal.MatchAnything,
				"totalTime": internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{"llm": true},
		},
	})
}
