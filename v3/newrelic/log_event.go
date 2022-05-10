// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	LogSeverityFieldName  = "level"
	LogMessageFieldName   = "message"
	LogTimestampFieldName = "timestamp"
	LogSpanIDFieldName    = "span.id"
	LogTraceIDFieldName   = "trace.id"

	maxLogBytes = 32768
)

type logEvent struct {
	priority priority
	traceID  string
	severity string
	log      string
}

// writeJSON prepares JSON in the format expected by the collector.
func (e *logEvent) WriteJSON(buf *bytes.Buffer) {
	buf.WriteString(e.log)
}

// MarshalJSON is used for testing.
func (e *logEvent) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 256))

	e.WriteJSON(buf)

	return buf.Bytes(), nil
}

type logJson struct {
	Timestamp float64 `json:"timestamp"`
	Severity  string  `json:"level"`
	Message   string  `json:"message"`
	SpanID    string  `json:"span.id"`
	TraceID   string  `json:"trace.id"`
}

var (
	// regex allows a single word, or number
	severityUnknown = "UNKNOWN"
	errEmptyLog     = errors.New("log event can not be empty")
	errLogTooLarge  = fmt.Errorf("log can not exceed %d bytes", maxLogBytes)
)

func CreateLogEvent(log []byte) (logEvent, error) {
	if len(log) > maxLogBytes {
		return logEvent{}, errLogTooLarge
	}
	if len(log) == 0 {
		return logEvent{}, errEmptyLog
	}

	l := &logJson{}
	err := json.Unmarshal(log, l)
	if err != nil {
		return logEvent{}, err
	}

	logEvent := logEvent{
		log:      string(log),
		severity: l.Severity,
		traceID:  l.TraceID,
	}

	return logEvent, nil
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
