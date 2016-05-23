package internal

import (
	"bytes"
	"math/rand"
	"time"

	"github.com/newrelic/go-sdk/internal/jsonx"
)

// https://source.datanerd.us/agents/agent-specs/blob/master/Transaction-Events-PORTED.md
// https://newrelic.atlassian.net/wiki/display/eng/Agent+Support+for+Synthetics%3A+Forced+Transaction+Traces+and+Analytic+Events
type txnEvent struct {
	Name      string
	Timestamp time.Time
	Duration  time.Duration
	zone      apdexZone
}

func (e *txnEvent) WriteJSON(buf *bytes.Buffer) {
	buf.WriteString(`[{"type":"Transaction","name":`)
	jsonx.AppendString(buf, e.Name)
	buf.WriteString(`,"timestamp":`)
	jsonx.AppendFloat(buf, timeToFloatSeconds(e.Timestamp))
	buf.WriteString(`,"duration":`)
	jsonx.AppendFloat(buf, e.Duration.Seconds())
	if apdexNone != e.zone {
		buf.WriteString(`,"nr.apdexPerfZone":`)
		jsonx.AppendString(buf, e.zone.label())
	}
	buf.WriteString(`},{},{}]`)
}

func (e *txnEvent) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 256))

	e.WriteJSON(buf)

	return buf.Bytes(), nil
}

func createTxnEvent(zone apdexZone, name string, d time.Duration, start time.Time) *txnEvent {
	event := txnEvent{
		Name:      name,
		Timestamp: start,
		Duration:  d,
		zone:      zone,
	}

	return &event
}

type txnEvents struct {
	events *analyticsEvents
}

func newTxnEvents(max int) *txnEvents {
	return &txnEvents{
		events: newAnalyticsEvents(max),
	}
}

func (events *txnEvents) AddTxnEvent(e *txnEvent) {
	stamp := eventStamp(rand.Float32())
	events.events.AddEvent(analyticsEvent{stamp, e})
}

func (events *txnEvents) MergeIntoHarvest(h *Harvest) {
	h.txnEvents.events.MergeFailed(events.events)
}

func (events *txnEvents) Data(agentRunID string, harvestStart time.Time) ([]byte, error) {
	return events.events.CollectorJSON(agentRunID)
}

func (events *txnEvents) numSeen() float64  { return events.events.NumSeen() }
func (events *txnEvents) numSaved() float64 { return events.events.NumSaved() }
