package nrgemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	testRequestID = "req_xyz789"
	testProject   = "my-gcp-project"
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

// responseText is the text of a response's first candidate.
func responseText(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 {
		return ""
	}
	text, _ := candidateText(resp.Candidates[0])
	return text
}

// drainStream consumes a stream the way calling code is expected to.
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
				"id":                               internal.MatchAnything,
				"span_id":                          internal.MatchAnything,
				"trace_id":                         internal.MatchAnything,
				"request_id":                       testMessageID,
				"request.model":                    testModel,
				"request.max_tokens":               int32(150),
				"vendor":                           "gemini",
				"ingest_source":                    "Go",
				"duration":                         internal.MatchAnything,
				"response.model":                   testModel,
				"response.choices.finish_reason":   "STOP",
				"response.number_of_messages":      2,
				"response.usage.prompt_tokens":     int32(9),
				"response.usage.completion_tokens": int32(12),
				"response.usage.total_tokens":      int32(21),
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
				"request_id":     testMessageID,
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
				"request_id":     testMessageID,
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
				// Read off the genai.APIError.
				"http.statusCode": 400,
				"error.code":      "INVALID_ARGUMENT",
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

func TestContentText(t *testing.T) {
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
			if got, _ := contentText(tc.in); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResponseText(t *testing.T) {
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
			if got := responseText(tc.in); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// --- NRModelsService.appendCustomAttrs tests ---

func TestAppendCustomAttrs(t *testing.T) {
	s := &NRModelsService{
		customAttributes: map[string]interface{}{
			"llm.key1": "val1",
			"llm.key2": 42,
		},
	}
	data := map[string]interface{}{"existing": "preserved"}
	s.appendCustomAttrs(data)

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

func TestAppendCustomAttrsEmpty(t *testing.T) {
	s := &NRModelsService{customAttributes: map[string]interface{}{}}
	data := map[string]interface{}{"k": "v"}
	s.appendCustomAttrs(data)
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

var errTest = errors.New("test error")

// newTestCompletion uses the fixed ids the event assertions expect.
func newTestCompletion(config *genai.GenerateContentConfig, resp *genai.GenerateContentResponse, err error, duration int64) *completion {
	return &completion{
		id:       "cid",
		spanID:   "sid",
		traceID:  "tid",
		model:    testModel,
		contents: baseContents(),
		config:   config,
		resp:     resp,
		err:      err,
		duration: duration,
	}
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

	svc.recordSummary(newTestCompletion(baseConfig(), resp, nil, 100))

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
				"request_id":                     testMessageID,
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

	svc.recordSummary(newTestCompletion(baseConfig(), nil, errTest, 50))

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

	svc.recordSummary(newTestCompletion(config, resp, nil, 10))

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
				"request_id":                     testMessageID,
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

	svc.recordSummary(newTestCompletion(baseConfig(), resp, nil, 10))

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

	svc.recordSummary(newTestCompletion(nil, resp, nil, 10))

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

	svc.recordMessages(newTestCompletion(nil, resp, nil, 0))

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
				"request_id":     testMessageID,
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
				"request_id":     testMessageID,
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

	svc.recordMessages(newTestCompletion(nil, nil, nil, 0))

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

	svc.recordMessages(newTestCompletion(nil, resp, nil, 0))

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
				"request_id":     testMessageID,
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
				"request_id":     testMessageID,
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

	svc.recordMessages(&completion{id: "cid", spanID: "sid", traceID: "tid", model: testModel, contents: genai.Text(longPrompt), resp: resp})

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
				"request_id":     testMessageID,
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
				"request_id":     testMessageID,
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

// --- Multiple candidates ---

// Each candidate is its own response message.
func TestRecordMessagesMultipleCandidates(t *testing.T) {
	svc, app := newTestService(t)
	resp := &genai.GenerateContentResponse{
		ResponseID:   testMessageID,
		ModelVersion: testModel,
		Candidates: []*genai.Candidate{
			{Content: &genai.Content{Role: genai.RoleModel, Parts: []*genai.Part{{Text: "first"}}}},
			{Content: &genai.Content{Role: genai.RoleModel, Parts: []*genai.Part{{Text: "second"}}}},
		},
	}
	svc.recordMessages(newTestCompletion(nil, resp, nil, 0))

	events := []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{"type": "LlmChatCompletionMessage", "timestamp": internal.MatchAnything},
			UserAttributes: map[string]interface{}{
				"id":             internal.MatchAnything,
				"span_id":        "sid",
				"trace_id":       "tid",
				"request_id":     testMessageID,
				"completion_id":  "cid",
				"sequence":       0,
				"role":           "user",
				"content":        testPrompt,
				"vendor":         "gemini",
				"ingest_source":  "Go",
				"response.model": testModel,
			},
		},
	}
	for i, text := range []string{"first", "second"} {
		events = append(events, internal.WantEvent{
			Intrinsics: map[string]interface{}{"type": "LlmChatCompletionMessage", "timestamp": internal.MatchAnything},
			UserAttributes: map[string]interface{}{
				"id":             fmt.Sprintf("%s-%d", testMessageID, i+1),
				"span_id":        "sid",
				"trace_id":       "tid",
				"request_id":     testMessageID,
				"completion_id":  "cid",
				"sequence":       i + 1,
				"role":           "model",
				"content":        text,
				"vendor":         "gemini",
				"ingest_source":  "Go",
				"response.model": testModel,
				"is_response":    true,
			},
		})
	}
	app.ExpectCustomEvents(t, events)
}

// number_of_messages counts every candidate.
func TestRecordSummaryMultipleCandidates(t *testing.T) {
	svc, app := newTestService(t)
	resp := &genai.GenerateContentResponse{
		ResponseID:   testMessageID,
		ModelVersion: testModel,
		Candidates: []*genai.Candidate{
			// No reason here, so the summary takes the next candidate's.
			{},
			{FinishReason: genai.FinishReasonMaxTokens},
		},
	}
	svc.recordSummary(newTestCompletion(nil, resp, nil, 10))

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{"type": "LlmChatCompletionSummary", "timestamp": internal.MatchAnything},
			UserAttributes: map[string]interface{}{
				"id":                             "cid",
				"span_id":                        "sid",
				"trace_id":                       "tid",
				"request_id":                     testMessageID,
				"request.model":                  testModel,
				"vendor":                         "gemini",
				"ingest_source":                  "Go",
				"duration":                       10,
				"response.model":                 testModel,
				"response.choices.finish_reason": "MAX_TOKENS",
				"response.number_of_messages":    3,
			},
		},
	})
}

