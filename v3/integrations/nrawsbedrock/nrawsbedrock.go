// Copyright New Relic, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package nrawsbedrock instruments AI model invocation requests made by the
// https://github.com/aws/aws-sdk-go-v2/service/bedrockruntime library.
//
// Specifically, this provides instrumentation for the InvokeModel and InvokeModelWithResponseStream
// bedrock client API library functions.
//
// To use this integration, enable the New Relic AIMonitoring configuration options
// in your application, import this integration, and use the model invocation calls
// from this library in place of the corresponding ones from the AWS Bedrock
// runtime library, as documented below.
//
// The relevant configuration options are passed to the NewApplication function and include
//    ConfigAIMonitoringEnabled(true),  // enable (or disable if false) this integration
//    ConfigAIMonitoringStreamingEnabled(true), // enable instrumentation of streaming invocations
//    ConfigAIMonitoringRecordContentEnabled(true), // include input/output data in instrumentation
//
// Or, if ConfigFromEnvironment() is included in your configuration options, the above configuration
// options may be specified using these environment variables, respectively:
//    NEW_RELIC_AI_MONITORING_ENABLED=true
//    NEW_RELIC_AI_MONITORING_STREAMING_ENABLED=true
//    NEW_RELIC_AI_MONITORING_RECORD_CONTENT_ENABLED=true
// The values for these variables may be any form accepted by strconv.ParseBool (e.g., 1, t, T, true, TRUE, True,
// 0, f, F, false, FALSE, or False).
//
// See example/main.go for a working sample.
package nrawsbedrock

import (
	"context"
	"encoding/json"
	"errors"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/google/uuid"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var (
	reportStreamingDisabled func()
	ErrMissingResponseData  = errors.New("missing response data")
)

func init() {
	reportStreamingDisabled = sync.OnceFunc(func() {
		internal.TrackUsage("Go", "ML", "Streaming", "Disabled")
	})

	// Get the version of the AWS Bedrock library we're using
	info, ok := debug.ReadBuildInfo()
	if info != nil && ok {
		for _, module := range info.Deps {
			if module != nil && strings.Contains(module.Path, "/aws/aws-sdk-go-v2/service/bedrockruntime") {
				internal.TrackUsage("Go", "ML", "Bedrock", module.Version)
				return
			}
		}
	}
	internal.TrackUsage("Go", "ML", "Bedrock", "unknown")
}

//
// isEnabled determines if AI Monitoring is enabled in the app's options.
// It returns true if we should proceed with instrumentation. Additionally,
// it sets the Go/ML/Streaming/Disabled supportability metric if we discover
// that streaming is disabled, but ONLY does so the first time we try. Since
// we need to initialize the app and load options before we know if that one
// gets sent, we have to wait until later on to report that.
//
// streaming indicates if you're asking if it's ok to instrument streaming calls.
// The return values are two booleans: the first indicates if AI instrumentation
// is enabled at all, the second tells if it is permitted to record request and
// response data (as opposed to just metadata).
//
func isEnabled(app *newrelic.Application, streaming bool) (bool, bool) {
	if app == nil {
		return false, false
	}
	config, _ := app.Config()
	if !config.AIMonitoring.Streaming.Enabled {
		if reportStreamingDisabled != nil {
			reportStreamingDisabled()
		}
		if streaming {
			// we asked for streaming but it's not enabled
			return false, false
		}
	}

	return config.AIMonitoring.Enabled, config.AIMonitoring.RecordContent.Enabled
}

// ResponseStream tracks the model invocation throughout its lifetime until all stream events
// are processed.
type ResponseStream struct {
	// The request parameters that started the invocation
	ctx                  context.Context
	app                  *newrelic.Application
	client               *bedrockruntime.Client
	params               *bedrockruntime.InvokeModelWithResponseStreamInput
	attrs                map[string]any
	recordContentEnabled bool
	closeTxn             bool
	txn                  *newrelic.Transaction
	seg                  *newrelic.Segment
	completionID         string
	seq                  int

	// The model output
	Response *bedrockruntime.InvokeModelWithResponseStreamOutput
}

type modelResultList struct {
	output           string
	completionReason string
	tokenCount       int
}

type modelInputList struct {
	input      string
	role       string
	tokenCount int
}

