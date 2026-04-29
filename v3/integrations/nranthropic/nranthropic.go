// Package nranthropic provides New Relic instrumentation for the Anthropic Go
// SDK (github.com/anthropics/anthropic-sdk-go).
//
// Wrap your Anthropic client with NewClient, then use nrClient.Messages.New in
// place of client.Messages.New to automatically record LlmChatCompletionSummary
// and LlmChatCompletionMessage custom events and a segment under the active
// transaction — mirroring the Python agent's mlmodel_anthropic instrumentation.
//
// The New Relic transaction must be present in the context (via
// newrelic.NewContext) for instrumentation to activate. AI monitoring must also
// be enabled on the application (newrelic.ConfigAIMonitoringEnabled(true)).
package nranthropic

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/google/uuid"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
)

// Version is the current version of the nranthropic integration.
const Version = "0.1.0"

func init() {
	info, ok := debug.ReadBuildInfo()
	if info != nil && ok {
		for _, module := range info.Deps {
			if module != nil && strings.Contains(module.Path, "anthropic-sdk-go") {
				internal.TrackUsage("Go", "ML", "Anthropic", module.Version)
				return
			}
		}
	}
	internal.TrackUsage("Go", "ML", "Anthropic", "unknown")
}

// NRClient wraps an Anthropic client with New Relic instrumentation.
type NRClient struct {
	Client           *anthropic.Client
	Messages         NRMessageService
	customAttributes map[string]interface{}
}

// NRMessageService wraps the Anthropic MessageService with instrumentation.
type NRMessageService struct {
	client *anthropic.Client
	app    *newrelic.Application
	nrc    *NRClient
}

// NewClient creates an NRClient wrapping the given Anthropic client.
func NewClient(app *newrelic.Application, client *anthropic.Client) *NRClient {
	nrc := &NRClient{Client: client}
	nrc.Messages = NRMessageService{client: client, app: app, nrc: nrc}
	return nrc
}

// AddCustomAttributes attaches llm.* prefixed key-value pairs to all LLM events
// recorded by this client.
func (c *NRClient) AddCustomAttributes(attrs map[string]interface{}) {
	if c.customAttributes == nil {
		c.customAttributes = make(map[string]interface{})
	}
	for k, v := range attrs {
		if strings.HasPrefix(k, "llm.") {
			c.customAttributes[k] = v
		}
	}
}

// New wraps client.Messages.New with New Relic instrumentation.
//
// If ctx carries a New Relic transaction (via newrelic.NewContext) that
// transaction is used and its lifecycle is left to the caller. If no
// transaction is present a new one named "AnthropicMessageNew" is started and
// ended automatically. AI monitoring must be enabled on the application
// (newrelic.ConfigAIMonitoringEnabled(true)); otherwise the call is forwarded
// to the underlying SDK without instrumentation.
func (s *NRMessageService) New(ctx context.Context, params anthropic.MessageNewParams, opts ...option.RequestOption) (*anthropic.Message, error) {
	cfg, _ := s.app.Config()
	if !cfg.AIMonitoring.Enabled {
		return s.client.Messages.New(ctx, params, opts...)
	}

	txn := newrelic.FromContext(ctx)
	if txn == nil {
		txn = s.app.StartTransaction("AnthropicMessageNew")
		defer txn.End()
		ctx = newrelic.NewContext(ctx, txn)
	}

	integrationsupport.AddAgentAttribute(txn, "llm", "", true)

	completionID := uuid.New().String()
	spanID := txn.GetTraceMetadata().SpanID
	traceID := txn.GetTraceMetadata().TraceID

	seg := txn.StartSegment("Llm/completion/Anthropic/MessageNew")
	start := time.Now()
	resp, err := s.client.Messages.New(ctx, params, opts...)
	duration := time.Since(start).Milliseconds()
	seg.End()

	if err != nil {
		txn.NoticeError(newrelic.Error{
			Message: err.Error(),
			Class:   "AnthropicError",
			Attributes: map[string]interface{}{
				"completion_id": completionID,
			},
		})
		s.recordSummary(completionID, spanID, traceID, params, nil, duration, true)
		s.recordMessages(completionID, spanID, traceID, params, nil)
		return resp, err
	}

	s.recordSummary(completionID, spanID, traceID, params, resp, duration, false)
	s.recordMessages(completionID, spanID, traceID, params, resp)
	return resp, nil
}

