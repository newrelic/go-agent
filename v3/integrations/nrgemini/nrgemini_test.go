package nrgemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
	"google.golang.org/genai"
)

const (
	testModel     = "gemini-2.5-flash"
	testPrompt    = "What is 8*5"
	testResponse  = "Hello there, how may I assist you today?"
	testMessageID = "resp_abc123"
)

// noCodeLevelMetrics disables CLM so code.* agent attributes don't appear in
// test assertions and cause spurious length mismatches.
func noCodeLevelMetrics(cfg *newrelic.Config) {
	cfg.CodeLevelMetrics.Enabled = false
}

// mockGeminiClient returns an NRClient wired to a test HTTP server. The
// handler is called for every request.
func mockGeminiClient(t *testing.T, app *newrelic.Application, handler http.HandlerFunc) *NRClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	nrClient, err := NewClient(app, context.Background(), &genai.ClientConfig{
		APIKey:  "test-key",
		Backend: genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{
			BaseURL: srv.URL,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return nrClient
}

func successHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"candidates": []map[string]interface{}{
			{
				"content": map[string]interface{}{
					"role":  "model",
					"parts": []map[string]interface{}{{"text": testResponse}},
				},
				"finishReason": "STOP",
			},
		},
		"modelVersion": testModel,
		"responseId":   testMessageID,
		"usageMetadata": map[string]interface{}{
			"promptTokenCount":     9,
			"candidatesTokenCount": 12,
			"totalTokenCount":      21,
		},
	})
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    400,
			"message": "test error",
			"status":  "INVALID_ARGUMENT",
		},
	})
}

func streamingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	writeSSE := func(payload map[string]interface{}) {
		data, _ := json.Marshal(payload)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}

	writeSSE(map[string]interface{}{
		"candidates": []map[string]interface{}{
			{
				"content": map[string]interface{}{
					"role":  "model",
					"parts": []map[string]interface{}{{"text": testResponse}},
				},
			},
		},
		"modelVersion": testModel,
		"responseId":   testMessageID,
	})

	writeSSE(map[string]interface{}{
		"candidates": []map[string]interface{}{
			{
				"content": map[string]interface{}{
					"role":  "model",
					"parts": []map[string]interface{}{{"text": ""}},
				},
				"finishReason": "STOP",
			},
		},
		"modelVersion": testModel,
		"responseId":   testMessageID,
	})

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func TestAddCustomAttributes(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, successHandler)

	nrClient.AddCustomAttributes(map[string]interface{}{
		"llm.foo": "bar",
	})
	if nrClient.Models.customAttributes["llm.foo"] != "bar" {
		t.Errorf("expected llm.foo=bar, got %v", nrClient.Models.customAttributes["llm.foo"])
	}
}

func TestAddCustomAttributesIncorrectPrefix(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, successHandler)

	nrClient.AddCustomAttributes(map[string]interface{}{
		"notllm.foo": "bar",
	})
	if len(nrClient.Models.customAttributes) != 0 {
		t.Errorf("expected no custom attributes, got %d", len(nrClient.Models.customAttributes))
	}
}

func TestGenerateContent(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, successHandler)

	resp, err := nrClient.Models.GenerateContent(context.Background(), testModel, genai.Text(testPrompt), &genai.GenerateContentConfig{MaxOutputTokens: 150})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text() != testResponse {
		t.Errorf("unexpected response text: %q", resp.Text())
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
				"request.max_tokens":             int32(150),
				"vendor":                         "gemini",
				"ingest_source":                  "Go",
				"duration":                       internal.MatchAnything,
				"response.model":                 testModel,
				"response.choices.finish_reason": "STOP",
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
				"vendor":         "gemini",
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
				"role":           "model",
				"content":        testResponse,
				"vendor":         "gemini",
				"ingest_source":  "Go",
				"response.model": testModel,
				"is_response":    true,
			},
		},
	})
}

