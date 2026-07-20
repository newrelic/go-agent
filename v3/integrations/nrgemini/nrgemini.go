// Package nrgemini provides New Relic instrumentation for the Google GenAI Go
// SDK (google.golang.org/genai).
//
// Wrap your GenAI client with NewClient, then use nrClient.Models.GenerateContent
// in place of client.Models.GenerateContent to automatically record
// LlmChatCompletionSummary and LlmChatCompletionMessage custom events and a
// segment under the active transaction — mirroring the Python agent's
// mlmodel_gemini instrumentation.
//
// The New Relic transaction must be present in the context (via
// newrelic.NewContext) for instrumentation to activate. AI monitoring must also
// be enabled on the application (newrelic.ConfigAIMonitoringEnabled(true)).
package nrgemini

import (
	"context"
	"fmt"
	"iter"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
	"google.golang.org/genai"
)

var reportStreamingDisabled func()

func init() {
	reportStreamingDisabled = sync.OnceFunc(func() {
		internal.TrackUsage("Go", "ML", "Streaming", "Disabled")
	})
	info, ok := debug.ReadBuildInfo()
	if info != nil && ok {
		for _, module := range info.Deps {
			if module != nil && strings.Contains(module.Path, "google.golang.org/genai") {
				internal.TrackUsage("Go", "ML", "Gemini", module.Version)
				return
			}
		}
	}
	internal.TrackUsage("Go", "ML", "Gemini", "unknown")
}

// NRClient wraps a GenAI client with New Relic instrumentation.
type NRClient struct {
	Client *genai.Client
	Models NRModelsService
}

// NRModelsService wraps the GenAI Models service with instrumentation.
type NRModelsService struct {
	models           *genai.Models
	app              *newrelic.Application
	customAttributes map[string]interface{}
}

// NewClient creates an NRClient wrapping a new GenAI client initialized with cfg.
func NewClient(app *newrelic.Application, ctx context.Context, cfg *genai.ClientConfig) (*NRClient, error) {
	client, err := genai.NewClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	nrc := &NRClient{Client: client}
	nrc.Models = NRModelsService{
		models:           client.Models,
		app:              app,
		customAttributes: make(map[string]interface{}),
	}
	return nrc, nil
}

// AddCustomAttributes attaches llm.* prefixed key-value pairs to all LLM events
// recorded by this client.
func (c *NRClient) AddCustomAttributes(attrs map[string]interface{}) {
	for k, v := range attrs {
		if strings.HasPrefix(k, "llm.") {
			c.Models.customAttributes[k] = v
		}
	}
}

// GenerateContent wraps client.Models.GenerateContent with New Relic instrumentation.
//
// If ctx carries a New Relic transaction (via newrelic.NewContext) that
// transaction is used and its lifecycle is left to the caller. If no
// transaction is present a new one named "GeminiGenerateContent" is started and
// ended automatically. AI monitoring must be enabled on the application
// (newrelic.ConfigAIMonitoringEnabled(true)); otherwise the call is forwarded
// to the underlying SDK without instrumentation.
func (s *NRModelsService) GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	cfg, _ := s.app.Config()
	if !cfg.AIMonitoring.Enabled {
		return s.models.GenerateContent(ctx, model, contents, config)
	}

	txn := newrelic.FromContext(ctx)
	if txn == nil {
		txn = s.app.StartTransaction("GeminiGenerateContent")
		defer txn.End()
		ctx = newrelic.NewContext(ctx, txn)
	}

	integrationsupport.AddAgentAttribute(txn, "llm", "", true)

	completionID := uuid.New().String()
	spanID := txn.GetTraceMetadata().SpanID
	traceID := txn.GetTraceMetadata().TraceID

	seg := txn.StartSegment("Llm/completion/Gemini/GenerateContent")
	start := time.Now()
	resp, err := s.models.GenerateContent(ctx, model, contents, config)
	duration := time.Since(start).Milliseconds()
	seg.End()

	if err != nil {
		txn.NoticeError(newrelic.Error{
			Message: err.Error(),
			Class:   "GeminiError",
			Attributes: map[string]interface{}{
				"completion_id": completionID,
			},
		})
		s.recordSummary(completionID, spanID, traceID, model, config, contents, nil, duration, true)
		s.recordMessages(completionID, spanID, traceID, model, contents, nil)
		return resp, err
	}

	s.recordSummary(completionID, spanID, traceID, model, config, contents, resp, duration, false)
	s.recordMessages(completionID, spanID, traceID, model, contents, resp)
	return resp, nil
}