//
// InvokeModelWithResponseStream invokes a model but unlike the InvokeModel method, the data returned
// is a stream of multiple events instead of a single response value.
// This function is the analogue of the bedrockruntime library InvokeModelWithResponseStream function,
// so that, given a bedrockruntime.Client b, where you would normally call the AWS method
//    response, err := b.InvokeModelWithResponseStream(c, p, f...)
// You instead invoke the New Relic InvokeModelWithResponseStream function as:
//    rstream, err := nrbedrock.InvokeModelWithResponseStream(app, b, c, p, f...)
// where app is your New Relic Application value.
//
// If using the bedrockruntime library directly, you would then process the response stream value
// (the response variable in the above example), iterating over the provided channel where the stream
// data appears until it is exhausted, and then calling Close() on the stream (see the bedrock API
// documentation for details).
//
// When using the New Relic nrawsbedrock integration, this response value is available as
// rstream.Response. You would perform the same operations as you would directly with the bedrock API
// once you have that value.
// Since this means control has passed back to your code for processing of the stream data, you need to
// add instrumentation calls to your processing code:
//    rstream.RecordEvent(content)   // for each event received from the stream
//    rstream.Close()                // when you are finished and are going to close the stream
//
// However, see ProcessModelWithResponseStream for an easier alternative.
//
// Either start a transaction on your own and add it to the context c  passed into this function, or
// a transaction will be started for you that lasts only for the duration of the model invocation.
//
func InvokeModelWithResponseStream(app *newrelic.Application, brc *bedrockruntime.Client, ctx context.Context, params *bedrockruntime.InvokeModelWithResponseStreamInput, optFns ...func(*bedrockruntime.Options)) (ResponseStream, error) {
	return InvokeModelWithResponseStreamAttributes(app, brc, ctx, params, nil, optFns...)
}

//
// InvokeModelWithResponseStreamAttributes is identical to InvokeModelWithResponseStream except that
// it adds the attrs parameter, which is a
// map of strings to values of any type. This map holds any custom attributes you wish to add to the reported metrics
// relating to this model invocation.
//
// Each key in the attrs map must begin with "llm."; if any of them do not, "llm." is automatically prepended to
// the attribute key before the metrics are sent out.
//
// We recommend including at least "llm.conversation_id" in your attributes.
//
func InvokeModelWithResponseStreamAttributes(app *newrelic.Application, brc *bedrockruntime.Client, ctx context.Context, params *bedrockruntime.InvokeModelWithResponseStreamInput, attrs map[string]any, optFns ...func(*bedrockruntime.Options)) (ResponseStream, error) {
	var aiEnabled bool
	var err error

	resp := ResponseStream{
		ctx:    ctx,
		app:    app,
		client: brc,
		params: params,
		attrs:  attrs,
	}

	aiEnabled, resp.recordContentEnabled = isEnabled(app, true)
	if aiEnabled {
		resp.txn = newrelic.FromContext(ctx)
		if resp.txn == nil {
			resp.txn = app.StartTransaction("InvokeModelWithResponseStream")
			resp.closeTxn = true
		}
	}

	if resp.txn != nil {
		integrationsupport.AddAgentAttribute(resp.txn, "llm", "", true)
		if params.ModelId != nil {
			resp.seg = resp.txn.StartSegment("Llm/completion/Bedrock/InvokeModelWithResponseStream")
		} else {
			// we don't have a model!
			resp.txn = nil
		}
	}

	start := time.Now()
	resp.Response, err = brc.InvokeModelWithResponseStream(ctx, params, optFns...)
	duration := time.Since(start).Milliseconds()

	if resp.txn != nil {
		md := resp.txn.GetTraceMetadata()
		resp.completionID = uuid.New().String()
		meta := map[string]any{
			"id":             resp.completionID,
			"span_id":        md.SpanID,
			"trace_id":       md.TraceID,
			"request.model":  *params.ModelId,
			"response.model": *params.ModelId,
			"vendor":         "bedrock",
			"ingest_source":  "Go",
			"duration":       duration,
		}

		if err != nil {
			resp.txn.NoticeError(newrelic.Error{
				Message: err.Error(),
				Class:   "BedrockError",
				Attributes: map[string]any{
					"completion_id": resp.completionID,
				},
			})
			meta["error"] = true
		}

		var modelInput []byte
		if params != nil && params.Body != nil {
			modelInput = params.Body
		}

		inputs, outputs, systemMessage := parseModelData(app, *params.ModelId, meta, modelInput, nil, attrs)
		// To be more runtime efficient, we don't copy the maps or rebuild them for each kind of message.
		// Instead, we build one map with most of the attributes common to all messages and then adjust as needed
		// when reporting out each metric.

		app.RecordCustomEvent("LlmChatCompletionSummary", meta)
		delete(meta, "duration")
		meta["completion_id"] = meta["id"]
		delete(meta, "id")

		if systemMessage != "" {
			meta["sequence"] = resp.seq
			resp.seq++
			meta["role"] = "system"
			if resp.recordContentEnabled {
				meta["content"] = systemMessage
			}
			app.RecordCustomEvent("LlmChatCompletionMessage", meta)
		}

		meta["role"] = "user"
		for _, msg := range inputs {
			meta["sequence"] = resp.seq
			resp.seq++
			if msg.tokenCount > 0 {
				meta["token_count"] = msg.tokenCount
			} else {
				delete(meta, "token_count")
			}
			if resp.recordContentEnabled {
				meta["content"] = msg.input
			} else {
				delete(meta, "content")
			}
			app.RecordCustomEvent("LlmChatCompletionMessage", meta)
		}
		for _, msg := range outputs {
			meta["sequence"] = resp.seq
			resp.seq++
			if msg.tokenCount > 0 {
				meta["token_count"] = msg.tokenCount
			} else {
				delete(meta, "token_count")
			}
			if resp.recordContentEnabled {
				meta["content"] = msg.output
			} else {
				delete(meta, "content")
			}
			app.RecordCustomEvent("LlmChatCompletionMessage", meta)
		}
	}
	return resp, nil
}

