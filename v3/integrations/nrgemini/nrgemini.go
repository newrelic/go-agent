// Package nrgemini provides New Relic instrumentation for the Google GenAI Go
// SDK (google.golang.org/genai).
//
// Wrap your GenAI client with NewClient, then use the wrapped service in place of
// the SDK's own: GenerateContent, GenerateContentStream, and ProcessContentStream
// record LlmChatCompletionSummary and LlmChatCompletionMessage events, and open a
// segment under the active transaction.
//
// AI monitoring must be enabled on the application
// (newrelic.ConfigAIMonitoringEnabled(true)), plus
// ConfigAIMonitoringStreamingEnabled(true) for streaming; otherwise calls are
// forwarded to the SDK uninstrumented. A transaction in the context (via
// newrelic.NewContext) is used if present, otherwise one is started for the
// duration of the call.
package nrgemini

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/http"
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

const (
	vendorGemini = "gemini"
	ingestSource = "Go"
	errorClass   = "GeminiError"

	// roleAssistant is the spec's role for a response message with no vendor role.
	roleAssistant = "assistant"
)

// requestIDHeaders hold the request ID, most specific first.
var requestIDHeaders = []string{"X-Goog-Request-Id", "X-Request-Id"}

// llmResponseHeaders maps a response header to its response.headers.<suffix>
// attribute. Gemini documents no rate-limit headers today, so most are absent.
var llmResponseHeaders = map[string]string{
	"Llm-Version":                              "llmVersion",
	"X-Ratelimit-Limit-Requests":               "ratelimitLimitRequests",
	"X-Ratelimit-Limit-Tokens":                 "ratelimitLimitTokens",
	"X-Ratelimit-Remaining-Requests":           "ratelimitRemainingRequests",
	"X-Ratelimit-Remaining-Tokens":             "ratelimitRemainingTokens",
	"X-Ratelimit-Reset-Requests":               "ratelimitResetRequests",
	"X-Ratelimit-Reset-Tokens":                 "ratelimitResetTokens",
	"X-Ratelimit-Limit-Tokens-Usage-Based":     "ratelimitLimitTokensUsageBased",
	"X-Ratelimit-Remaining-Tokens-Usage-Based": "ratelimitRemainingTokensUsageBased",
	"X-Ratelimit-Reset-Tokens-Usage-Based":     "ratelimitResetTokensUsageBased",
}

// organizationHeaders identify the serving account or project, most specific first.
var organizationHeaders = []string{"X-Goog-User-Project", "X-Goog-Project-Id"}

// lookupHeader returns the first of names present in hdr.
func lookupHeader(hdr http.Header, names ...string) string {
	for _, name := range names {
		if v := hdr.Get(name); v != "" {
			return v
		}
	}
	return ""
}

// addHeaderAttrs adds response.organization and response.headers.<name>, skipping
// headers the response did not carry.
func addHeaderAttrs(data map[string]interface{}, hdr http.Header) {
	if len(hdr) == 0 {
		return
	}
	if org := lookupHeader(hdr, organizationHeaders...); org != "" {
		data["response.organization"] = org
	}
	for header, suffix := range llmResponseHeaders {
		if v := hdr.Get(header); v != "" {
			data["response.headers."+suffix] = v
		}
	}
}

// noticeError reports a failed completion.
func noticeError(txn *newrelic.Transaction, completionID string, err error) {
	attrs := map[string]interface{}{"completion_id": completionID}

	var apiErr genai.APIError
	if errors.As(err, &apiErr) {
		if apiErr.Code != 0 {
			attrs["http.statusCode"] = apiErr.Code
		}
		if apiErr.Status != "" {
			attrs["error.code"] = apiErr.Status
		}
	}

	txn.NoticeError(newrelic.Error{
		Message:    err.Error(),
		Class:      errorClass,
		Attributes: attrs,
	})
}

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

