// Copyright 2025 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import "bytes"

type spanLink struct {
	spanLinkType  string
	timestamp     int64 // epoch milliseconds
	id            string
	traceId       string
	linkedSpanId  string
	linkedTraceId string
}

// WriteJSON prepares JSON in the format expected by the collector. A span link
// is serialized as its own event within the span_event_data payload, using the
// same three-element [intrinsics, userAttrs, agentAttrs] shape as a span event;
// only the intrinsic attributes differ.
func (e *spanLink) WriteJSON(buf *bytes.Buffer) {
	w := jsonFieldsWriter{buf: buf}
	buf.WriteByte('[')
	buf.WriteByte('{')
	w.stringField("type", e.spanLinkType)
	w.stringField("trace.id", e.traceId)
	w.stringField("id", e.id)
	w.stringField("linkedTraceId", e.linkedTraceId)
	w.stringField("linkedSpanId", e.linkedSpanId)
	w.intField("timestamp", e.timestamp)
	buf.WriteByte('}')
	buf.WriteByte(',')
	buf.WriteByte('{')
	// user attributes (link attributes are not yet captured)
	buf.WriteByte('}')
	buf.WriteByte(',')
	buf.WriteByte('{')
	// agent attributes
	buf.WriteByte('}')
	buf.WriteByte(']')
}

// MarshalJSON is used for testing.
func (e *spanLink) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 256))

	e.WriteJSON(buf)

	return buf.Bytes(), nil
}