func (s *NRMessageService) recordSummary(completionID, spanID, traceID string, params anthropic.MessageNewParams, resp *anthropic.Message, duration int64, isError bool) {
	data := map[string]interface{}{
		"id":                 completionID,
		"span_id":            spanID,
		"trace_id":           traceID,
		"request.model":      string(params.Model),
		"request.max_tokens": params.MaxTokens,
		"vendor":             "anthropic",
		"ingest_source":      "Go",
		"duration":           duration,
	}

	if params.Temperature.Valid() {
		data["request.temperature"] = params.Temperature.Value
	}

	if isError {
		data["error"] = true
		data["response.number_of_messages"] = len(params.Messages)
	} else if resp != nil {
		data["response.model"] = string(resp.Model)
		data["response.choices.finish_reason"] = string(resp.StopReason)
		data["response.number_of_messages"] = len(params.Messages) + 1
	}

	s.appendCustomAttrs(data)
	s.app.RecordCustomEvent("LlmChatCompletionSummary", data)
}

func (s *NRMessageService) recordMessages(completionID, spanID, traceID string, params anthropic.MessageNewParams, resp *anthropic.Message) {
	cfg, _ := s.app.Config()
	model := string(params.Model)
	if resp != nil {
		model = string(resp.Model)
	}

	for i, msg := range params.Messages {
		text := extractParamText(msg.Content)
		data := map[string]interface{}{
			"id":             uuid.New().String(),
			"span_id":        spanID,
			"trace_id":       traceID,
			"role":           string(msg.Role),
			"completion_id":  completionID,
			"sequence":       i,
			"response.model": model,
			"vendor":         "anthropic",
			"ingest_source":  "Go",
		}
		if cfg.AIMonitoring.RecordContent.Enabled && text != "" {
			data["content"] = text
		}
		if tokens, ok := s.app.InvokeLLMTokenCountCallback(model, text); ok {
			data["token_count"] = tokens
		}
		s.appendCustomAttrs(data)
		s.app.RecordCustomEvent("LlmChatCompletionMessage", data)
	}

	if resp == nil {
		return
	}

	responseText := extractResponseText(resp.Content)
	responseSeq := len(params.Messages)
	data := map[string]interface{}{
		"id":             fmt.Sprintf("%s-%d", resp.ID, responseSeq),
		"span_id":        spanID,
		"trace_id":       traceID,
		"role":           "assistant",
		"completion_id":  completionID,
		"sequence":       responseSeq,
		"response.model": model,
		"vendor":         "anthropic",
		"ingest_source":  "Go",
		"is_response":    true,
	}
	if cfg.AIMonitoring.RecordContent.Enabled && responseText != "" {
		data["content"] = responseText
	}
	if tokens, ok := s.app.InvokeLLMTokenCountCallback(model, responseText); ok {
		data["token_count"] = tokens
	}
	s.appendCustomAttrs(data)
	s.app.RecordCustomEvent("LlmChatCompletionMessage", data)
}

func (s *NRMessageService) appendCustomAttrs(data map[string]interface{}) {
	for k, v := range s.nrc.customAttributes {
		data[k] = v
	}
}

func extractParamText(blocks []anthropic.ContentBlockParamUnion) string {
	var parts []string
	for _, b := range blocks {
		if b.OfText != nil {
			parts = append(parts, b.OfText.Text)
		}
	}
	return strings.Join(parts, " ")
}

func extractResponseText(blocks []anthropic.ContentBlockUnion) string {
	var parts []string
	for _, b := range blocks {
		if b.Type == "text" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, " ")
}
