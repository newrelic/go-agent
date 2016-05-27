package internal

import (
	"bytes"
	"math/rand"
	"time"

	"github.com/newrelic/go-sdk/internal/jsonx"
)

type errorEvent struct {
	klass    string
	msg      string
	when     time.Time
	txnName  string
	duration time.Duration
	attrs    *attributes
}

func (e *errorEvent) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 256))

	e.WriteJSON(buf)

	return buf.Bytes(), nil
}

// https://source.datanerd.us/agents/agent-specs/blob/master/Error-Events.md
func (e *errorEvent) WriteJSON(buf *bytes.Buffer) {
	buf.WriteString(`[{"type":"TransactionError","error.class":`)
	jsonx.AppendString(buf, e.klass)

	buf.WriteString(`,"error.message":`)
	jsonx.AppendString(buf, e.msg)

	buf.WriteString(`,"timestamp":`)
	jsonx.AppendFloat(buf, timeToFloatSeconds(e.when))

	buf.WriteString(`,"transactionName":`)
	jsonx.AppendString(buf, e.txnName)

	buf.WriteString(`,"duration":`)
	jsonx.AppendFloat(buf, e.duration.Seconds())

	buf.WriteByte('}')
	buf.WriteByte(',')
	userAttributesJSON(e.attrs, buf, destError)
	buf.WriteByte(',')
	agentAttributesJSON(e.attrs, buf, destError)
	buf.WriteByte(']')
}

func createErrorEvent(e *txnError, txnName string, duration time.Duration, attrs *attributes) *errorEvent {
	return &errorEvent{
		klass:    e.klass,
		msg:      e.msg,
		when:     e.when,
		txnName:  txnName,
		duration: duration,
		attrs:    attrs,
	}
}

type errorEvents struct {
	events *analyticsEvents
}

func newErrorEvents(max int) *errorEvents {
	return &errorEvents{
		events: newAnalyticsEvents(max),
	}
}

func (events *errorEvents) Add(e *errorEvent) {
	stamp := eventStamp(rand.Float32())
	events.events.AddEvent(analyticsEvent{stamp, e})
}

func (events *errorEvents) mergeIntoHarvest(h *harvest) {
	h.errorEvents.events.MergeFailed(events.events)
}

func (events *errorEvents) Data(agentRunID string, harvestStart time.Time) ([]byte, error) {
	return events.events.CollectorJSON(agentRunID)
}

func (events *errorEvents) numSeen() float64  { return events.events.NumSeen() }
func (events *errorEvents) numSaved() float64 { return events.events.NumSaved() }