//
// RecordEvent records a single stream event as read from the data stream started by InvokeModelWithStreamResponse.
//
func (s *ResponseStream) RecordEvent(data []byte) error {
	if s == nil || s.txn == nil || s.app == nil {
		return nil
	}
	if s.params == nil || s.params.ModelId == nil {
		return ErrMissingResponseData
	}

	md := s.txn.GetTraceMetadata()

	meta := map[string]any{
		"completion_id":  s.completionID,
		"span_id":        md.SpanID,
		"trace_id":       md.TraceID,
		"request.model":  *s.params.ModelId,
		"response.model": *s.params.ModelId,
		"vendor":         "bedrock",
		"ingest_source":  "Go",
		"role":           "assistant",
	}

	_, outputs, _ := parseModelData(s.app, *s.params.ModelId, meta, s.params.Body, data, s.attrs)

	for _, msg := range outputs {
		meta["sequence"] = s.seq
		s.seq++
		if msg.tokenCount > 0 {
			meta["token_count"] = msg.tokenCount
		} else {
			delete(meta, "token_count")
		}
		if s.recordContentEnabled {
			meta["content"] = msg.output
		} else {
			delete(meta, "content")
		}
		s.app.RecordCustomEvent("LlmChatCompletionMessage", meta)
	}
	return nil
}

//
// Close finishes up the instrumentation for a response stream.
//
func (s *ResponseStream) Close() error {
	if s == nil || s.txn == nil {
		return nil
	}

	if s.seg != nil {
		s.seg.End()
	}
	if s.closeTxn {
		s.txn.End()
	}
	return nil
}

//
// ProcessModelWithResponseStream works just like InvokeModelWithResponseStream, except that
// it handles all the stream processing automatically for you. For each event received from
// the response stream, it will invoke the callback function you pass into the function call
// so that your application can act on the response data. When the stream is complete, the
// ProcessModelWithResponseStream call will return.
//
// If your callback function returns an error, the processing of the response stream will
// terminate at that point.
//
func ProcessModelWithResponseStream(app *newrelic.Application, brc *bedrockruntime.Client, ctx context.Context, callback func([]byte) error, params *bedrockruntime.InvokeModelWithResponseStreamInput, optFns ...func(*bedrockruntime.Options)) error {
	return ProcessModelWithResponseStreamAttributes(app, brc, ctx, callback, params, nil, optFns...)
}