// --- Roles ---

func TestRoleDefaults(t *testing.T) {
	// Requests default to "user".
	if got := contentRole(&genai.Content{}); got != genai.RoleUser {
		t.Errorf("contentRole with no role: got %q, want %q", got, genai.RoleUser)
	}
	if got := contentRole(nil); got != genai.RoleUser {
		t.Errorf("contentRole(nil): got %q, want %q", got, genai.RoleUser)
	}
	// Responses default to "assistant".
	if got := candidateRole(&genai.Candidate{Content: &genai.Content{}}); got != roleAssistant {
		t.Errorf("candidateRole with no role: got %q, want %q", got, roleAssistant)
	}
	if got := candidateRole(nil); got != roleAssistant {
		t.Errorf("candidateRole(nil): got %q, want %q", got, roleAssistant)
	}
	// Gemini's own role passes through.
	if got := candidateRole(&genai.Candidate{Content: &genai.Content{Role: genai.RoleModel}}); got != genai.RoleModel {
		t.Errorf("candidateRole: got %q, want %q", got, genai.RoleModel)
	}
}

// --- Message ids ---

func TestMessageID(t *testing.T) {
	withID := &completion{resp: &genai.GenerateContentResponse{ResponseID: testMessageID}}
	if got, want := withID.messageID(2), testMessageID+"-2"; got != want {
		t.Errorf("messageID: got %q, want %q", got, want)
	}

	// No response ID means a UUID, never a bare "-<sequence>".
	for _, c := range []*completion{
		{resp: &genai.GenerateContentResponse{}},
		{},
	} {
		got := c.messageID(0)
		if strings.HasPrefix(got, "-") || got == "" {
			t.Errorf("messageID fallback should be a UUID, got %q", got)
		}
		if len(got) != len("00000000-0000-0000-0000-000000000000") {
			t.Errorf("messageID fallback is not UUID-shaped: %q", got)
		}
	}
}

// --- request_id and response headers ---

