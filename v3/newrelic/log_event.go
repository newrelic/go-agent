// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Exported Constants for log decorators
const (
	// LogSeverityFieldName is the name of the log level field in New Relic logging JSON
	LogSeverityFieldName = "level"

	// LogMessageFieldName is the name of the log message field in New Relic logging JSON
	LogMessageFieldName = "message"

	// LogTimestampFieldName is the name of the timestamp field in New Relic logging JSON
	LogTimestampFieldName = "timestamp"

	// LogSpanIDFieldName is the name of the span ID field in the New Relic logging JSON
	LogSpanIDFieldName = "span.id"

	// LogTraceIDFieldName is the name of the trace ID field in the New Relic logging JSON
	LogTraceIDFieldName = "trace.id"

	// LogSeverityUnknown is the value the log severity should be set to if no log severity is known
	LogSeverityUnknown = "UNKNOWN"

	// MaxLogLength is the maximum number of bytes the log message is allowed to be
	MaxLogLength = 32768
)

// internal variable names and constants
const (
	// number of bytes expected to be needed for the average log message
	averageLogSizeEstimate = 400
)

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
	Timestamp int64  // Optional: Unix Millisecond Timestamp; A timestamp will be generated if unset
	Severity  string // Optional: Severity of log being consumed
	Message   string // Optional: Message of log being consumed; Maximum size: 32768 Bytes.
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
	buf := bytes.NewBuffer(make([]byte, 0, averageLogSizeEstimate))
	e.WriteJSON(buf)
	return buf.Bytes(), nil
}

var (
	errNilLogData         = errors.New("log data can not be nil")
	errLogMessageTooLarge = fmt.Errorf("log message can not exceed %d bytes", MaxLogLength)
)

func (data *LogData) toLogEvent() (logEvent, error) {
	if data == nil {
		return logEvent{}, errNilLogData
	}
	if data.Severity == "" {
		data.Severity = LogSeverityUnknown
	}
	if len(data.Message) > MaxLogLength {
		return logEvent{}, errLogMessageTooLarge
	}
	if data.Timestamp == 0 {
		data.Timestamp = int64(timeToUnixMilliseconds(time.Now()))
	}

	data.Message = strings.TrimSpace(data.Message)
	data.Severity = strings.TrimSpace(data.Severity)

	event := logEvent{
		priority:  newPriority(),
		message:   data.Message,
		severity:  data.Severity,
		timestamp: data.Timestamp,
	}

	return event, nil
}

func (e *logEvent) MergeIntoHarvest(h *harvest) {
	h.LogEvents.Add(e)
}