//
// ProcessModelWithResponseStreamAttributes is identical to ProcessModelWithResponseStream except that
// it adds the attrs parameter, which is a
// map of strings to values of any type. This map holds any custom attributes you wish to add to the reported metrics
// relating to this model invocation.
//
// Each key in the attrs map must begin with "llm."; if any of them do not, "llm." is automatically prepended to
// the attribute key before the metrics are sent out.
//
// We recommend including at least "llm.conversation_id" in your attributes.
//
func ProcessModelWithResponseStreamAttributes(app *newrelic.Application, brc *bedrockruntime.Client, ctx context.Context, callback func([]byte) error, params *bedrockruntime.InvokeModelWithResponseStreamInput, attrs map[string]any, optFns ...func(*bedrockruntime.Options)) error {
	var err error
	var userErr error

	response, err := InvokeModelWithResponseStreamAttributes(app, brc, ctx, params, attrs, optFns...)
	if err != nil {
		return err
	}
	if response.Response == nil {
		return response.Close()
	}

	stream := response.Response.GetStream()
	defer func() {
		err = stream.Close()
	}()

	for event := range stream.Events() {
		if v, ok := event.(*types.ResponseStreamMemberChunk); ok {
			if userErr = callback(v.Value.Bytes); userErr != nil {
				break
			}
			response.RecordEvent(v.Value.Bytes)
		}
	}

	err = response.Close()
	if userErr != nil {
		return userErr
	}
	return err
}

//
// InvokeModel provides an instrumented interface through which to call the AWS Bedrock InvokeModel function.
// Where you would normally invoke the InvokeModel method on a bedrockruntime.Client value b from AWS as:
//    b.InvokeModel(c, p, f...)
// You instead invoke the New Relic InvokeModel function as:
//    nrbedrock.InvokeModel(app, b, c, p, f...)
// where app is the New Relic Application value returned from NewApplication when you started
// your application. If you start a transaction and add it to the passed context value c in the above
// invocation, the instrumentation will be recorded on that transaction, including a segment for the Bedrock
// call itself. If you don't, a new transaction will be started for you, which will be terminated when the
// InvokeModel function exits.
//
// If the transaction is unable to be created or used, the Bedrock call will be made anyway, without instrumentation.
//
func InvokeModel(app *newrelic.Application, brc *bedrockruntime.Client, ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
	return InvokeModelWithAttributes(app, brc, ctx, params, nil, optFns...)
}