func TestRequestIDPrefersHeaderOverResponseID(t *testing.T) {
	hdr := http.Header{}
	hdr.Set("X-Goog-Request-Id", testRequestID)
	c := &completion{resp: &genai.GenerateContentResponse{
		ResponseID:      testMessageID,
		SDKHTTPResponse: &genai.HTTPResponse{Headers: hdr},
	}}
	if got := c.requestID(); got != testRequestID {
		t.Errorf("requestID: got %q, want the header value %q", got, testRequestID)
	}

	// No header: the response ID stands in.
	c.resp.SDKHTTPResponse = nil
	if got := c.requestID(); got != testMessageID {
		t.Errorf("requestID fallback: got %q, want %q", got, testMessageID)
	}

	// A failed call has neither.
	if got := (&completion{}).requestID(); got != "" {
		t.Errorf("requestID with no response: got %q, want empty", got)
	}
}

func TestAddHeaderAttrs(t *testing.T) {
	hdr := http.Header{}
	hdr.Set("X-Goog-User-Project", testProject)
	hdr.Set("X-Ratelimit-Limit-Requests", "200")
	hdr.Set("X-Ratelimit-Reset-Tokens", "2ms")

	data := map[string]interface{}{}
	addHeaderAttrs(data, hdr)

	want := map[string]interface{}{
		"response.organization":                   testProject,
		"response.headers.ratelimitLimitRequests": "200",
		"response.headers.ratelimitResetTokens":   "2ms",
	}
	for k, v := range want {
		if data[k] != v {
			t.Errorf("%s: got %v, want %v", k, data[k], v)
		}
	}
	// Absent headers are omitted.
	if len(data) != len(want) {
		t.Errorf("unexpected extra attributes: %v", data)
	}

	// No headers: event untouched.
	empty := map[string]interface{}{}
	addHeaderAttrs(empty, nil)
	if len(empty) != 0 {
		t.Errorf("expected no attributes, got %v", empty)
	}
}

// Only the first chunk sets it. TestGenerateContent covers the other half: its
// summary is asserted exactly and carries no such attribute.
func TestContentTextSkipsNonTextParts(t *testing.T) {
	content := &genai.Content{
		Role: genai.RoleUser,
		Parts: []*genai.Part{
			{Text: "look at this"},
			{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte{1, 2, 3}}},
			{FunctionCall: &genai.FunctionCall{Name: "get_weather"}},
			nil,
			{Text: "and this"},
		},
	}
	text, skipped := contentText(content)
	if want := "look at this and this"; text != want {
		t.Errorf("text: got %q, want %q", text, want)
	}
	// The blob and the function call; the nil part is not a dropped part.
	if skipped != 2 {
		t.Errorf("skipped: got %d, want 2", skipped)
	}

	if _, skipped := contentText(&genai.Content{Parts: []*genai.Part{{Text: "all text"}}}); skipped != 0 {
		t.Errorf("skipped: got %d, want 0", skipped)
	}
}

// A message whose parts are all non-text still records, just without content.
func TestRecordMessagesNonTextOnly(t *testing.T) {
	svc, app := newTestService(t)
	c := &completion{
		id:      "cid",
		spanID:  "sid",
		traceID: "tid",
		model:   testModel,
		contents: []*genai.Content{{
			Role:  genai.RoleUser,
			Parts: []*genai.Part{{InlineData: &genai.Blob{MIMEType: "image/png"}}},
		}},
	}
	svc.recordMessages(c)

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{"type": "LlmChatCompletionMessage", "timestamp": internal.MatchAnything},
			UserAttributes: map[string]interface{}{
				"id":             internal.MatchAnything,
				"span_id":        "sid",
				"trace_id":       "tid",
				"completion_id":  "cid",
				"sequence":       0,
				"role":           "user",
				"vendor":         "gemini",
				"ingest_source":  "Go",
				"response.model": testModel,
			},
		},
	})
}

// Deltas go into a strings.Builder per candidate, so a long stream stays linear
// rather than recopying the accumulated text on every chunk.

// --- Streaming ---

func drainStream(seq iter.Seq2[*genai.GenerateContentResponse, error]) error {
	for _, err := range seq {
		if err != nil {
			return err
		}
	}
	return nil
}