func (s *NRModelsService) recordSummary(completionID, spanID, traceID, model string, config *genai.GenerateContentConfig, contents []*genai.Content, resp *genai.GenerateContentResponse, duration int64, isError bool) {
	data := map[string]interface{}{
		"id":            completionID,
		"span_id":       spanID,
		"trace_id":      traceID,
		"request.model": model,
		"vendor":        "gemini",
		"ingest_source": "Go",
		"duration":      duration,
	}

	if config != nil {
		if config.MaxOutputTokens != 0 {
			data["request.max_tokens"] = config.MaxOutputTokens
		}
		if config.Temperature != nil {
			data["request.temperature"] = *config.Temperature
		}
	}

	if isError {
		data["error"] = true
		data["response.number_of_messages"] = len(contents)
	} else if resp != nil {
		if resp.ModelVersion != "" {
			data["response.model"] = resp.ModelVersion
		}
		if len(resp.Candidates) > 0 && resp.Candidates[0].FinishReason != "" {
			data["response.choices.finish_reason"] = string(resp.Candidates[0].FinishReason)
		}
		data["response.number_of_messages"] = len(contents) + 1
	}

	s.appendCustomAttrs(data)
	s.app.RecordCustomEvent("LlmChatCompletionSummary", data)
}

func (s *NRModelsService) recordMessages(completionID, spanID, traceID, model string, contents []*genai.Content, resp *genai.GenerateContentResponse) {
	cfg, _ := s.app.Config()
	responseModel := model
	if resp != nil && resp.ModelVersion != "" {
		responseModel = resp.ModelVersion
	}

	for i, c := range contents {
		text := extractContentText(c)
		role := c.Role
		if role == "" {
			role = genai.RoleUser
		}
		data := map[string]interface{}{
			"id":             uuid.New().String(),
			"span_id":        spanID,
			"trace_id":       traceID,
			"role":           role,
			"completion_id":  completionID,
			"sequence":       i,
			"response.model": responseModel,
			"vendor":         "gemini",
			"ingest_source":  "Go",
		}
		if cfg.AIMonitoring.RecordContent.Enabled && text != "" {
			data["content"] = text
		}
		if tokens, ok := s.app.InvokeLLMTokenCountCallback(responseModel, text); ok {
			data["token_count"] = tokens
		}
		s.appendCustomAttrs(data)
		s.app.RecordCustomEvent("LlmChatCompletionMessage", data)
	}

	if resp == nil {
		return
	}

	responseText := extractResponseText(resp)
	responseSeq := len(contents)
	responseRole := genai.RoleModel
	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil && resp.Candidates[0].Content.Role != "" {
		responseRole = resp.Candidates[0].Content.Role
	}
	data := map[string]interface{}{
		"id":             fmt.Sprintf("%s-%d", resp.ResponseID, responseSeq),
		"span_id":        spanID,
		"trace_id":       traceID,
		"role":           responseRole,
		"completion_id":  completionID,
		"sequence":       responseSeq,
		"response.model": responseModel,
		"vendor":         "gemini",
		"ingest_source":  "Go",
		"is_response":    true,
	}
	if cfg.AIMonitoring.RecordContent.Enabled && responseText != "" {
		data["content"] = responseText
	}
	if tokens, ok := s.app.InvokeLLMTokenCountCallback(responseModel, responseText); ok {
		data["token_count"] = tokens
	}
	s.appendCustomAttrs(data)
	s.app.RecordCustomEvent("LlmChatCompletionMessage", data)
}

func (s *NRModelsService) appendCustomAttrs(data map[string]interface{}) {
	for k, v := range s.customAttributes {
		data[k] = v
	}
}

func extractContentText(c *genai.Content) string {
	if c == nil {
		return ""
	}
	var parts []string
	for _, p := range c.Parts {
		if p != nil && p.Text != "" {
			parts = append(parts, p.Text)
		}
	}
	return strings.Join(parts, " ")
}