//
// InvokeModelWithAttributes is identical to InvokeModel except for the addition of the attrs parameter, which is a
// map of strings to values of any type. This map holds any custom attributes you wish to add to the reported metrics
// relating to this model invocation.
//
// Each key in the attrs map must begin with "llm."; if any of them do not, "llm." is automatically prepended to
// the attribute key before the metrics are sent out.
//
// We recommend including at least "llm.conversation_id" in your attributes.
//
func InvokeModelWithAttributes(app *newrelic.Application, brc *bedrockruntime.Client, ctx context.Context, params *bedrockruntime.InvokeModelInput, attrs map[string]any, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
	var txn *newrelic.Transaction // the transaction to record in, or nil if we aren't instrumenting this time

	aiEnabled, recordContentEnabled := isEnabled(app, false)
	if aiEnabled {
		txn = newrelic.FromContext(ctx)
		if txn == nil {
			if txn = app.StartTransaction("InvokeModel"); txn != nil {
				defer txn.End()
			}
		}
	}

	var embedding bool
	id_key := "completion_id"

	if txn != nil {
		integrationsupport.AddAgentAttribute(txn, "llm", "", true)
		if params.ModelId != nil {
			if embedding = strings.Contains(*params.ModelId, "embed"); embedding {
				defer txn.StartSegment("Llm/embedding/Bedrock/InvokeModel").End()
				id_key = "embedding_id"
			} else {
				defer txn.StartSegment("Llm/completion/Bedrock/InvokeModel").End()
			}
		} else {
			// we don't have a model!
			txn = nil
		}
	}

	start := time.Now()
	output, err := brc.InvokeModel(ctx, params, optFns...)
	duration := time.Since(start).Milliseconds()

	if txn != nil {
		md := txn.GetTraceMetadata()
		uuid := uuid.New()
		meta := map[string]any{
			"id":             uuid.String(),
			"span_id":        md.SpanID,
			"trace_id":       md.TraceID,
			"request.model":  *params.ModelId,
			"response.model": *params.ModelId,
			"vendor":         "bedrock",
			"ingest_source":  "Go",
			"duration":       duration,
		}

		if err != nil {
			txn.NoticeError(newrelic.Error{
				Message: err.Error(),
				Class:   "BedrockError",
				Attributes: map[string]any{
					id_key: uuid.String(),
				},
			})
			meta["error"] = true
		}

		var modelInput, modelOutput []byte
		if params != nil && params.Body != nil {
			modelInput = params.Body
		}
		if output != nil && output.Body != nil {
			modelOutput = output.Body
		}

		inputs, outputs, systemMessage := parseModelData(app, *params.ModelId, meta, modelInput, modelOutput, attrs)
		// To be more runtime efficient, we don't copy the maps or rebuild them for each kind of message.
		// Instead, we build one map with most of the attributes common to all messages and then adjust as needed
		// when reporting out each metric.

		if embedding {
			for _, theInput := range inputs {
				if theInput.tokenCount > 0 {
					meta["token_count"] = theInput.tokenCount
				} else {
					delete(meta, "token_count")
				}
				if recordContentEnabled && theInput.input != "" {
					meta["input"] = theInput.input
				} else {
					delete(meta, "input")
				}
				app.RecordCustomEvent("LlmEmbedding", meta)
			}
		} else {
			messageQty := len(inputs) + len(outputs)
			messageSeq := 0
			if systemMessage != "" {
				messageQty++
			}

			meta["response.number_of_messages"] = messageQty
			app.RecordCustomEvent("LlmChatCompletionSummary", meta)
			delete(meta, "duration")
			meta["completion_id"] = meta["id"]
			delete(meta, "id")
			delete(meta, "response.number_of_messages")

			if systemMessage != "" {
				meta["sequence"] = messageSeq
				messageSeq++
				meta["role"] = "system"
				if recordContentEnabled {
					meta["content"] = systemMessage
				}
				app.RecordCustomEvent("LlmChatCompletionMessage", meta)
			}

			maxIterations := len(inputs)
			if maxIterations < len(outputs) {
				maxIterations = len(outputs)
			}
			for i := 0; i < maxIterations; i++ {
				if i < len(inputs) {
					meta["sequence"] = messageSeq
					messageSeq++
					if inputs[i].tokenCount > 0 {
						meta["token_count"] = inputs[i].tokenCount
					} else {
						delete(meta, "token_count")
					}
					if recordContentEnabled {
						meta["content"] = inputs[i].input
					} else {
						delete(meta, "content")
					}
					delete(meta, "is_response")
					delete(meta, "response.choices.finish_reason")
					meta["role"] = "user"
					app.RecordCustomEvent("LlmChatCompletionMessage", meta)
				}
				if i < len(outputs) {
					meta["sequence"] = messageSeq
					messageSeq++
					if outputs[i].tokenCount > 0 {
						meta["token_count"] = outputs[i].tokenCount
					} else {
						delete(meta, "token_count")
					}
					if recordContentEnabled {
						meta["content"] = outputs[i].output
					} else {
						delete(meta, "content")
					}
					meta["role"] = "assistant"
					meta["is_response"] = true
					if outputs[i].completionReason != "" {
						meta["response.choices.finish_reason"] = outputs[i].completionReason
					} else {
						delete(meta, "response.choices.finish_reason")
					}
					app.RecordCustomEvent("LlmChatCompletionMessage", meta)
				}
			}
		}
	}
	return output, nil
}

