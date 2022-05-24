// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"errors"
	"fmt"
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

// For customer use
type LogData struct {
	Timestamp int64
	Severity  string
	Message   string
	SpanID    string
	TraceID   string
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
	errEmptySeverity      = errors.New("severity can not be empty")
	errNilLogData         = errors.New("log data can not be nil")
	errLogMessageTooLarge = fmt.Errorf("log message can not exceed %d bytes", MaxLogLength)
)

func (data *LogData) ToLogEvent() (*logEvent, error) {
	if data == nil {
		return nil, errNilLogData
	}
	if data.Severity == "" {
		return nil, errEmptySeverity
	}
	if len(data.Message) > MaxLogLength {
		return nil, errLogMessageTooLarge
	}
	if data.Timestamp == 0 {
		return nil, errEmptyTimestamp
	}

	event := logEvent{
		message:   data.Message,
		severity:  data.Severity,
		spanID:    data.SpanID,
		traceID:   data.TraceID,
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
