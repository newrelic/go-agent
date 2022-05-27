// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	LogSeverityFieldName  = "level"
	LogMessageFieldName   = "message"
	LogTimestampFieldName = "timestamp"
	LogSpanIDFieldName    = "span.id"
	LogTraceIDFieldName   = "trace.id"
	LogSeverityUnknown    = "UNKNOWN"

	MaxLogLength = 32768
)

// for internal user only
type logEvent struct {
	priority  priority
	timestamp int64
	severity  string
	message   string
	spanID    string
	traceID   string
}

// LogData contains data fields that are needed to generate log events.
type LogData struct {
	Timestamp int64           // Required: Unix Millisecond Timestamp
	Severity  string          // Optional: Severity of log being consumed
	Message   string          // Optional: Message of log being consumed; Maximum size: 32768 Bytes.
	Context   context.Context // Optional: context containing a New Relic Transaction
}

// writeJSON prepares JSON in the format expected by the collector.
func (e *logEvent) WriteJSON(buf *bytes.Buffer) {
	w := jsonFieldsWriter{buf: buf}
	buf.WriteByte('{')
	w.stringField(LogSeverityFieldName, e.severity)
	w.stringField(LogMessageFieldName, e.message)

	if len(e.spanID) > 0 {
		w.stringField(LogSpanIDFieldName, e.spanID)
	}
	if len(e.traceID) > 0 {
		w.stringField(LogTraceIDFieldName, e.traceID)
	}

	w.needsComma = false
	buf.WriteByte(',')
	w.intField(LogTimestampFieldName, e.timestamp)
	buf.WriteByte('}')
}

// MarshalJSON is used for testing.
func (e *logEvent) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 256))
	e.WriteJSON(buf)
	return buf.Bytes(), nil
}

var (
	// regex allows a single word, or number
	severityUnknown = "UNKNOWN"

	errEmptyTimestamp     = errors.New("timestamp can not be empty")
	errNilLogData         = errors.New("log data can not be nil")
	errLogMessageTooLarge = fmt.Errorf("log message can not exceed %d bytes", MaxLogLength)
)

func (data *LogData) ToLogEvent() (*logEvent, error) {
	if data == nil {
		return nil, errNilLogData
	}
	if data.Severity == "" {
		data.Severity = LogSeverityUnknown
	}
	if len(data.Message) > MaxLogLength {
		return nil, errLogMessageTooLarge
	}
	if data.Timestamp == 0 {
		return nil, errEmptyTimestamp
	}

	data.Message = strings.TrimSpace(data.Message)
	data.Severity = strings.TrimSpace(data.Severity)

	var spanID, traceID string

	if data.Context != nil {
		txn := FromContext(data.Context)
		traceMetadata := txn.GetTraceMetadata()
		spanID = traceMetadata.SpanID
		traceID = traceMetadata.TraceID
	}

	event := logEvent{
		message:   data.Message,
		severity:  data.Severity,
		spanID:    spanID,
		traceID:   traceID,
		timestamp: data.Timestamp,
	}

	return &event, nil
}

func (e *logEvent) MergeIntoHarvest(h *harvest) {
	// Inherit priority from traces or spans if possible
	if e.traceID != "" {
		priority, known := h.knownPriorities.get(e.traceID)
		if known {
			e.priority = priority
		}
	}

	h.LogEvents.Add(e)
}
