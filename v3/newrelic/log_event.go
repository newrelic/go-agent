// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
)

var (
	// regex allows a single word, or number
	severityRegexRaw = `^[a-zA-Z]+$|^[0-9]+$`
	severityRegex    = regexp.MustCompile(severityRegexRaw)
	severityUnknown  = "UNKNOWN"

	errNilLogEvent      = errors.New("log event can not be nil")
	errEmptySeverity    = errors.New("severity can not be an empty string")
	errSeverityTooLarge = fmt.Errorf("severity exceeds length limit of %d", attributeKeyLengthLimit)
	errSeverityRegex    = fmt.Errorf("severity must match %s", severityRegexRaw)
	errMessageSizeZero  = errors.New("message must be a non empty string")
)

type logEvent struct {
	priority  priority
	severity  string
	message   string
	spanID    string
	traceID   string
	timestamp int64
}

// ValidateAndRender validates inputs, and creates a rendered log event with
// a jsonWriter buffer populated by rendered json
func (event *logEvent) Validate() error {
	if event == nil {
		return errNilLogEvent
	}

	// Default severity to "UNKNOWN" if no severity is passed.
	if len(event.severity) == 0 {
		event.severity = severityUnknown
	}

	if ok, err := validateSeverity(event.severity); !ok {
		return fmt.Errorf("invalid severity: %s", err)
	}

	if len(event.message) == 0 {
		return errMessageSizeZero
	}

	return nil
}

// writeJSON prepares JSON in the format expected by the collector.
func (e *logEvent) WriteJSON(buf *bytes.Buffer) {
	w := jsonFieldsWriter{buf: buf}
	buf.WriteByte('{')
	w.stringField("severity", e.severity)
	w.stringField("message", e.message)

	if len(e.spanID) > 0 {
		w.stringField("span.id", e.spanID)
	}
	if len(e.traceID) > 0 {
		w.stringField("trace.id", e.traceID)
	}

	w.needsComma = false
	buf.WriteByte(',')
	w.intField("timestamp", e.timestamp)
	buf.WriteByte('}')
}

// MarshalJSON is used for testing.
func (e *logEvent) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 256))

	e.WriteJSON(buf)

	return buf.Bytes(), nil
}

// must be a single word or number. If unknown, should be "UNKNOWN"
func validateSeverity(severity string) (bool, error) {
	size := len(severity)
	if size == 0 {
		return false, errEmptySeverity
	}
	if size > attributeKeyLengthLimit {
		return false, errSeverityTooLarge
	}

	if !severityRegex.MatchString(severity) {
		return false, errSeverityRegex
	}
	return true, nil
}

func (e *logEvent) MergeIntoHarvest(h *harvest) {
	h.LogEvents.Add(e)
}