func extractResponseText(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return ""
	}
	var parts []string
	for _, p := range resp.Candidates[0].Content.Parts {
		if p != nil && p.Text != "" {
			parts = append(parts, p.Text)
		}
	}
	return strings.Join(parts, " ")
}

// NRGenerateContentStreamWrapper wraps a GenAI streaming iterator with New Relic
// instrumentation. Call Next() to advance the stream, Current() to read the
// current chunk, Err() to check for errors, and Close() to flush NR events and
// release resources.
type NRGenerateContentStreamWrapper struct {
	next             func() (*genai.GenerateContentResponse, error, bool)
	stop             func()
	err              error
	current          *genai.GenerateContentResponse
	app              *newrelic.Application
	txn              *newrelic.Transaction
	txnOwned         bool
	seg              *newrelic.Segment
	customAttributes map[string]interface{}
	model            string
	contents         []*genai.Content
	config           *genai.GenerateContentConfig
	completionID     string
	spanID           string
	traceID          string
	responseText     strings.Builder
	responseID       string
	responseModel    string
	finishReason     string
	responseRole     string
	start            time.Time
	closed           bool
}

// GenerateContentStream wraps client.Models.GenerateContentStream with New Relic
// instrumentation. The returned wrapper exposes Next/Current/Err/Close. Call
// Close() after the stream is consumed to record NR events.
func (s *NRModelsService) GenerateContentStream(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) *NRGenerateContentStreamWrapper {
	cfg, _ := s.app.Config()

	if !cfg.AIMonitoring.Streaming.Enabled {
		if reportStreamingDisabled != nil {
			reportStreamingDisabled()
		}
	}

	seq := s.models.GenerateContentStream(ctx, model, contents, config)
	next, stop := iter.Pull2(seq)

	if !cfg.AIMonitoring.Enabled || !cfg.AIMonitoring.Streaming.Enabled {
		return &NRGenerateContentStreamWrapper{
			app:      s.app,
			model:    model,
			contents: contents,
			config:   config,
			next:     next,
			stop:     stop,
			start:    time.Now(),
		}
	}

	txn := newrelic.FromContext(ctx)
	txnOwned := false
	if txn == nil {
		txn = s.app.StartTransaction("GeminiGenerateContentStream")
		txnOwned = true
		ctx = newrelic.NewContext(ctx, txn)
	}

	integrationsupport.AddAgentAttribute(txn, "llm", "", true)
	seg := txn.StartSegment("Llm/completion/Gemini/GenerateContentStream")

	return &NRGenerateContentStreamWrapper{
		app:              s.app,
		txn:              txn,
		txnOwned:         txnOwned,
		seg:              seg,
		customAttributes: s.customAttributes,
		model:            model,
		contents:         contents,
		config:           config,
		completionID:     uuid.New().String(),
		spanID:           txn.GetTraceMetadata().SpanID,
		traceID:          txn.GetTraceMetadata().TraceID,
		start:            time.Now(),
		next:             next,
		stop:             stop,
	}
}

// Next advances the stream to the next chunk and accumulates response state.
func (w *NRGenerateContentStreamWrapper) Next() bool {
	chunk, err, ok := w.next()
	if !ok {
		return false
	}
	if err != nil {
		w.err = err
		return false
	}
	w.current = chunk
	if chunk != nil {
		if chunk.ResponseID != "" {
			w.responseID = chunk.ResponseID
		}
		if chunk.ModelVersion != "" {
			w.responseModel = chunk.ModelVersion
		}
		if len(chunk.Candidates) > 0 {
			cand := chunk.Candidates[0]
			if cand.Content != nil {
				if cand.Content.Role != "" {
					w.responseRole = cand.Content.Role
				}
				for _, p := range cand.Content.Parts {
					if p != nil && p.Text != "" {
						w.responseText.WriteString(p.Text)
					}
				}
			}
			if cand.FinishReason != "" {
				w.finishReason = string(cand.FinishReason)
			}
		}
	}
	return true
}

// Current returns the most recent stream chunk.
func (w *NRGenerateContentStreamWrapper) Current() *genai.GenerateContentResponse {
	return w.current
}

