package internal

import (
	"bytes"
	"math/rand"
	"time"

	"github.com/newrelic/go-agent/internal/jsonx"
)

// ErrorEvent is an error event.
type ErrorEvent struct {
	Klass    string
	Msg      string
	When     time.Time
	TxnName  string
	Duration time.Duration
	Queuing  time.Duration
	Attrs    *Attributes
	DatastoreExternalTotals
}

// MarshalJSON is used for testing.
func (e *ErrorEvent) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 256))

	e.WriteJSON(buf)

	return buf.Bytes(), nil
}

// WriteJSON prepares JSON in the format expected by the collector.
// https://source.datanerd.us/agents/agent-specs/blob/master/Error-Events.md
func (e *ErrorEvent) WriteJSON(buf *bytes.Buffer) {
	buf.WriteString(`[{"type":"TransactionError","error.class":`)
	jsonx.AppendString(buf, e.Klass)

	buf.WriteString(`,"error.message":`)
	jsonx.AppendString(buf, e.Msg)

	buf.WriteString(`,"timestamp":`)
	jsonx.AppendFloat(buf, timeToFloatSeconds(e.When))

	buf.WriteString(`,"transactionName":`)
	jsonx.AppendString(buf, e.TxnName)

	buf.WriteString(`,"duration":`)
	jsonx.AppendFloat(buf, e.Duration.Seconds())

	if e.Queuing > 0 {
		buf.WriteString(`,"queueDuration":`)
		jsonx.AppendFloat(buf, e.Queuing.Seconds())
	}

	if e.externalCallCount > 0 {
		buf.WriteString(`,"externalCallCount":`)
		jsonx.AppendInt(buf, int64(e.externalCallCount))
		buf.WriteString(`,"externalDuration":`)
		jsonx.AppendFloat(buf, e.externalDuration.Seconds())
	}

	if e.datastoreCallCount > 0 {
		// Note that "database" is used for the keys here instead of
		// "datastore" for historical reasons.
		buf.WriteString(`,"databaseCallCount":`)
		jsonx.AppendInt(buf, int64(e.datastoreCallCount))
		buf.WriteString(`,"databaseDuration":`)
		jsonx.AppendFloat(buf, e.datastoreDuration.Seconds())
	}

	buf.WriteByte('}')
	buf.WriteByte(',')
	userAttributesJSON(e.Attrs, buf, destError)
	buf.WriteByte(',')
	agentAttributesJSON(e.Attrs, buf, destError)
	buf.WriteByte(']')
}

type errorEvents struct {
	events *analyticsEvents
}

func newErrorEvents(max int) *errorEvents {
	return &errorEvents{
		events: newAnalyticsEvents(max),
	}
}

func (events *errorEvents) Add(e *ErrorEvent) {
	stamp := eventStamp(rand.Float32())
	events.events.addEvent(analyticsEvent{stamp, e})
}

func (events *errorEvents) MergeIntoHarvest(h *Harvest) {
	h.ErrorEvents.events.mergeFailed(events.events)
}

func (events *errorEvents) Data(agentRunID string, harvestStart time.Time) ([]byte, error) {
	return events.events.CollectorJSON(agentRunID)
}

func (events *errorEvents) numSeen() float64  { return events.events.NumSeen() }
func (events *errorEvents) numSaved() float64 { return events.events.NumSaved() }
