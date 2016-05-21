package internal

import "time"

type validator interface {
	Error(...interface{})
}

func validateStringField(v validator, fieldName, v1, v2 string) {
	if v1 != v2 {
		v.Error(fieldName, v1, v2)
	}
}

type WantMetric struct {
	Name   string
	Scope  string
	Forced bool
	Data   []float64
}

type WantCustomEvent struct {
	Type   string
	Params map[string]interface{}
}

type WantError struct {
	TxnName string
	Msg     string
	Klass   string
	Caller  string
	URL     string
}

type WantErrorEvent struct {
	TxnName string
	Msg     string
	Klass   string
}

type WantTxnEvent struct {
	Name string
	Zone string
}

func (h *Harvest) ExpectCustomEvents(t validator, want []WantCustomEvent) {
	expectCustomEvents(t, h.customEvents, want)
}

func (h *Harvest) ExpectErrors(t validator, want []WantError) {
	expectErrors(t, h.errorTraces, want)
}

func (h *Harvest) ExpectErrorEvents(t validator, want []WantErrorEvent) {
	expectErrorEvents(t, h.errorEvents, want)
}

func (h *Harvest) ExpectTxnEvents(t validator, want []WantTxnEvent) {
	expectTxnEvents(t, h.txnEvents, want)
}

func (h *Harvest) ExpectMetrics(t validator, want []WantMetric) {
	expectMetrics(t, h.metrics, want)
}

func expectMetricField(t validator, id metricID, v1, v2 float64, fieldName string) {
	if v1 != v2 {
		t.Error("metric fields do not match", id, v1, v2, fieldName)
	}
}

func expectMetrics(t validator, mt *metricTable, expect []WantMetric) {
	if len(mt.metrics) != len(expect) {
		t.Error("metric counts do not match expectations", len(mt.metrics), len(expect))
	}
	expectedIds := make(map[metricID]struct{})
	for _, e := range expect {
		id := metricID{Name: e.Name, Scope: e.Scope}
		expectedIds[id] = struct{}{}
		m := mt.metrics[id]
		if nil == m {
			t.Error("unable to find metric", id)
			continue
		}

		if e.Forced != (forced == m.forced) {
			t.Error("metric forced incorrect", e.Forced, m.forced, id)
		}

		if nil != e.Data {
			expectMetricField(t, id, e.Data[0], m.data.countSatisfied, "countSatisfied")
			expectMetricField(t, id, e.Data[1], m.data.totalTolerated, "totalTolerated")
			expectMetricField(t, id, e.Data[2], m.data.exclusiveFailed, "exclusiveFailed")
			expectMetricField(t, id, e.Data[3], m.data.min, "min")
			expectMetricField(t, id, e.Data[4], m.data.max, "max")
			expectMetricField(t, id, e.Data[5], m.data.sumSquares, "sumSquares")
		}
	}
	for id := range mt.metrics {
		if _, ok := expectedIds[id]; !ok {
			t.Error("expected metrics does not contain", id.Name, id.Scope)
		}
	}
}

func expectCustomEvent(v validator, event *customEvent, expect WantCustomEvent) {
	if event.eventType != expect.Type {
		v.Error("type mismatch", event.eventType, expect.Type)
	}
	now := time.Now()
	diff := absTimeDiff(now, event.timestamp)
	if diff > time.Hour {
		v.Error("large timestamp difference", event.eventType, now, event.timestamp)
	}
	// TODO: This params comparison can be made smarter: Alert differences
	// based on sub/super set behavior.
	if len(event.truncatedParams) != len(expect.Params) {
		v.Error("params length difference", event.truncatedParams, expect.Params)
		return
	}
	for key, val := range expect.Params {
		found, ok := event.truncatedParams[key]
		if !ok {
			v.Error("missing key", key)
		} else if val != found {
			v.Error("value difference", val, found)
		}
	}
}

func expectCustomEvents(v validator, cs *customEvents, expect []WantCustomEvent) {
	if len(*cs.events.events) != len(expect) {
		v.Error("number of custom events does not match", len(*cs.events.events),
			len(expect))
		return
	}
	for i, e := range expect {
		event, ok := (*cs.events.events)[i].JSONWriter.(*customEvent)
		if !ok {
			v.Error("wrong custom event")
		} else {
			expectCustomEvent(v, event, e)
		}
	}
}

func expectErrorEvent(v validator, err *errorEvent, expect WantErrorEvent) {
	validateStringField(v, "txnName", expect.TxnName, err.txnName)
	validateStringField(v, "klass", expect.Klass, err.klass)
	validateStringField(v, "msg", expect.Msg, err.msg)
}

func expectErrorEvents(v validator, events *errorEvents, expect []WantErrorEvent) {
	if len(*events.events.events) != len(expect) {
		v.Error("number of custom events does not match",
			len(*events.events.events), len(expect))
		return
	}
	for i, e := range expect {
		event, ok := (*events.events.events)[i].JSONWriter.(*errorEvent)
		if !ok {
			v.Error("wrong error event")
		} else {
			expectErrorEvent(v, event, e)
		}
	}
}

func expectTxnEvent(v validator, e *TxnEvent, expect WantTxnEvent) {
	validateStringField(v, "apdex zone", expect.Zone, e.zone.label())
	validateStringField(v, "name", expect.Name, e.Name)
	if 0 == e.Duration {
		v.Error("zero duration", e.Duration)
	}
}

func expectTxnEvents(v validator, events *txnEvents, expect []WantTxnEvent) {
	if len(*events.events.events) != len(expect) {
		v.Error("number of txn events does not match",
			len(*events.events.events), len(expect))
		return
	}
	for i, e := range expect {
		event, ok := (*events.events.events)[i].JSONWriter.(*TxnEvent)
		if !ok {
			v.Error("wrong txn event")
		} else {
			expectTxnEvent(v, event, e)
		}
	}
}

func expectError(v validator, err *harvestError, expect WantError) {
	caller := topCallerNameBase(err.txnError.stack)
	validateStringField(v, "caller", expect.Caller, caller)
	validateStringField(v, "txnName", expect.TxnName, err.txnName)
	validateStringField(v, "klass", expect.Klass, err.txnError.klass)
	validateStringField(v, "msg", expect.Msg, err.txnError.msg)
	validateStringField(v, "URL", expect.URL, err.requestURI)
}

func expectErrors(v validator, errors *harvestErrors, expect []WantError) {
	if len(errors.errors) != len(expect) {
		v.Error("number of errors mismatch", len(errors.errors), len(expect))
		return
	}
	for i, e := range expect {
		expectError(v, errors.errors[i], e)
	}
}