// Err returns any error encountered during streaming.
func (w *NRGenerateContentStreamWrapper) Err() error {
	return w.err
}

// Close records LlmChatCompletionSummary and LlmChatCompletionMessage events,
// ends the segment, ends the transaction if it was started by this wrapper,
// and releases the underlying iterator.
func (w *NRGenerateContentStreamWrapper) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	w.recordCustomEvent()
	if w.txnOwned {
		w.txn.End()
	}
	w.stop()
	return nil
}

func (w *NRGenerateContentStreamWrapper) recordCustomEvent() {
	cfg, _ := w.app.Config()
	if !cfg.AIMonitoring.Enabled || !cfg.AIMonitoring.Streaming.Enabled {
		return
	}

	if w.seg != nil {
		w.seg.End()
	}

	duration := time.Since(w.start).Milliseconds()
	isError := w.err != nil

	if isError {
		w.txn.NoticeError(newrelic.Error{
			Message: w.err.Error(),
			Class:   "GeminiError",
			Attributes: map[string]interface{}{
				"completion_id": w.completionID,
			},
		})
	}

	model := w.model
	if w.responseModel != "" {
		model = w.responseModel
	}

	summaryData := map[string]interface{}{
		"id":            w.completionID,
		"span_id":       w.spanID,
		"trace_id":      w.traceID,
		"request.model": w.model,
		"vendor":        "gemini",
		"ingest_source": "Go",
		"duration":      duration,
	}
	if w.config != nil {
		if w.config.MaxOutputTokens != 0 {
			summaryData["request.max_tokens"] = w.config.MaxOutputTokens
		}
		if w.config.Temperature != nil {
			summaryData["request.temperature"] = *w.config.Temperature
		}
	}
	if isError {
		summaryData["error"] = true
		summaryData["response.number_of_messages"] = len(w.contents)
	} else {
		summaryData["response.model"] = model
		if w.finishReason != "" {
			summaryData["response.choices.finish_reason"] = w.finishReason
		}
		summaryData["response.number_of_messages"] = len(w.contents) + 1
	}
	w.appendCustomAttrs(summaryData)
	w.app.RecordCustomEvent("LlmChatCompletionSummary", summaryData)

	for i, c := range w.contents {
		text := extractContentText(c)
		role := c.Role
		if role == "" {
			role = genai.RoleUser
		}
		msgData := map[string]interface{}{
			"id":             uuid.New().String(),
			"span_id":        w.spanID,
			"trace_id":       w.traceID,
			"role":           role,
			"completion_id":  w.completionID,
			"sequence":       i,
			"response.model": model,
			"vendor":         "gemini",
			"ingest_source":  "Go",
		}
		if cfg.AIMonitoring.RecordContent.Enabled && text != "" {
			msgData["content"] = text
		}
		if tokens, ok := w.app.InvokeLLMTokenCountCallback(model, text); ok {
			msgData["token_count"] = tokens
		}
		w.appendCustomAttrs(msgData)
		w.app.RecordCustomEvent("LlmChatCompletionMessage", msgData)
	}

	if isError {
		return
	}

	responseText := w.responseText.String()
	responseSeq := len(w.contents)
	responseRole := w.responseRole
	if responseRole == "" {
		responseRole = genai.RoleModel
	}
	respData := map[string]interface{}{
		"id":             fmt.Sprintf("%s-%d", w.responseID, responseSeq),
		"span_id":        w.spanID,
		"trace_id":       w.traceID,
		"role":           responseRole,
		"completion_id":  w.completionID,
		"sequence":       responseSeq,
		"response.model": model,
		"vendor":         "gemini",
		"ingest_source":  "Go",
		"is_response":    true,
	}
	if cfg.AIMonitoring.RecordContent.Enabled && responseText != "" {
		respData["content"] = responseText
	}
	if tokens, ok := w.app.InvokeLLMTokenCountCallback(model, responseText); ok {
		respData["token_count"] = tokens
	}
	w.appendCustomAttrs(respData)
	w.app.RecordCustomEvent("LlmChatCompletionMessage", respData)
}

func (w *NRGenerateContentStreamWrapper) appendCustomAttrs(data map[string]interface{}) {
	for k, v := range w.customAttributes {
		data[k] = v
	}
}
