package internal

import (
	"encoding/json"
	"fmt"
	"time"
)

func validateStringField(v Validator, fieldName, v1, v2 string) {
	if v1 != v2 {
		v.Error(fieldName, v1, v2)
	}
}

type addValidatorField struct {
	field    interface{}
	original Validator
}

func (a addValidatorField) Error(fields ...interface{}) {
	fields = append([]interface{}{a.field}, fields...)
	a.original.Error(fields...)
}

// ExtendValidator is used to add more context to a validator.
func ExtendValidator(v Validator, field interface{}) Validator {
	return addValidatorField{
		field:    field,
		original: v,
	}
}

// ExpectTxnMetrics tests that the app contains metrics for a transaction.
func ExpectTxnMetrics(t Validator, mt *metricTable, want WantTxn) {
	var metrics []WantMetric
	var scope string
	var allWebOther string
	if want.IsWeb {
		scope = "WebTransaction/Go/" + want.Name
		allWebOther = "allWeb"
		metrics = []WantMetric{
			{Name: "WebTransaction/Go/" + want.Name, Scope: "", Forced: true, Data: nil},
			{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
			{Name: "WebTransactionTotalTime/Go/" + want.Name, Scope: "", Forced: false, Data: nil},
			{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
			{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
			{Name: "Apdex", Scope: "", Forced: true, Data: nil},
			{Name: "Apdex/Go/" + want.Name, Scope: "", Forced: false, Data: nil},
		}
	} else {
		scope = "OtherTransaction/Go/" + want.Name
		allWebOther = "allOther"
		metrics = []WantMetric{
			{Name: "OtherTransaction/Go/" + want.Name, Scope: "", Forced: true, Data: nil},
			{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
			{Name: "OtherTransactionTotalTime/Go/" + want.Name, Scope: "", Forced: false, Data: nil},
			{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		}
	}
	if want.NumErrors > 0 {
		data := []float64{float64(want.NumErrors), 0, 0, 0, 0, 0}
		metrics = append(metrics, []WantMetric{
			{Name: "Errors/all", Scope: "", Forced: true, Data: data},
			{Name: "Errors/" + allWebOther, Scope: "", Forced: true, Data: data},
			{Name: "Errors/" + scope, Scope: "", Forced: true, Data: data},
		}...)
	}
	ExpectMetrics(t, mt, metrics)
}

func expectMetricField(t Validator, id metricID, v1, v2 float64, fieldName string) {
	if v1 != v2 {
		t.Error("metric fields do not match", id, v1, v2, fieldName)
	}
}

// ExpectMetricsPresent allows testing of metrics without requiring an exact match
func ExpectMetricsPresent(t Validator, mt *metricTable, expect []WantMetric) {
	expectMetrics(t, mt, expect, false)
}

// ExpectMetrics allows testing of metrics.  It passes if mt exactly matches expect.
func ExpectMetrics(t Validator, mt *metricTable, expect []WantMetric) {
	expectMetrics(t, mt, expect, true)
}

func expectMetrics(t Validator, mt *metricTable, expect []WantMetric, exactMatch bool) {
	if exactMatch {
		if len(mt.metrics) != len(expect) {
			t.Error("metric counts do not match expectations", len(mt.metrics), len(expect))
		}
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

		if b, ok := e.Forced.(bool); ok {
			if b != (forced == m.forced) {
				t.Error("metric forced incorrect", b, m.forced, id)
			}
		}

		if nil != e.Data {
			expectMetricField(t, id, e.Data[0], m.data.countSatisfied, "countSatisfied")

			if len(e.Data) > 1 {
				expectMetricField(t, id, e.Data[1], m.data.totalTolerated, "totalTolerated")
				expectMetricField(t, id, e.Data[2], m.data.exclusiveFailed, "exclusiveFailed")
				expectMetricField(t, id, e.Data[3], m.data.min, "min")
				expectMetricField(t, id, e.Data[4], m.data.max, "max")
				expectMetricField(t, id, e.Data[5], m.data.sumSquares, "sumSquares")
			}
		}
	}
	if exactMatch {
		for id := range mt.metrics {
			if _, ok := expectedIds[id]; !ok {
				t.Error("expected metrics does not contain", id.Name, id.Scope)
			}
		}
	}
}

func expectAttributes(v Validator, exists map[string]interface{}, expect map[string]interface{}) {
	// TODO: This params comparison can be made smarter: Alert differences
	// based on sub/super set behavior.
	if len(exists) != len(expect) {
		v.Error("attributes length difference", len(exists), len(expect))
	}
	for key, val := range expect {
		found, ok := exists[key]
		if !ok {
			v.Error("expected attribute not found: ", key)
			continue
		}
		if val == MatchAnything {
			continue
		}
		v1 := fmt.Sprint(found)
		v2 := fmt.Sprint(val)
		if v1 != v2 {
			v.Error("value difference", fmt.Sprintf("key=%s", key), v1, v2)
		}
	}
	for key, val := range exists {
		_, ok := expect[key]
		if !ok {
			v.Error("unexpected attribute present: ", key, val)
			continue
		}
	}
}

// ExpectCustomEvents allows testing of custom events.  It passes if cs exactly matches expect.
func ExpectCustomEvents(v Validator, cs *customEvents, expect []WantEvent) {
	expectEvents(v, cs.analyticsEvents, expect, nil)
}

func expectEvent(v Validator, e json.Marshaler, expect WantEvent) {
	js, err := e.MarshalJSON()
	if nil != err {
		v.Error("unable to marshal event", err)
		return
	}
	var event []map[string]interface{}
	err = json.Unmarshal(js, &event)
	if nil != err {
		v.Error("unable to parse event json", err)
		return
	}
	intrinsics := event[0]
	userAttributes := event[1]
	agentAttributes := event[2]

	if nil != expect.Intrinsics {
		expectAttributes(v, intrinsics, expect.Intrinsics)
	}
	if nil != expect.UserAttributes {
		expectAttributes(v, userAttributes, expect.UserAttributes)
	}
	if nil != expect.AgentAttributes {
		expectAttributes(v, agentAttributes, expect.AgentAttributes)
	}
}

func expectEvents(v Validator, events *analyticsEvents, expect []WantEvent, extraAttributes map[string]interface{}) {
	if len(events.events) != len(expect) {
		v.Error("number of events does not match", len(events.events), len(expect))
		return
	}
	for i, e := range expect {
		event, ok := events.events[i].jsonWriter.(json.Marshaler)
		if !ok {
			v.Error("event does not implement json.Marshaler")
			continue
		}
		if nil != e.Intrinsics {
			e.Intrinsics = mergeAttributes(extraAttributes, e.Intrinsics)
		}
		expectEvent(v, event, e)
	}
}

// Second attributes have priority.
func mergeAttributes(a1, a2 map[string]interface{}) map[string]interface{} {
	a := make(map[string]interface{})
	for k, v := range a1 {
		a[k] = v
	}
	for k, v := range a2 {
		a[k] = v
	}
	return a
}

// ExpectErrorEvents allows testing of error events.  It passes if events exactly matches expect.
func ExpectErrorEvents(v Validator, events *errorEvents, expect []WantEvent) {
	expectEvents(v, events.analyticsEvents, expect, map[string]interface{}{
		// The following intrinsics should always be present in
		// error events:
		"type":      "TransactionError",
		"timestamp": MatchAnything,
		"duration":  MatchAnything,
	})
}

// ExpectSpanEvents allows testing of span events.  It passes if events exactly matches expect.
func ExpectSpanEvents(v Validator, events *spanEvents, expect []WantEvent) {
	expectEvents(v, events.analyticsEvents, expect, map[string]interface{}{
		// The following intrinsics should always be present in
		// span events:
		"type":          "Span",
		"timestamp":     MatchAnything,
		"duration":      MatchAnything,
		"traceId":       MatchAnything,
		"guid":          MatchAnything,
		"transactionId": MatchAnything,
		// All span events are currently sampled.
		"sampled":  true,
		"priority": MatchAnything,
	})
}

// ExpectTxnEvents allows testing of txn events.
func ExpectTxnEvents(v Validator, events *txnEvents, expect []WantEvent) {
	expectEvents(v, events.analyticsEvents, expect, map[string]interface{}{
		// The following intrinsics should always be present in
		// txn events:
		"type":      "Transaction",
		"timestamp": MatchAnything,
		"duration":  MatchAnything,
		"totalTime": MatchAnything,
		"error":     MatchAnything,
	})
}

func expectError(v Validator, err *tracedError, expect WantError) {
	validateStringField(v, "txnName", expect.TxnName, err.FinalName)
	validateStringField(v, "klass", expect.Klass, err.Klass)
	validateStringField(v, "msg", expect.Msg, err.Msg)
	js, errr := err.MarshalJSON()
	if nil != errr {
		v.Error("unable to marshal error json", errr)
		return
	}
	var unmarshalled []interface{}
	errr = json.Unmarshal(js, &unmarshalled)
	if nil != errr {
		v.Error("unable to unmarshal error json", errr)
		return
	}
	attributes := unmarshalled[4].(map[string]interface{})
	agentAttributes := attributes["agentAttributes"].(map[string]interface{})
	userAttributes := attributes["userAttributes"].(map[string]interface{})

	if nil != expect.UserAttributes {
		expectAttributes(v, userAttributes, expect.UserAttributes)
	}
	if nil != expect.AgentAttributes {
		expectAttributes(v, agentAttributes, expect.AgentAttributes)
	}
	if stack := attributes["stack_trace"]; nil == stack {
		v.Error("missing error stack trace")
	}
}

// ExpectErrors allows testing of errors.
func ExpectErrors(v Validator, errors harvestErrors, expect []WantError) {
	if len(errors) != len(expect) {
		v.Error("number of errors mismatch", len(errors), len(expect))
		return
	}
	for i, e := range expect {
		expectError(v, errors[i], e)
	}
}

func countSegments(node []interface{}) int {
	count := 1
	children := node[4].([]interface{})
	for _, c := range children {
		node := c.([]interface{})
		count += countSegments(node)
	}
	return count
}

func expectTraceSegment(v Validator, nodeObj interface{}, expect WantTraceSegment) {
	node := nodeObj.([]interface{})
	start := int(node[0].(float64))
	stop := int(node[1].(float64))
	name := node[2].(string)
	attributes := node[3].(map[string]interface{})
	children := node[4].([]interface{})

	validateStringField(v, "segmentName", expect.SegmentName, name)
	if nil != expect.RelativeStartMillis {
		expectStart, ok := expect.RelativeStartMillis.(int)
		if !ok {
			v.Error("invalid expect.RelativeStartMillis", expect.RelativeStartMillis)
		} else if expectStart != start {
			v.Error("segmentStartTime", expect.SegmentName, start, expectStart)
		}
	}
	if nil != expect.RelativeStopMillis {
		expectStop, ok := expect.RelativeStopMillis.(int)
		if !ok {
			v.Error("invalid expect.RelativeStopMillis", expect.RelativeStopMillis)
		} else if expectStop != stop {
			v.Error("segmentStopTime", expect.SegmentName, stop, expectStop)
		}
	}
	if nil != expect.Attributes {
		expectAttributes(v, attributes, expect.Attributes)
	}
	if len(children) != len(expect.Children) {
		v.Error("segmentChildrenCount", expect.SegmentName, len(children), len(expect.Children))
	} else {
		for idx, child := range children {
			expectTraceSegment(v, child, expect.Children[idx])
		}
	}
}

func expectTxnTrace(v Validator, got interface{}, expect WantTxnTrace) {
	unmarshalled := got.([]interface{})
	duration := unmarshalled[1].(float64)
	name := unmarshalled[2].(string)
	var arrayURL string
	if nil != unmarshalled[3] {
		arrayURL = unmarshalled[3].(string)
	}
	traceData := unmarshalled[4].([]interface{})

	rootNode := traceData[3].([]interface{})
	attributes := traceData[4].(map[string]interface{})
	userAttributes := attributes["userAttributes"].(map[string]interface{})
	agentAttributes := attributes["agentAttributes"].(map[string]interface{})
	intrinsics := attributes["intrinsics"].(map[string]interface{})

	validateStringField(v, "metric name", expect.MetricName, name)

	if d := expect.DurationMillis; nil != d && *d != duration {
		v.Error("incorrect trace duration millis", *d, duration)
	}

	if nil != expect.UserAttributes {
		expectAttributes(v, userAttributes, expect.UserAttributes)
	}
	if nil != expect.AgentAttributes {
		expectAttributes(v, agentAttributes, expect.AgentAttributes)
		expectURL, _ := expect.AgentAttributes["request.uri"].(string)
		if "" != expectURL {
			validateStringField(v, "request url in array", expectURL, arrayURL)
		}
	}
	if nil != expect.Intrinsics {
		expectAttributes(v, intrinsics, expect.Intrinsics)
	}
	if expect.Root.SegmentName != "" {
		expectTraceSegment(v, rootNode, expect.Root)
	} else {
		numSegments := countSegments(rootNode)
		// The expectation segment count does not include the two root nodes.
		numSegments -= 2
		if expect.NumSegments != numSegments {
			v.Error("wrong number of segments", expect.NumSegments, numSegments)
		}
	}
}

// ExpectTxnTraces allows testing of transaction traces.
func ExpectTxnTraces(v Validator, traces *harvestTraces, want []WantTxnTrace) {
	if len(want) != traces.Len() {
		v.Error("number of traces do not match", len(want), traces.Len())
		return
	}
	if len(want) == 0 {
		return
	}
	js, err := traces.Data("agentRunID", time.Now())
	if nil != err {
		v.Error("error creasing harvest traces data", err)
		return
	}

	var unmarshalled []interface{}
	err = json.Unmarshal(js, &unmarshalled)
	if nil != err {
		v.Error("unable to unmarshal error json", err)
		return
	}
	if "agentRunID" != unmarshalled[0].(string) {
		v.Error("traces agent run id wrong", unmarshalled[0])
		return
	}
	gotTraces := unmarshalled[1].([]interface{})
	if len(gotTraces) != len(want) {
		v.Error("number of traces in json does not match", len(gotTraces), len(want))
		return
	}
	for i, expected := range want {
		expectTxnTrace(v, gotTraces[i], expected)
	}
}

func expectSlowQuery(t Validator, slowQuery *slowQuery, want WantSlowQuery) {
	if slowQuery.Count != want.Count {
		t.Error("wrong Count field", slowQuery.Count, want.Count)
	}
	uri, _ := slowQuery.TxnEvent.Attrs.GetAgentValue(attributeRequestURI, destTxnTrace)
	validateStringField(t, "MetricName", slowQuery.DatastoreMetric, want.MetricName)
	validateStringField(t, "Query", slowQuery.ParameterizedQuery, want.Query)
	validateStringField(t, "TxnEvent.FinalName", slowQuery.TxnEvent.FinalName, want.TxnName)
	validateStringField(t, "request.uri", uri, want.TxnURL)
	validateStringField(t, "DatabaseName", slowQuery.DatabaseName, want.DatabaseName)
	validateStringField(t, "Host", slowQuery.Host, want.Host)
	validateStringField(t, "PortPathOrID", slowQuery.PortPathOrID, want.PortPathOrID)
	expectAttributes(t, map[string]interface{}(slowQuery.QueryParameters), want.Params)
}

// ExpectSlowQueries allows testing of slow queries.
func ExpectSlowQueries(t Validator, slowQueries *slowQueries, want []WantSlowQuery) {
	if len(want) != len(slowQueries.priorityQueue) {
		t.Error("wrong number of slow queries",
			"expected", len(want), "got", len(slowQueries.priorityQueue))
		return
	}
	for _, s := range want {
		idx, ok := slowQueries.lookup[s.Query]
		if !ok {
			t.Error("unable to find slow query", s.Query)
			continue
		}
		expectSlowQuery(t, slowQueries.priorityQueue[idx], s)
	}
}