func parseModelData(app *newrelic.Application, modelID string, meta map[string]any, modelInput, modelOutput []byte, attrs map[string]any) ([]modelInputList, []modelResultList, string) {
	inputs := []modelInputList{}
	outputs := []modelResultList{}

	// Go fishing in the request and response JSON strings to find values we want to
	// record with our instrumentation. Since each model can define its own set of
	// expected input and output data formats, we either have to specifically define
	// model-specific templates or try to heuristically find our values in the places
	// we'd expect given the existing patterns shown in the model set we have today.
	//
	// This implementation takes the latter approach so as to be as flexible as possible
	// and have a good chance to find the data we're looking for even in new models
	// that follow the same general pattern as those models that came before them.
	//
	// Thanks to the fact that the input and output can be a JSON data structure
	// of literally anything, there's a lot of type assertion shenanigans going on
	// below, as we unmarshal the JSON into a map[string]any at the top level, and
	// then explore the "any" values on the way down, asserting them to be the actual
	// expected types as needed.

	var requestData, responseData map[string]any
	var systemMessage string

	if modelInput != nil && json.Unmarshal(modelInput, &requestData) == nil {
		// if the input contains a messages list, we have multiple messages to record
		if rs, ok := requestData["messages"]; ok {
			if rss, ok := rs.([]any); ok {
				for _, em := range rss {
					if eachMessage, ok := em.(map[string]any); ok {
						var role string
						if r, ok := eachMessage["role"]; ok {
							role, _ = r.(string)
						}
						if cs, ok := eachMessage["content"]; ok {
							if css, ok := cs.([]any); ok {
								for _, ec := range css {
									if eachContent, ok := ec.(map[string]any); ok {
										if ty, ok := eachContent["type"]; ok {
											if typ, ok := ty.(string); ok && typ == "text" {
												if txt, ok := eachContent["text"]; ok {
													if txts, ok := txt.(string); ok {
														inputs = append(inputs, modelInputList{input: txts, role: role})
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
		if sys, ok := requestData["system"]; ok {
			systemMessage, _ = sys.(string)
		}

		// otherwise, look for what the single or multiple prompt input is called
		var inputString string
		if s, ok := requestData["inputText"]; ok {
			inputString, _ = s.(string)
		} else if s, ok := requestData["prompt"]; ok {
			inputString, _ = s.(string)
		} else if ss, ok := requestData["texts"]; ok {
			if slist, ok := ss.([]string); ok {
				for _, inpStr := range slist {
					inputs = append(inputs, modelInputList{input: inpStr, role: "user"})
				}
			}
		}
		if inputString != "" {
			inputs = append(inputs, modelInputList{input: inputString, role: "user"})
		}

		if cfg, ok := requestData["textGenerationConfig"]; ok {
			if cfgMap, ok := cfg.(map[string]any); ok {
				if t, ok := cfgMap["temperature"]; ok {
					meta["request.temperature"] = t
				}
				if m, ok := cfgMap["maxTokenCount"]; ok {
					meta["request.max_tokens"] = m
				}
			}
		} else if t, ok := requestData["temperature"]; ok {
			meta["request.temperature"] = t
		}
		if m, ok := requestData["max_tokens_to_sample"]; ok {
			meta["request.max_tokens"] = m
		} else if m, ok := requestData["max_tokens"]; ok {
			meta["request.max_tokens"] = m
		} else if m, ok := requestData["maxTokens"]; ok {
			meta["request.max_tokens"] = m
		} else if m, ok := requestData["max_gen_len"]; ok {
			meta["request.max_tokens"] = m
		}
	}

	var stopReason string
	var outputString string
	if modelOutput != nil {
		if json.Unmarshal(modelOutput, &responseData) == nil {
			if len(inputs) == 0 {
				if s, ok := responseData["prompt"]; ok {
					if inpStr, ok := s.(string); ok {
						inputs = append(inputs, modelInputList{input: inpStr, role: "user"})
					}
				}
			}
			if id, ok := responseData["id"]; ok {
				meta["request_id"] = id
			}

			if s, ok := responseData["stop_reason"]; ok {
				stopReason, _ = s.(string)
			}

			if out, ok := responseData["completion"]; ok {
				outputString, _ = out.(string)
			}

			if rs, ok := responseData["results"]; ok {
				if crs, ok := rs.([]any); ok {
					for _, crv := range crs {
						if crvv, ok := crv.(map[string]any); ok {
							var stopR, outputS string
							if reason, ok := crvv["completionReason"]; ok {
								stopR, _ = reason.(string)
							}
							if out, ok := crvv["outputText"]; ok {
								outputS, _ = out.(string)
								outputs = append(outputs, modelResultList{output: outputS, completionReason: stopR})
							}
						}
					}
				}
			}
			//modelResultList{output: completionReason:}
			if rs, ok := responseData["completions"]; ok {
				if crs, ok := rs.([]any); ok {
					for _, crsv := range crs {
						if crv, ok := crsv.(map[string]any); ok {
							var outputR string

							if cdata, ok := crv["finishReason"]; ok {
								if cdatamap, ok := cdata.(map[string]any); ok {
									if reason, ok := cdatamap["reason"]; ok {
										outputR, _ = reason.(string)
									}
								}
							}
							if cdata, ok := crv["data"]; ok {
								if cdatamap, ok := cdata.(map[string]any); ok {
									if out, ok := cdatamap["text"]; ok {
										if outS, ok := out.(string); ok {
											outputs = append(outputs, modelResultList{output: outS, completionReason: outputR})
										}
									}
								}
							}
						}
					}
				}
			}
			if rs, ok := responseData["outputs"]; ok {
				if crs, ok := rs.([]any); ok {
					for _, crvv := range crs {
						if crv, ok := crvv.(map[string]any); ok {
							var stopR string
							if reason, ok := crv["stop_reason"]; ok {
								stopR, _ = reason.(string)
							}
							if out, ok := crv["text"]; ok {
								if outS, ok := out.(string); ok {
									outputs = append(outputs, modelResultList{output: outS, completionReason: stopR})
								}
							}
						}
					}
				}
			}
			if rs, ok := responseData["generations"]; ok {
				if crs, ok := rs.([]any); ok {
					for _, crvv := range crs {
						if crv, ok := crvv.(map[string]any); ok {
							var stopR string
							if reason, ok := crv["finish_reason"]; ok {
								stopR, _ = reason.(string)
							}
							if out, ok := crv["text"]; ok {
								if outS, ok := out.(string); ok {
									outputs = append(outputs, modelResultList{output: outS, completionReason: stopR})
								}
							}
						}
					}
				}
			}
			if outputString == "" {
				if out, ok := responseData["generation"]; ok {
					outputString, _ = out.(string)
				}
			}

			if outputString != "" {
				outputs = append(outputs, modelResultList{output: outputString, completionReason: stopReason})
			}
		}
	}

	if attrs != nil {
		for k, v := range attrs {
			if strings.HasPrefix(k, "llm.") {
				meta[k] = v
			} else {
				meta["llm."+k] = v
			}
		}
	}

	if app.HasLLMTokenCountCallback() {
		for i, _ := range inputs {
			if inputs[i].input != "" {
				inputs[i].tokenCount, _ = app.InvokeLLMTokenCountCallback(modelID, inputs[i].input)
			}
		}
		for i, _ := range outputs {
			if outputs[i].output != "" {
				outputs[i].tokenCount, _ = app.InvokeLLMTokenCountCallback(modelID, outputs[i].output)
			}
		}
	}

	return inputs, outputs, systemMessage
}

/***
We support:
	Anthropic Claude
		anthropic.claude-v2
		anthropic.claude-v2:1
		anthropic.claude-3-sonnet-...
		anthropic.claude-3-haiku-...
		anthropic.claude-instant-v1
	Amazon Titan
		amazon.titan-text-express-v1
		amazon.titan-text-lite-v1
E		amazon.titan-embed-text-v1
	Meta Llama 2
		meta.llama2-13b-chat-v1
		meta.llama2-70b-chat-v1
	Cohere Command
		cohere.command-text-v14
		cohere.command-light-text-v14
E		cohere.embed-english-v3
E		cohere.embed-multilingual-v3
			texts:[string]				embeddings:[1024 floats]
			input_type:s		=>		id:s
			truncate:s					response_type:s
										texts:[s]
	AI21 Labs Jurassic
		ai21.j2-mid-v1
		ai21.j2-ultra-v1

only text-based models
send LLM events as custom events ONLY when there is a transaction active
attrs limited to 4095 normally but LLM events are an exception to this. NO limits.
MAY limit other but MUST leave these unlimited:
	LlmChatCompletionMessage event, attr content
	LlmEmbedding             event, attr input

Events recorded:
	LlmEmbedding (creation of an embedding)
		id			UUID we generate
		request_id	from response headers usually
		span_id		GUID assoc'd with activespan
		trace_id	current trace ID
		input		input to the embedding creation call
		request.model	model name e.g. gpt-3.5-turbo
		response.model	model name returned in response
		response.organization	org ID returned in response or headers
		token_count				value from LLMTokenCountCallback or omitted
		vendor					"bedrock"
		ingest_source			"Go"
		duration				total time taken for chat completiong in mS
		error					true if error occurred or omitted
		llm.<user_defined_metadata>		**custom**
		response.headers.<vendor_specific_headers>	**response**
	LlmChatCompletionSummary (high-level data about creation of chat completion including request, response, and call info)
		id			UUID we generate
		request_id	from response headers usually
		span_id		GUID assoc'd with active span
		trace_id	current trace ID
		request.temperature	how random/deterministic output shoudl be
		request.max_tokens	max #tokens that can be generated
		request.model	model name e.g. gpt-3.5-turbo
		response.model	model name returned in response
		response.number_of_messages	number of msgs comprising completiong
		response.choices.finish_reason	reason model stopped (e.g. "stop")
		vendor					"bedrock"
		ingest_source			"Go"
		duration				total time taken for chat completiong in mS
		error					true if error occurred or omitted
		llm.<user_defined_metadata>		**custom**
		response.headers.<vendor_specific_headers>	**response**

	LlmChatCompletionMessage (each message sent/rec'd from chat completion call.
		id			UUID we generate OR <response_id>-<sequence> returned by LLM
		request_id	from response headers usually
		span_id		GUID assoc'd with active span
		trace_id	current trace ID
		??request.model	model name e.g. gpt-3.5-turbo
		response.model	model name returned in response
		vendor					"bedrock"
		ingest_source			"Go"
		content					content of msg
		role					role of msg creator
		sequence				index (0..) w/each msg including prompt and responses
		completion_id			ID of LlmChatCompletionSummary event that event is connected to
		is_response				true if msg is result of completion, not input msg OR omitted
		token_count				value from LLMTokenCountCallback or omitted
		llm.<user_defined_metadata>		**custom**

response.model = request.model if we don't get a response.model
custom attributes to LLM events have llm. prefix and this should be retained
llm.conversation_id

**custom**
user may add custom attributes to txn but we MUST strip out all that don't start with
"llm."
we recommend adding llm.conversation_id since that has UI implications

**response**
Capture response header values and add them as attributes to LLMEmbedding and
LLMChatCompletionSummary events as "response.headers.<header_name>" if present,
omit any that are not present.

OpenAI: llmVersion, ratelimitLimitRequests, ratelimitResetTokens, ratelimitLimitTokens,
ratelimitRemainingTokens, ratelimitRemainingRequests, ratelimitLimitTokensUsageBased,
ratelimitResetTokensUsageBased, ratelimitRemainingTokensUsageBased
Bedrock: ??

MUST add "llm: True" as agent attr to txn that contain instrumented LLM functions.
MUST be sent to txn events attr dest (DST_TRANSACTION_EVENTS). OMIT if there are no
LLM events in the txn.

MUST create span for each LLM embedding and chat completion call. MUST only be created
if there is a txn. MUST name them "Llm/completion|embedding/Bedrock/invoke_model|create|etc"

Errors -> notice_error
	http.statusCode, error.code (exception), error.param (exception), completion_id, embedding_id
	STILL create LlmChatCompletionSummary and LlmEmbedding events in error context
	with all attrs that can be captured, plus set error=true.


Supportability Metric
X	Supportability/Go/Bedrock/<vendor_lib_version>
X	Supportability/Go/ML/Streaming/Disabled		if !ai_monitoring.streaming.enabled

Config
	ai_monitoring.enabled
	ai_monitoring.streaming.enabled
	ai_monitoring.record_content.enabled
		If true, suppress
			LlmChatCompletionMessage.content
			LlmEmbedding.imput
			LlmTool.input
			LlmTool.output
			LlmVectorSearch.request.query
			LlmVectorSearchResult.page_content

Feedback
	tracked on trace ID
	API: getCurrentTraceID() or something to get the ID of the current active trace
	OR use pre-existing getLinkingMetadata to pull from map of returned data values
	**this means DT must be enabled to use feedback

	API: RecordLLMFeedbackEvent() -> custom event which includes end user feedback data
	API: LLMTokenCountCallback() to get the token count
		pass model name (string), content of message/prompt (string)
		receive integer count value -> token_count attr in LlmChatCompletionMessage or
		LlmEmbedding event UNLESS value <= 0, in which case ignore it.
	API: function to register the callback function, allowed to replace with a new one
		at any time.

New models mistral.mistral-7b-instruct-v0:2, mistral.mixtral-8x7b-instruct-v0:1 support?
 -> body looks like {
	  'prompt': <prompt engineering + question>,
	  'max_tokens': <optional | default 512>
	  'temperature': <optional | default 0.5>
	}

openai response headers include these but not always since they aren't always present
	ratelimitLimitTokensUsageBased
	ratelimitResetTokensUsageBased
	ratelimitRemainingTokensUsageBased


	ModelResultList
		Output
		CompletionReason
		TokenCount
	ModelInputList
		Role
		Input

amazon titan
	out:
		results[] outputText, completionReason
	stream:
		chunk/bytes/index, outputText, completionReason
Claude
	in:
		messages[] role, content[] type='text', text
		system: "system message"
	out:
		content[] type="text", text
		stop_reason
Cohere:
	out:
		generations[] finish_reason, id, text, index?
		id
		prompt
Mistral
	out:
		outputs[] text, stop_reason


***/