func TestGenerateContentAIMonitoringNotEnabled(t *testing.T) {
	app := integrationsupport.NewTestApp(nil) // AI monitoring NOT enabled
	nrClient := mockGeminiClient(t, app.Application, successHandler)

	resp, err := nrClient.Models.GenerateContent(context.Background(), testModel, genai.Text(testPrompt), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text() != testResponse {
		t.Errorf("unexpected response text: %q", resp.Text())
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{})
}

func TestGenerateContentError(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, errorHandler)

	_, err := nrClient.Models.GenerateContent(context.Background(), testModel, genai.Text(testPrompt), &genai.GenerateContentConfig{MaxOutputTokens: 150})
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
				"request.max_tokens":          int32(150),
				"vendor":                      "gemini",
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
				"vendor":         "gemini",
				"ingest_source":  "Go",
				"response.model": testModel,
			},
		},
	})

	app.ExpectErrorEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":            "TransactionError",
				"transactionName": "OtherTransaction/Go/GeminiGenerateContent",
				"guid":            internal.MatchAnything,
				"priority":        internal.MatchAnything,
				"sampled":         internal.MatchAnything,
				"traceId":         internal.MatchAnything,
				"error.class":     "GeminiError",
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

func TestGenerateContentWithExistingTxn(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, successHandler)

	txn := app.StartTransaction("my-existing-txn")
	ctx := newrelic.NewContext(context.Background(), txn)

	resp, err := nrClient.Models.GenerateContent(ctx, testModel, genai.Text(testPrompt), &genai.GenerateContentConfig{MaxOutputTokens: 150})
	txn.End()

	if err != nil {
		t.Fatal(err)
	}
	if resp.Text() != testResponse {
		t.Errorf("unexpected response text: %q", resp.Text())
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

func TestGenerateContentStream(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	stream := nrClient.Models.GenerateContentStream(context.Background(), testModel, genai.Text(testPrompt), &genai.GenerateContentConfig{MaxOutputTokens: 150})

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
				"request.max_tokens":             int32(150),
				"vendor":                         "gemini",
				"ingest_source":                  "Go",
				"duration":                       internal.MatchAnything,
				"response.model":                 testModel,
				"response.choices.finish_reason": "STOP",
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
				"vendor":         "gemini",
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
				"role":           "model",
				"content":        testResponse,
				"vendor":         "gemini",
				"ingest_source":  "Go",
				"response.model": testModel,
				"is_response":    true,
			},
		},
	})
}

func TestGenerateContentStreamAIMonitoringNotEnabled(t *testing.T) {
	app := integrationsupport.NewTestApp(nil) // AI monitoring NOT enabled
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	stream := nrClient.Models.GenerateContentStream(context.Background(), testModel, genai.Text(testPrompt), nil)
	for stream.Next() {
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{})
	app.ExpectTxnEvents(t, []internal.WantEvent{})
}

