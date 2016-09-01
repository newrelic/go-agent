package internal

import (
	"bytes"
	"math/rand"
	"time"

	"github.com/newrelic/go-agent/internal/jsonx"
)

// DatastoreExternalTotals contains overview of external and datastore calls
// made during a transaction.
type DatastoreExternalTotals struct {
	externalCallCount  uint64
	externalDuration   time.Duration
	datastoreCallCount uint64
	datastoreDuration  time.Duration
}

// TxnEvent represents a transaction.
// https://source.datanerd.us/agents/agent-specs/blob/master/Transaction-Events-PORTED.md
// https://newrelic.atlassian.net/wiki/display/eng/Agent+Support+for+Synthetics%3A+Forced+Transaction+Traces+and+Analytic+Events
type TxnEvent struct {
	Name      string
	Timestamp time.Time
	Duration  time.Duration
	Queuing   time.Duration
	Zone      ApdexZone
	Attrs     *Attributes
	DatastoreExternalTotals
}

// WriteJSON prepares JSON in the format expected by the collector.
func (e *TxnEvent) WriteJSON(buf *bytes.Buffer) {
	buf.WriteString(`[{"type":"Transaction","name":`)
	jsonx.AppendString(buf, e.Name)
	buf.WriteString(`,"timestamp":`)
	jsonx.AppendFloat(buf, timeToFloatSeconds(e.Timestamp))
	buf.WriteString(`,"duration":`)
	jsonx.AppendFloat(buf, e.Duration.Seconds())
	if ApdexNone != e.Zone {
		buf.WriteString(`,"nr.apdexPerfZone":`)
		jsonx.AppendString(buf, e.Zone.label())
	}
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
	userAttributesJSON(e.Attrs, buf, destTxnEvent)
	buf.WriteByte(',')
	agentAttributesJSON(e.Attrs, buf, destTxnEvent)
	buf.WriteByte(']')
}

// MarshalJSON is used for testing.
func (e *TxnEvent) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 256))

	e.WriteJSON(buf)

	return buf.Bytes(), nil
}

type txnEvents struct {
	events *analyticsEvents
}

func newTxnEvents(max int) *txnEvents {
	return &txnEvents{
		events: newAnalyticsEvents(max),
	}
}

func (events *txnEvents) AddTxnEvent(e *TxnEvent) {
	stamp := eventStamp(rand.Float32())
	events.events.addEvent(analyticsEvent{stamp, e})
}

func (events *txnEvents) MergeIntoHarvest(h *Harvest) {
	h.TxnEvents.events.mergeFailed(events.events)
}

func (events *txnEvents) Data(agentRunID string, harvestStart time.Time) ([]byte, error) {
	return events.events.CollectorJSON(agentRunID)
}

func (events *txnEvents) numSeen() float64  { return events.events.NumSeen() }
func (events *txnEvents) numSaved() float64 { return events.events.NumSaved() }
