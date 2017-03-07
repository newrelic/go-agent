package internal

import (
	"bytes"
	"time"
)

// DatastoreExternalTotals contains overview of external and datastore calls
// made during a transaction.
type DatastoreExternalTotals struct {
	externalCallCount  uint64
	externalDuration   time.Duration
	datastoreCallCount uint64
	datastoreDuration  time.Duration
}

// WriteJSON prepares JSON in the format expected by the collector.
func (e *TxnEvent) WriteJSON(buf *bytes.Buffer) {
	w := jsonFieldsWriter{buf: buf}
	buf.WriteByte('[')
	buf.WriteByte('{')
	w.stringField("type", "Transaction")
	w.stringField("name", e.FinalName)
	w.floatField("timestamp", timeToFloatSeconds(e.Start))
	if ApdexNone != e.Zone {
		w.stringField("nr.apdexPerfZone", e.Zone.label())
	}
	w.stringField("nr.guid", e.ID)
	sharedIntrinsics(e, &w, false)
	buf.WriteByte('}')
	buf.WriteByte(',')
	userAttributesJSON(e.Attrs, buf, destTxnEvent)
	buf.WriteByte(',')
	agentAttributesJSON(e.Attrs, buf, destTxnEvent)
	buf.WriteByte(']')
}

func sharedIntrinsics(e *TxnEvent, w *jsonFieldsWriter, isTrace bool) {
	e.Proxies.createIntrinsics(w)
	if p := e.Inbound; nil != p {
		w.stringField("caller.type", p.Type)
		w.stringField("caller.app", p.App)
		w.stringField("caller.account", p.Account)
		w.stringField("caller.transportType", p.TransportType)
		if "" != p.Host {
			w.stringField("caller.host", p.Host)
		}
		w.floatField("caller.transportDuration", p.TransportDuration.Seconds())
		if p.Order >= 0 {
			w.intField("nr.order", int64(p.Order))
		}
		if "" != p.ID {
			if isTrace {
				w.stringField("referring_transaction_guid", p.ID)
			} else {
				w.stringField("nr.referringTransactionGuid", p.ID)
			}
		}
		if nil != p.Synthetics {
			if isTrace {
				w.stringField("synthetics_resource_id", p.Synthetics.Resource)
				w.stringField("synthetics_job_id", p.Synthetics.Job)
				w.stringField("synthetics_monitor_id", p.Synthetics.Monitor)
			} else {
				w.stringField("nr.syntheticsResourceId", p.Synthetics.Resource)
				w.stringField("nr.syntheticsJobId", p.Synthetics.Job)
				w.stringField("nr.syntheticsMonitorId", p.Synthetics.Monitor)
			}
		}
	}
	w.intField("nr.priority", int64(e.Priority.Value()))
	w.intField("nr.depth", int64(e.Depth()))
	w.stringField("nr.tripId", e.TripID())

	if !isTrace {
		w.floatField("duration", e.Duration.Seconds())
		if e.externalCallCount > 0 {
			w.intField("externalCallCount", int64(e.externalCallCount))
			w.floatField("externalDuration", e.externalDuration.Seconds())
		}
		if e.datastoreCallCount > 0 {
			// Note that "database" is used for the keys here instead of
			// "datastore" for historical reasons.
			w.intField("databaseCallCount", int64(e.datastoreCallCount))
			w.floatField("databaseDuration", e.datastoreDuration.Seconds())
		}
	}
}

func traceIntrinsics(e *TxnEvent, buf *bytes.Buffer) {
	w := jsonFieldsWriter{buf: buf}
	buf.WriteByte('{')
	sharedIntrinsics(e, &w, true)
	buf.WriteByte('}')
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

func (events *txnEvents) AddTxnEvent(e *TxnEvent, stamp uint32) {
	events.events.addEvent(analyticsEvent{eventStamp(stamp), e})
}

func (events *txnEvents) MergeIntoHarvest(h *Harvest) {
	h.TxnEvents.events.mergeFailed(events.events)
}

func (events *txnEvents) Data(agentRunID string, harvestStart time.Time) ([]byte, error) {
	return events.events.CollectorJSON(agentRunID)
}

func (events *txnEvents) numSeen() float64  { return events.events.NumSeen() }
func (events *txnEvents) numSaved() float64 { return events.events.NumSaved() }
