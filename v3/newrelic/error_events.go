// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"time"
)

// MarshalJSON is used for testing.
func (e *errorEvent) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 256))

	e.WriteJSON(buf)

	return buf.Bytes(), nil
}

// WriteJSON prepares JSON in the format expected by the collector.
// https://source.datanerd.us/agents/agent-specs/blob/master/Error-Events.md
func (e *errorEvent) WriteJSON(buf *bytes.Buffer) {
	w := jsonFieldsWriter{buf: buf}
	buf.WriteByte('[')
	buf.WriteByte('{')
	w.stringField("type", "TransactionError")
	w.stringField("error.class", e.Klass)
	w.stringField("error.message", e.Msg)
	w.intField("timestamp", timeToIntMillis(e.When))
	w.stringField("transactionName", e.FinalName)
	if e.SpanID != "" {
		w.stringField("spanId", e.SpanID)
	}

	sharedTransactionIntrinsics(&e.txnEvent, &w)
	sharedBetterCATIntrinsics(&e.txnEvent, &w)

	buf.WriteByte('}')
	buf.WriteByte(',')
	userAttributesJSON(e.Attrs, buf, destError, e.errorData.ExtraAttributes)
	buf.WriteByte(',')
	agentAttributesJSON(e.Attrs, buf, destError)
	buf.WriteByte(']')
}

type errorEvents struct {
	*analyticsEvents
}

func newErrorEvents(max int) *errorEvents {
	return &errorEvents{
		analyticsEvents: newAnalyticsEvents(max),
	}
}

func (events *errorEvents) Add(e *errorEvent, p priority) {
	events.addEvent(analyticsEvent{p, e})
}

func (events *errorEvents) MergeIntoHarvest(h *harvest) {
	h.ErrorEvents.mergeFailed(events.analyticsEvents)
}

func (events *errorEvents) DataBuffer() *bytes.Buffer {
	if 0 == len(events.events) {
		return nil
	}

	estimate := 256 * len(events.events)
	return bytes.NewBuffer(make([]byte, 0, estimate))
}

func (events *errorEvents) WriteData(buf *bytes.Buffer, agentRunID string, harvestStart time.Time) error {
	return events.CollectorJSON(buf, agentRunID)
}

func (events *errorEvents) EndpointMethod() string {
	return cmdErrorEvents
}