// GenerateContent wraps client.Models.GenerateContent with New Relic
// instrumentation, recording LlmChatCompletionSummary and
// LlmChatCompletionMessage events. A transaction in ctx is used as-is; otherwise
// one named "GeminiGenerateContent" is started and ended here.
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

	md := txn.GetTraceMetadata()
	c := &completion{
		id:       uuid.New().String(),
		spanID:   md.SpanID,
		traceID:  md.TraceID,
		model:    model,
		contents: contents,
		config:   config,
	}

	seg := txn.StartSegment("Llm/completion/Gemini/GenerateContent")
	start := time.Now()
	resp, err := s.models.GenerateContent(ctx, model, contents, config)
	c.duration = time.Since(start).Milliseconds()
	seg.End()

	c.resp, c.err = resp, err
	if err != nil {
		noticeError(txn, c.id, err)
	}

	s.recordCompletion(c)
	return resp, err
}

// completion is the state one chat completion needs to report its events. The
// streaming and non-streaming paths both fill one in, so both emit the same
// attributes.
type completion struct {
	id       string
	spanID   string
	traceID  string
	model    string
	contents []*genai.Content
	config   *genai.GenerateContentConfig

	// resp is nil if the call failed; Gemini never returns both.
	resp *genai.GenerateContentResponse
	err  error

	duration int64
	// timeToFirstToken is streaming-only. A pointer, because a first chunk inside
	// the same millisecond measures zero, unlike a stream that produced nothing.
	timeToFirstToken *int64
}

// responseModel is the response's model, or the requested one if absent.
func (c *completion) responseModel() string {
	if c.resp != nil && c.resp.ModelVersion != "" {
		return c.resp.ModelVersion
	}
	return c.model
}

func (c *completion) headers() http.Header {
	if c.resp == nil || c.resp.SDKHTTPResponse == nil {
		return nil
	}
	return c.resp.SDKHTTPResponse.Headers
}

// requestID comes from the response headers, falling back to the body's response ID.
func (c *completion) requestID() string {
	if id := lookupHeader(c.headers(), requestIDHeaders...); id != "" {
		return id
	}
	if c.resp != nil {
		return c.resp.ResponseID
	}
	return ""
}

// messageID is "<response id>-<sequence>", or a UUID if there is no response ID.
func (c *completion) messageID(sequence int) string {
	if c.resp != nil && c.resp.ResponseID != "" {
		return fmt.Sprintf("%s-%d", c.resp.ResponseID, sequence)
	}
	return uuid.New().String()
}

// recordCompletion records the summary plus one message per request and response.
func (s *NRModelsService) recordCompletion(c *completion) {
	s.recordSummary(c)
	s.recordMessages(c)
}

func (s *NRModelsService) recordSummary(c *completion) {
	data := map[string]interface{}{
		"id":            c.id,
		"span_id":       c.spanID,
		"trace_id":      c.traceID,
		"request.model": c.model,
		"vendor":        vendorGemini,
		"ingest_source": ingestSource,
		"duration":      c.duration,
	}
	if id := c.requestID(); id != "" {
		data["request_id"] = id
	}
	if c.config != nil {
		if c.config.MaxOutputTokens != 0 {
			data["request.max_tokens"] = c.config.MaxOutputTokens
		}
		if c.config.Temperature != nil {
			data["request.temperature"] = *c.config.Temperature
		}
	}
	if c.timeToFirstToken != nil {
		data["time_to_first_token"] = *c.timeToFirstToken
	}
	if c.err != nil {
		data["error"] = true
	}

	if c.resp == nil {
		data["response.number_of_messages"] = len(c.contents)
	} else {
		if model := c.resp.ModelVersion; model != "" {
			data["response.model"] = model
		}
		// The attribute is singular, so the first candidate reporting a reason wins.
		for _, candidate := range c.resp.Candidates {
			if candidate != nil && candidate.FinishReason != "" {
				data["response.choices.finish_reason"] = string(candidate.FinishReason)
				break
			}
		}
		// Each candidate is a response message; CandidateCount can exceed 1.
		data["response.number_of_messages"] = len(c.contents) + len(c.resp.Candidates)
		if u := c.resp.UsageMetadata; u != nil {
			if u.PromptTokenCount != 0 {
				data["response.usage.prompt_tokens"] = u.PromptTokenCount
			}
			if u.CandidatesTokenCount != 0 {
				data["response.usage.completion_tokens"] = u.CandidatesTokenCount
			}
			if u.TotalTokenCount != 0 {
				data["response.usage.total_tokens"] = u.TotalTokenCount
			}
		}
		addHeaderAttrs(data, c.headers())
	}

	s.appendCustomAttrs(data)
	s.app.RecordCustomEvent("LlmChatCompletionSummary", data)
}