func TestGenerateContentStreamDisabled(t *testing.T) {
	// AI monitoring enabled but streaming specifically disabled — no txn should be started.
	app := integrationsupport.NewTestApp(nil,
		newrelic.ConfigAIMonitoringEnabled(true),
		func(cfg *newrelic.Config) { cfg.AIMonitoring.Streaming.Enabled = false },
		noCodeLevelMetrics,
	)
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	stream := nrClient.Models.GenerateContentStream(context.Background(), testModel, genai.Text(testPrompt), nil)

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

func TestGenerateContentStreamWithExistingTxn(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	txn := app.StartTransaction("my-existing-streaming-txn")
	ctx := newrelic.NewContext(context.Background(), txn)

	stream := nrClient.Models.GenerateContentStream(ctx, testModel, genai.Text(testPrompt), nil)
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

func TestExtractContentText(t *testing.T) {
	tests := []struct {
		name string
		in   *genai.Content
		want string
	}{
		{name: "nil", in: nil, want: ""},
		{name: "empty parts", in: &genai.Content{}, want: ""},
		{
			name: "single text",
			in:   &genai.Content{Parts: []*genai.Part{{Text: "hello"}}},
			want: "hello",
		},
		{
			name: "multiple text",
			in: &genai.Content{Parts: []*genai.Part{
				{Text: "hello"},
				{Text: "world"},
			}},
			want: "hello world",
		},
		{
			name: "empty text ignored",
			in: &genai.Content{Parts: []*genai.Part{
				{Text: "hello"},
				{Text: ""},
				{Text: "world"},
			}},
			want: "hello world",
		},
		{
			name: "nil part ignored",
			in: &genai.Content{Parts: []*genai.Part{
				{Text: "hello"},
				nil,
				{Text: "world"},
			}},
			want: "hello world",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractContentText(tc.in); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractResponseText(t *testing.T) {
	tests := []struct {
		name string
		in   *genai.GenerateContentResponse
		want string
	}{
		{name: "nil", in: nil, want: ""},
		{name: "no candidates", in: &genai.GenerateContentResponse{}, want: ""},
		{
			name: "nil content",
			in: &genai.GenerateContentResponse{Candidates: []*genai.Candidate{
				{Content: nil},
			}},
			want: "",
		},
		{
			name: "single text",
			in: &genai.GenerateContentResponse{Candidates: []*genai.Candidate{
				{Content: &genai.Content{Parts: []*genai.Part{{Text: "hello"}}}},
			}},
			want: "hello",
		},
		{
			name: "multiple text",
			in: &genai.GenerateContentResponse{Candidates: []*genai.Candidate{
				{Content: &genai.Content{Parts: []*genai.Part{
					{Text: "hello"},
					{Text: "world"},
				}}},
			}},
			want: "hello world",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractResponseText(tc.in); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// --- NRGenerateContentStreamWrapper.appendCustomAttrs tests ---

func TestStreamWrapperAppendCustomAttrs(t *testing.T) {
	w := &NRGenerateContentStreamWrapper{
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
	w := &NRGenerateContentStreamWrapper{customAttributes: map[string]interface{}{}}
	data := map[string]interface{}{"k": "v"}
	w.appendCustomAttrs(data)
	if len(data) != 1 || data["k"] != "v" {
		t.Errorf("data should be unchanged, got %v", data)
	}
}

// --- NRClient.AddCustomAttributes additional tests ---

func TestAddCustomAttributesMixed(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, successHandler)

	nrClient.AddCustomAttributes(map[string]interface{}{
		"llm.valid":   "yes",
		"notllm.skip": "no",
		"also.skip":   "no",
	})

	if len(nrClient.Models.customAttributes) != 1 {
		t.Fatalf("expected 1 attribute, got %d: %v", len(nrClient.Models.customAttributes), nrClient.Models.customAttributes)
	}
	if nrClient.Models.customAttributes["llm.valid"] != "yes" {
		t.Errorf("llm.valid: got %v, want yes", nrClient.Models.customAttributes["llm.valid"])
	}
}

func TestAddCustomAttributesEmpty(t *testing.T) {
	app := integrationsupport.NewTestApp(nil)
	nrClient := mockGeminiClient(t, app.Application, successHandler)
	nrClient.AddCustomAttributes(map[string]interface{}{})
	if len(nrClient.Models.customAttributes) != 0 {
		t.Errorf("expected no attributes, got %d", len(nrClient.Models.customAttributes))
	}
}

// --- NRModelsService.recordSummary tests ---

func newTestService(t *testing.T) (*NRModelsService, integrationsupport.ExpectApp) {
	t.Helper()
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	svc := &NRModelsService{
		app:              app.Application,
		customAttributes: make(map[string]interface{}),
	}
	return svc, app
}

func baseContents() []*genai.Content {
	return genai.Text(testPrompt)
}

func baseConfig() *genai.GenerateContentConfig {
	return &genai.GenerateContentConfig{MaxOutputTokens: 150}
}

func TestRecordSummarySuccess(t *testing.T) {
	svc, app := newTestService(t)
	resp := &genai.GenerateContentResponse{
		ResponseID:   testMessageID,
		ModelVersion: testModel,
		Candidates: []*genai.Candidate{
			{FinishReason: genai.FinishReasonStop},
		},
	}

	svc.recordSummary("cid", "sid", "tid", testModel, baseConfig(), baseContents(), resp, 100, false)

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
				"request.max_tokens":             int32(150),
				"vendor":                         "gemini",
				"ingest_source":                  "Go",
				"duration":                       int64(100),
				"response.model":                 testModel,
				"response.choices.finish_reason": "STOP",
				"response.number_of_messages":    2,
			},
		},
	})
}

func TestRecordSummaryError(t *testing.T) {
	svc, app := newTestService(t)

	svc.recordSummary("cid", "sid", "tid", testModel, baseConfig(), baseContents(), nil, 50, true)

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
				"request.max_tokens":          int32(150),
				"vendor":                      "gemini",
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
	// Use a value exactly representable in float32 so the harness comparison
	// isn't tripped by float32 precision loss.
	temp := float32(0.5)
	config := baseConfig()
	config.Temperature = &temp
	resp := &genai.GenerateContentResponse{
		ResponseID:   testMessageID,
		ModelVersion: testModel,
		Candidates: []*genai.Candidate{
			{FinishReason: genai.FinishReasonStop},
		},
	}

	svc.recordSummary("cid", "sid", "tid", testModel, config, baseContents(), resp, 10, false)

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
				"request.max_tokens":             int32(150),
				"request.temperature":            float32(0.5),
				"vendor":                         "gemini",
				"ingest_source":                  "Go",
				"duration":                       int64(10),
				"response.model":                 testModel,
				"response.choices.finish_reason": "STOP",
				"response.number_of_messages":    2,
			},
		},
	})
}