// The wrapper returns the SDK's own iter.Seq2, so a caller ranges over it exactly
// as they would the unwrapped client, and the events land when the loop ends.
func TestGenerateContentStream(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	var got strings.Builder
	for chunk, err := range nrClient.Models.GenerateContentStream(context.Background(), testModel, genai.Text(testPrompt), &genai.GenerateContentConfig{MaxOutputTokens: 150}) {
		if err != nil {
			t.Fatal(err)
		}
		got.WriteString(responseText(chunk))
	}
	if got.String() != testResponse {
		t.Errorf("streamed text: got %q, want %q", got.String(), testResponse)
	}

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{"type": "LlmChatCompletionSummary", "timestamp": internal.MatchAnything},
			UserAttributes: map[string]interface{}{
				"id":                             internal.MatchAnything,
				"span_id":                        internal.MatchAnything,
				"trace_id":                       internal.MatchAnything,
				"request_id":                     testMessageID,
				"request.model":                  testModel,
				"request.max_tokens":             int32(150),
				"vendor":                         "gemini",
				"ingest_source":                  "Go",
				"duration":                       internal.MatchAnything,
				"time_to_first_token":            internal.MatchAnything,
				"response.model":                 testModel,
				"response.choices.finish_reason": "STOP",
				"response.number_of_messages":    2,
			},
		},
		{
			Intrinsics: map[string]interface{}{"type": "LlmChatCompletionMessage", "timestamp": internal.MatchAnything},
			UserAttributes: map[string]interface{}{
				"id":             internal.MatchAnything,
				"span_id":        internal.MatchAnything,
				"trace_id":       internal.MatchAnything,
				"request_id":     testMessageID,
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
			Intrinsics: map[string]interface{}{"type": "LlmChatCompletionMessage", "timestamp": internal.MatchAnything},
			UserAttributes: map[string]interface{}{
				"id":             internal.MatchAnything,
				"span_id":        internal.MatchAnything,
				"trace_id":       internal.MatchAnything,
				"request_id":     testMessageID,
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

// Breaking out of the loop early still records, covering the chunks seen so far.
// finish_reason is absent because it arrives on the last chunk.
func TestGenerateContentStreamEarlyBreak(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	chunks := 0
	for range nrClient.Models.GenerateContentStream(context.Background(), testModel, genai.Text(testPrompt), nil) {
		chunks++
		break
	}
	if chunks != 1 {
		t.Fatalf("chunks: got %d, want 1", chunks)
	}

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{"type": "LlmChatCompletionSummary", "timestamp": internal.MatchAnything},
			UserAttributes: map[string]interface{}{
				"id":                          internal.MatchAnything,
				"span_id":                     internal.MatchAnything,
				"trace_id":                    internal.MatchAnything,
				"request_id":                  testMessageID,
				"request.model":               testModel,
				"vendor":                      "gemini",
				"ingest_source":               "Go",
				"duration":                    internal.MatchAnything,
				"time_to_first_token":         internal.MatchAnything,
				"response.model":              testModel,
				"response.number_of_messages": 2,
			},
		},
		{
			Intrinsics: map[string]interface{}{"type": "LlmChatCompletionMessage", "timestamp": internal.MatchAnything},
			UserAttributes: map[string]interface{}{
				"id":             internal.MatchAnything,
				"span_id":        internal.MatchAnything,
				"trace_id":       internal.MatchAnything,
				"request_id":     testMessageID,
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
			Intrinsics: map[string]interface{}{"type": "LlmChatCompletionMessage", "timestamp": internal.MatchAnything},
			UserAttributes: map[string]interface{}{
				"id":             internal.MatchAnything,
				"span_id":        internal.MatchAnything,
				"trace_id":       internal.MatchAnything,
				"request_id":     testMessageID,
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

	if err := drainStream(nrClient.Models.GenerateContentStream(context.Background(), testModel, genai.Text(testPrompt), nil)); err != nil {
		t.Fatal(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{})
	app.ExpectTxnEvents(t, []internal.WantEvent{})
}

// AI monitoring on, streaming off: the SDK iterator passes through untouched, so
// no transaction is started and nothing is recorded.
func TestGenerateContentStreamDisabled(t *testing.T) {
	app := integrationsupport.NewTestApp(nil,
		newrelic.ConfigAIMonitoringEnabled(true),
		func(cfg *newrelic.Config) { cfg.AIMonitoring.Streaming.Enabled = false },
		noCodeLevelMetrics,
	)
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	if err := drainStream(nrClient.Models.GenerateContentStream(context.Background(), testModel, genai.Text(testPrompt), nil)); err != nil {
		t.Fatal(err)
	}
	app.ExpectCustomEvents(t, []internal.WantEvent{})
	app.ExpectTxnEvents(t, []internal.WantEvent{})
}

// A caller-supplied transaction is left for the caller to end.
func TestGenerateContentStreamWithExistingTxn(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	txn := app.StartTransaction("my-existing-streaming-txn")
	ctx := newrelic.NewContext(context.Background(), txn)
	if err := drainStream(nrClient.Models.GenerateContentStream(ctx, testModel, genai.Text(testPrompt), nil)); err != nil {
		t.Fatal(err)
	}
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
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{"llm": true},
		},
	})
}

// With no transaction in the context the stream starts and ends its own.
func TestGenerateContentStreamStartsOwnTxn(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	if err := drainStream(nrClient.Models.GenerateContentStream(context.Background(), testModel, genai.Text(testPrompt), nil)); err != nil {
		t.Fatal(err)
	}

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

// Ranging a second time must not report the completion twice.
func TestGenerateContentStreamSingleUse(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	nrClient := mockGeminiClient(t, app.Application, streamingHandler)

	seq := nrClient.Models.GenerateContentStream(context.Background(), testModel, genai.Text(testPrompt), nil)
	if err := drainStream(seq); err != nil {
		t.Fatal(err)
	}
	for range seq {
		t.Error("second range yielded a chunk")
	}

	app.ExpectCustomEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{"type": "LlmChatCompletionSummary", "timestamp": internal.MatchAnything},
			UserAttributes: map[string]interface{}{
				"id":                             internal.MatchAnything,
				"span_id":                        internal.MatchAnything,
				"trace_id":                       internal.MatchAnything,
				"request_id":                     testMessageID,
				"request.model":                  testModel,
				"vendor":                         "gemini",
				"ingest_source":                  "Go",
				"duration":                       internal.MatchAnything,
				"time_to_first_token":            internal.MatchAnything,
				"response.model":                 testModel,
				"response.choices.finish_reason": "STOP",
				"response.number_of_messages":    2,
			},
		},
		{
			Intrinsics: map[string]interface{}{"type": "LlmChatCompletionMessage", "timestamp": internal.MatchAnything},
			UserAttributes: map[string]interface{}{
				"id":             internal.MatchAnything,
				"span_id":        internal.MatchAnything,
				"trace_id":       internal.MatchAnything,
				"request_id":     testMessageID,
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
			Intrinsics: map[string]interface{}{"type": "LlmChatCompletionMessage", "timestamp": internal.MatchAnything},
			UserAttributes: map[string]interface{}{
				"id":             internal.MatchAnything,
				"span_id":        internal.MatchAnything,
				"trace_id":       internal.MatchAnything,
				"request_id":     testMessageID,
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

func newTestStreamState() *streamState {
	return &streamState{completion: &completion{model: testModel}, start: time.Now()}
}

// textChunk builds a chunk carrying one text delta per candidate.
func textChunk(deltas ...string) *genai.GenerateContentResponse {
	chunk := &genai.GenerateContentResponse{}
	for _, d := range deltas {
		chunk.Candidates = append(chunk.Candidates, &genai.Candidate{
			Content: &genai.Content{Role: genai.RoleModel, Parts: []*genai.Part{{Text: d}}},
		})
	}
	return chunk
}

func TestStreamStateAccumulates(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	txn := app.StartTransaction("accumulate")
	defer txn.End()

	st := newTestStreamState()
	st.observe(txn, &genai.GenerateContentResponse{
		ResponseID:   testMessageID,
		ModelVersion: testModel,
		Candidates: []*genai.Candidate{{
			Content: &genai.Content{Role: genai.RoleModel, Parts: []*genai.Part{{Text: testResponse}}},
		}},
	}, nil)
	st.observe(txn, &genai.GenerateContentResponse{
		Candidates:    []*genai.Candidate{{FinishReason: genai.FinishReasonStop}},
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{TotalTokenCount: 21},
	}, nil)

	resp := st.finish().resp
	if resp.ResponseID != testMessageID {
		t.Errorf("ResponseID: got %q, want %q", resp.ResponseID, testMessageID)
	}
	if resp.ModelVersion != testModel {
		t.Errorf("ModelVersion: got %q, want %q", resp.ModelVersion, testModel)
	}
	if got := responseText(resp); got != testResponse {
		t.Errorf("text: got %q, want %q", got, testResponse)
	}
	if resp.Candidates[0].FinishReason != genai.FinishReasonStop {
		t.Errorf("FinishReason: got %q, want STOP", resp.Candidates[0].FinishReason)
	}
	if resp.UsageMetadata == nil || resp.UsageMetadata.TotalTokenCount != 21 {
		t.Errorf("usage not carried from the final chunk: %+v", resp.UsageMetadata)
	}
}

func TestStreamStateConcatenatesDeltas(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	txn := app.StartTransaction("deltas")
	defer txn.End()

	st := newTestStreamState()
	for _, delta := range []string{"Hel", "lo", " wor", "ld"} {
		st.observe(txn, textChunk(delta), nil)
	}
	if got := responseText(st.finish().resp); got != "Hello world" {
		t.Errorf("text: got %q, want %q", got, "Hello world")
	}
}

func TestStreamStateManyChunks(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	txn := app.StartTransaction("many-chunks")
	defer txn.End()

	st := newTestStreamState()
	var want strings.Builder
	for i := 0; i < 500; i++ {
		delta := fmt.Sprintf("chunk%d ", i)
		want.WriteString(delta)
		st.observe(txn, textChunk(delta), nil)
	}
	if got := responseText(st.finish().resp); got != want.String() {
		t.Errorf("accumulated text mismatch: got %d bytes, want %d", len(got), want.Len())
	}
}

// Each candidate accumulates its own deltas.
func TestStreamStatePerCandidate(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	txn := app.StartTransaction("two-candidates")
	defer txn.End()

	st := newTestStreamState()
	st.observe(txn, textChunk("one ", "two "), nil)
	st.observe(txn, textChunk("first", "second"), nil)

	resp := st.finish().resp
	if len(resp.Candidates) != 2 {
		t.Fatalf("candidates: got %d, want 2", len(resp.Candidates))
	}
	for i, want := range []string{"one first", "two second"} {
		if got, _ := candidateText(resp.Candidates[i]); got != want {
			t.Errorf("candidate %d: got %q, want %q", i, got, want)
		}
	}
}

func TestStreamStateTimeToFirstTokenSetOnce(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	txn := app.StartTransaction("ttft")
	defer txn.End()

	st := newTestStreamState()
	st.observe(txn, textChunk("first"), nil)
	if st.completion.timeToFirstToken == nil {
		t.Fatal("expected the first chunk to record a time to first token")
	}
	first := *st.completion.timeToFirstToken

	time.Sleep(5 * time.Millisecond)
	st.observe(txn, textChunk("later"), nil)
	if got := *st.completion.timeToFirstToken; got != first {
		t.Errorf("timeToFirstToken moved: got %d, want %d", got, first)
	}
}

func TestStreamStateDurationStampedAsChunksArrive(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	txn := app.StartTransaction("duration")
	defer txn.End()

	st := newTestStreamState()
	st.observe(txn, textChunk("only"), nil)
	atChunk := st.completion.duration

	time.Sleep(25 * time.Millisecond)
	if got := st.finish().duration; got != atChunk {
		t.Errorf("duration moved after the delay: got %d, want %d", got, atChunk)
	}
}

// A stream that yielded nothing still gets a duration.
func TestStreamStateDurationWhenNothingYielded(t *testing.T) {
	st := newTestStreamState()
	time.Sleep(5 * time.Millisecond)
	if got := st.finish().duration; got <= 0 {
		t.Errorf("duration: got %d, want > 0", got)
	}
}

func TestStreamStateKeepsFirstError(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	txn := app.StartTransaction("errors")
	defer txn.End()

	first := errors.New("first failure")
	st := newTestStreamState()
	st.observe(txn, nil, first)
	st.observe(txn, nil, errors.New("second failure"))

	c := st.finish()
	if !errors.Is(c.err, first) {
		t.Errorf("err: got %v, want %v", c.err, first)
	}
	// A failed stream reports request messages only.
	if c.resp != nil {
		t.Errorf("resp: got %+v, want nil", c.resp)
	}
}

// A yield carrying neither a chunk nor an error must not mark the stream as
// observed, or finish would skip measuring the duration and report zero.
func TestStreamStateEmptyYieldLeavesDurationMeasured(t *testing.T) {
	app := integrationsupport.NewTestApp(nil, newrelic.ConfigAIMonitoringEnabled(true), noCodeLevelMetrics)
	txn := app.StartTransaction("empty-yield")
	defer txn.End()

	st := newTestStreamState()
	st.observe(txn, nil, nil)
	if st.observed {
		t.Error("an empty yield should not count as observed")
	}

	time.Sleep(5 * time.Millisecond)
	if got := st.finish().duration; got <= 0 {
		t.Errorf("duration: got %d, want > 0", got)
	}
}