func (s *NRModelsService) recordMessages(c *completion) {
	cfg, _ := s.app.Config()
	recordContent := cfg.AIMonitoring.RecordContent.Enabled
	responseModel := c.responseModel()
	requestID := c.requestID()

	sequence := 0
	record := func(role, text string, skipped int, isResponse bool) {
		data := map[string]interface{}{
			"id":             c.messageID(sequence),
			"span_id":        c.spanID,
			"trace_id":       c.traceID,
			"role":           role,
			"completion_id":  c.id,
			"sequence":       sequence,
			"response.model": responseModel,
			"vendor":         vendorGemini,
			"ingest_source":  ingestSource,
		}
		if requestID != "" {
			data["request_id"] = requestID
		}
		if isResponse {
			data["is_response"] = true
		}
		if recordContent && text != "" {
			data["content"] = text
		}
		if tokens, ok := s.app.InvokeLLMTokenCountCallback(responseModel, text); ok {
			data["token_count"] = tokens
		}
		s.appendCustomAttrs(data)
		s.app.RecordCustomEvent("LlmChatCompletionMessage", data)

		if skipped > 0 && cfg.Logger != nil && cfg.Logger.DebugEnabled() {
			cfg.Logger.Debug("nrgemini: message parts omitted from content", map[string]interface{}{
				"completion_id": c.id,
				"sequence":      sequence,
				"parts":         skipped,
			})
		}
		sequence++
	}

	for _, content := range c.contents {
		text, skipped := contentText(content)
		record(contentRole(content), text, skipped, false)
	}

	if c.resp == nil {
		return
	}
	for _, candidate := range c.resp.Candidates {
		text, skipped := candidateText(candidate)
		record(candidateRole(candidate), text, skipped, true)
	}
}

func (s *NRModelsService) appendCustomAttrs(data map[string]interface{}) {
	maps.Copy(data, s.customAttributes)
}

// contentRole defaults to "user" per the spec.
func contentRole(c *genai.Content) string {
	if c != nil && c.Role != "" {
		return c.Role
	}
	return genai.RoleUser
}

// candidateRole passes Gemini's "model" through, defaulting to "assistant" per the spec.
func candidateRole(c *genai.Candidate) string {
	if c != nil && c.Content != nil && c.Content.Role != "" {
		return c.Content.Role
	}
	return roleAssistant
}

// contentText joins a message's text parts. The count reports parts left out
// because they carry data other than text: blobs, file references, function calls
// or responses, executable code.
func contentText(c *genai.Content) (text string, skipped int) {
	if c == nil {
		return "", 0
	}
	var texts []string
	for _, p := range c.Parts {
		switch {
		case p == nil:
		case p.Text != "":
			texts = append(texts, p.Text)
		default:
			skipped++
		}
	}
	return strings.Join(texts, " "), skipped
}

func candidateText(c *genai.Candidate) (string, int) {
	if c == nil {
		return "", 0
	}
	return contentText(c.Content)
}