func TestRecordSummaryCustomAttrs(t *testing.T) {
	svc, app := newTestService(t)
	svc.customAttributes["llm.env"] = "prod"
	resp := &genai.GenerateContentResponse{
		ModelVersion: testModel,
		Candidates: []*genai.Candidate{
			{FinishReason: genai.FinishReasonStop},
		},
	}

	svc.recordSummary("cid", "sid", "tid", testModel, baseConfig(), baseContents(), resp, 10, false)

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
				"request.max_tokens":             int32(150),
				"vendor":                         "gemini",
				"ingest_source":                  "Go",
				"duration":                       int64(10),
				"response.model":                 testModel,
				"response.choices.finish_reason": "STOP",
				"response.number_of_messages":    2,
				"llm.env":                        "prod",
			},
		},
	})
}

func TestRecordSummaryNilConfig(t *testing.T) {
	svc, app := newTestService(t)
	resp := &genai.GenerateContentResponse{
		ModelVersion: testModel,
		Candidates: []*genai.Candidate{
			{FinishReason: genai.FinishReasonStop},
		},
	}

	svc.recordSummary("cid", "sid", "tid", testModel, nil, baseContents(), resp, 10, false)

	// no request.max_tokens or request.temperature should be recorded when config is nil
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
				"vendor":                         "gemini",
				"ingest_source":                  "Go",
				"duration":                       int64(10),
				"response.model":                 testModel,
				"response.choices.finish_reason": "STOP",
				"response.number_of_messages":    2,
			},
		},
	})
}

// --- NRModelsService.recordMessages tests ---

func TestRecordMessagesWithResp(t *testing.T) {
	svc, app := newTestService(t)
	resp := &genai.GenerateContentResponse{
		ResponseID:   testMessageID,
		ModelVersion: testModel,
		Candidates: []*genai.Candidate{
			{Content: &genai.Content{
				Role:  genai.RoleModel,
				Parts: []*genai.Part{{Text: testResponse}},
			}},
		},
	}

	svc.recordMessages("cid", "sid", "tid", testModel, baseContents(), resp)

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
				"vendor":         "gemini",
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
				"role":           "model",
				"completion_id":  "cid",
				"sequence":       1,
				"response.model": testModel,
				"vendor":         "gemini",
				"ingest_source":  "Go",
				"content":        testResponse,
				"is_response":    true,
			},
		},
	})
}

func TestRecordMessagesNilResp(t *testing.T) {
	svc, app := newTestService(t)

	svc.recordMessages("cid", "sid", "tid", testModel, baseContents(), nil)

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
				"vendor":         "gemini",
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
	svc := &NRModelsService{
		app:              app.Application,
		customAttributes: make(map[string]interface{}),
	}
	resp := &genai.GenerateContentResponse{
		ResponseID:   testMessageID,
		ModelVersion: testModel,
		Candidates: []*genai.Candidate{
			{Content: &genai.Content{
				Role:  genai.RoleModel,
				Parts: []*genai.Part{{Text: testResponse}},
			}},
		},
	}

	svc.recordMessages("cid", "sid", "tid", testModel, baseContents(), resp)

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
				"vendor":         "gemini",
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
				"role":           "model",
				"completion_id":  "cid",
				"sequence":       1,
				"response.model": testModel,
				"vendor":         "gemini",
				"ingest_source":  "Go",
				"is_response":    true,
			},
		},
	})
}

// --- NRGenerateContentStreamWrapper.Next state accumulation ---

func TestStreamWrapperNextAccumulatesState(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	stream := nrClient.Models.GenerateContentStream(context.Background(), testModel, genai.Text(testPrompt), nil)

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
	if stream.finishReason != "STOP" {
		t.Errorf("finishReason: got %q, want %q", stream.finishReason, "STOP")
	}
	if stream.responseRole != genai.RoleModel {
		t.Errorf("responseRole: got %q, want %q", stream.responseRole, genai.RoleModel)
	}

	stream.Close()
}

// --- NRGenerateContentStreamWrapper.Close tests ---

func TestStreamWrapperCloseReturnsNilOnSuccess(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	stream := nrClient.Models.GenerateContentStream(context.Background(), testModel, genai.Text(testPrompt), nil)
	for stream.Next() {
	}

	if err := stream.Close(); err != nil {
		t.Errorf("Close() returned unexpected error: %v", err)
	}
}

func TestStreamWrapperCloseIdempotent(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	stream := nrClient.Models.GenerateContentStream(context.Background(), testModel, genai.Text(testPrompt), nil)
	for stream.Next() {
	}

	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	// Second Close should be a no-op (return nil, no double-flush of events).
	if err := stream.Close(); err != nil {
		t.Errorf("second Close() should be nil, got: %v", err)
	}
}

func TestStreamWrapperCloseEndsTxnWhenOwned(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	// No txn in context — wrapper creates and owns the transaction.
	stream := nrClient.Models.GenerateContentStream(context.Background(), testModel, genai.Text(testPrompt), nil)
	for stream.Next() {
	}
	stream.Close()

	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"type":      "Transaction",
				"name":      "OtherTransaction/Go/GeminiGenerateContentStream",
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
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	// Inject an existing txn — wrapper must NOT end it.
	txn := app.StartTransaction("caller-txn")
	ctx := newrelic.NewContext(context.Background(), txn)

	stream := nrClient.Models.GenerateContentStream(ctx, testModel, genai.Text(testPrompt), nil)
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

// --- LLM content spec check: full content survives to serialized event ---

// The LLM spec requires that LlmChatCompletionMessage.content is NOT truncated
// at any stage. This test drives a long (>256-byte) prompt through the sync
// integration and verifies the recorded event carries the full content.
func TestGenerateContentLongPromptNotTruncated(t *testing.T) {
	svc, app := newTestService(t)
	longPrompt := strings.Repeat("a", 1024)
	resp := &genai.GenerateContentResponse{
		ResponseID:   testMessageID,
		ModelVersion: testModel,
		Candidates: []*genai.Candidate{
			{Content: &genai.Content{
				Role:  genai.RoleModel,
				Parts: []*genai.Part{{Text: testResponse}},
			}},
		},
	}

	svc.recordMessages("cid", "sid", "tid", testModel, genai.Text(longPrompt), resp)

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
				"vendor":         "gemini",
				"ingest_source":  "Go",
				"content":        longPrompt,
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
				"role":           "model",
				"completion_id":  "cid",
				"sequence":       1,
				"response.model": testModel,
				"vendor":         "gemini",
				"ingest_source":  "Go",
				"content":        testResponse,
				"is_response":    true,
			},
		},
	})
}