// NRGenerateContentStream tracks a streaming GenerateContent call until every
// chunk has been processed. The SDK hands back an iter.Seq2, so your code drives
// the loop and reports each chunk back:
//
//	stream := nrClient.Models.GenerateContentStream(ctx, model, contents, cfg)
//	defer stream.Close()
//	for chunk, err := range stream.Stream {
//	    stream.RecordEvent(chunk, err)
//	    if err != nil {
//	        break
//	    }
//	    // ... consume chunk ...
//	}
//
// Ranging over Stream is what releases the HTTP body; the SDK closes it when the
// loop ends or breaks. Close only finishes the New Relic side, so it is safe to
// defer. See ProcessContentStream for an easier alternative.
type NRGenerateContentStream struct {
	// Stream is the SDK's iterator. Range over it directly.
	Stream iter.Seq2[*genai.GenerateContentResponse, error]

	svc      *NRModelsService
	txn      *newrelic.Transaction
	seg      *newrelic.Segment
	closeTxn bool
	closed   bool

	// completion holds what Close reports; its resp is accumulated chunk by chunk
	// into the shape a non-streaming call returns.
	completion *completion
	// text collects each candidate's deltas. A Builder per candidate keeps the
	// accumulation linear; concatenating onto the response recopies it every chunk.
	text  []*strings.Builder
	start time.Time
}

// GenerateContentStream wraps client.Models.GenerateContentStream with New Relic
// instrumentation. Report each chunk with RecordEvent and call Close when done;
// see NRGenerateContentStream for an example. A transaction in ctx is used as-is,
// otherwise one named "GeminiGenerateContentStream" is started and ended by
// Close. With AI monitoring or streaming disabled, Stream is the untouched SDK
// iterator and RecordEvent/Close do nothing.
func (s *NRModelsService) GenerateContentStream(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) *NRGenerateContentStream {
	cfg, _ := s.app.Config()

	if !cfg.AIMonitoring.Streaming.Enabled {
		if reportStreamingDisabled != nil {
			reportStreamingDisabled()
		}
	}

	stream := &NRGenerateContentStream{
		svc:   s,
		start: time.Now(),
		completion: &completion{
			model:    model,
			contents: contents,
			config:   config,
		},
	}

	if cfg.AIMonitoring.Enabled && cfg.AIMonitoring.Streaming.Enabled {
		stream.txn = newrelic.FromContext(ctx)
		if stream.txn == nil {
			stream.txn = s.app.StartTransaction("GeminiGenerateContentStream")
			stream.closeTxn = true
			ctx = newrelic.NewContext(ctx, stream.txn)
		}
		integrationsupport.AddAgentAttribute(stream.txn, "llm", "", true)

		md := stream.txn.GetTraceMetadata()
		stream.completion.id = uuid.New().String()
		stream.completion.spanID = md.SpanID
		stream.completion.traceID = md.TraceID
		stream.seg = stream.txn.StartSegment("Llm/completion/Gemini/GenerateContentStream")
	}

	// Last, so the request sees the transaction added to ctx above.
	stream.Stream = s.models.GenerateContentStream(ctx, model, contents, config)
	return stream
}

// RecordEvent reports one chunk from Stream. Pass the chunk and error exactly as
// the iterator yielded them.
func (w *NRGenerateContentStream) RecordEvent(chunk *genai.GenerateContentResponse, err error) {
	if w == nil {
		return
	}
	if err != nil {
		// Keep the first; the SDK may yield more chunks after an error.
		if w.completion.err == nil {
			w.completion.err = err
			w.completion.duration = time.Since(w.start).Milliseconds()
			if w.txn != nil {
				noticeError(w.txn, w.completion.id, err)
			}
		}
		return
	}
	if w.txn == nil || chunk == nil {
		return
	}
	if w.completion.timeToFirstToken == nil {
		elapsed := time.Since(w.start).Milliseconds()
		w.completion.timeToFirstToken = &elapsed
	}
	w.completion.duration = time.Since(w.start).Milliseconds()
	w.accumulate(chunk)
}

// accumulate folds a chunk into the response Close reports on. Deltas append to a
// single Part, so no separator lands between them.
func (w *NRGenerateContentStream) accumulate(chunk *genai.GenerateContentResponse) {
	if w.completion.resp == nil {
		w.completion.resp = &genai.GenerateContentResponse{}
	}
	resp := w.completion.resp

	if chunk.ResponseID != "" {
		resp.ResponseID = chunk.ResponseID
	}
	if chunk.ModelVersion != "" {
		resp.ModelVersion = chunk.ModelVersion
	}
	// Usage arrives on the final chunk, so the latest wins.
	if chunk.UsageMetadata != nil {
		resp.UsageMetadata = chunk.UsageMetadata
	}
	if chunk.SDKHTTPResponse != nil {
		resp.SDKHTTPResponse = chunk.SDKHTTPResponse
	}

	// Each requested candidate streams its own deltas.
	for len(resp.Candidates) < len(chunk.Candidates) {
		resp.Candidates = append(resp.Candidates, &genai.Candidate{
			Content: &genai.Content{Parts: []*genai.Part{{}}},
		})
		w.text = append(w.text, &strings.Builder{})
	}

	for i, src := range chunk.Candidates {
		if src == nil {
			continue
		}
		dst := resp.Candidates[i]
		if src.FinishReason != "" {
			dst.FinishReason = src.FinishReason
		}
		if src.Content == nil {
			continue
		}
		if src.Content.Role != "" {
			dst.Content.Role = src.Content.Role
		}
		for _, p := range src.Content.Parts {
			if p != nil && p.Text != "" {
				w.text[i].WriteString(p.Text)
			}
		}
	}
}

// response writes the accumulated text into the response and returns it.
func (w *NRGenerateContentStream) response() *genai.GenerateContentResponse {
	resp := w.completion.resp
	if resp == nil {
		return nil
	}
	for i, b := range w.text {
		if i < len(resp.Candidates) {
			resp.Candidates[i].Content.Parts[0].Text = b.String()
		}
	}
	return resp
}

// Err returns the first error the stream yielded.
func (w *NRGenerateContentStream) Err() error {
	if w == nil {
		return nil
	}
	return w.completion.err
}

// Close records the events, ends the segment, and ends the transaction if
// GenerateContentStream started one. Idempotent, and a no-op when uninstrumented.
func (w *NRGenerateContentStream) Close() error {
	if w == nil || w.closed {
		return nil
	}
	w.closed = true
	if w.txn == nil {
		return nil
	}

	if w.seg != nil {
		w.seg.End()
	}

	// A failed stream reports request messages only, like the non-streaming path.
	if w.completion.err != nil {
		w.completion.resp = nil
	}
	w.response()
	// duration is stamped as chunks arrive, so a Close that runs much later — a
	// deferred one, say — does not stretch it. Only a stream nothing was reported
	// for needs it measured here.
	if w.completion.duration == 0 {
		w.completion.duration = time.Since(w.start).Milliseconds()
	}
	w.svc.recordCompletion(w.completion)

	if w.closeTxn {
		w.txn.End()
	}
	return nil
}

// ProcessContentStream is GenerateContentStream with the loop and Close handled
// for you, invoking callback for each chunk. A callback error stops the stream and
// is returned; otherwise the stream's own error is.
func (s *NRModelsService) ProcessContentStream(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig, callback func(*genai.GenerateContentResponse) error) error {
	stream := s.GenerateContentStream(ctx, model, contents, config)
	defer stream.Close()

	var userErr error
	for chunk, err := range stream.Stream {
		stream.RecordEvent(chunk, err)
		if err != nil {
			break
		}
		if userErr = callback(chunk); userErr != nil {
			break
		}
	}

	if userErr != nil {
		return userErr
	}
	return stream.Err()
}
