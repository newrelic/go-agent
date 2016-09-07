package internal

import (
	"net/url"
	"time"
)

type segmentStamp uint64

type segmentTime struct {
	Stamp segmentStamp
	Time  time.Time
}

// SegmentStartTime is embedded into the top level segments (rather than
// segmentTime) to minimize the structure sizes to minimize allocations.
type SegmentStartTime struct {
	Stamp segmentStamp
	Depth int
}

type segmentFrame struct {
	segmentTime
	children time.Duration
}

type segmentEnd struct {
	valid     bool
	start     segmentTime
	stop      segmentTime
	duration  time.Duration
	exclusive time.Duration
}

// Tracer tracks segments.
type Tracer struct {
	finishedChildren time.Duration
	stamp            segmentStamp
	currentDepth     int
	stack            []segmentFrame

	customSegments    map[string]*metricData
	datastoreSegments map[DatastoreMetricKey]*metricData
	externalSegments  map[externalMetricKey]*metricData

	DatastoreExternalTotals

	TxnTrace
}

const (
	startingStackDepthAlloc   = 128
	datastoreProductUnknown   = "Unknown"
	datastoreOperationUnknown = "other"
)

func (t *Tracer) time(now time.Time) segmentTime {
	// Update the stamp before using it so that a 0 stamp can be special.
	t.stamp++
	return segmentTime{
		Time:  now,
		Stamp: t.stamp,
	}
}

// TracerRootChildren is used to calculate a transaction's exclusive duration.
func TracerRootChildren(t *Tracer) time.Duration {
	var lostChildren time.Duration
	for i := 0; i < t.currentDepth; i++ {
		lostChildren += t.stack[i].children
	}
	return t.finishedChildren + lostChildren
}

// StartSegment begins a segment.
func StartSegment(t *Tracer, now time.Time) SegmentStartTime {
	if nil == t.stack {
		t.stack = make([]segmentFrame, startingStackDepthAlloc)
	}
	if cap(t.stack) == t.currentDepth {
		newLimit := 2 * t.currentDepth
		newStack := make([]segmentFrame, newLimit)
		copy(newStack, t.stack)
		t.stack = newStack
	}

	tm := t.time(now)

	depth := t.currentDepth
	t.currentDepth++
	t.stack[depth].children = 0
	t.stack[depth].segmentTime = tm

	return SegmentStartTime{
		Stamp: tm.Stamp,
		Depth: depth,
	}
}

func endSegment(t *Tracer, start SegmentStartTime, now time.Time) segmentEnd {
	var s segmentEnd
	if 0 == start.Stamp {
		return s
	}
	if start.Depth >= t.currentDepth {
		return s
	}
	if start.Depth < 0 {
		return s
	}
	if start.Stamp != t.stack[start.Depth].Stamp {
		return s
	}

	var children time.Duration
	for i := start.Depth; i < t.currentDepth; i++ {
		children += t.stack[i].children
	}
	s.valid = true
	s.stop = t.time(now)
	s.start = t.stack[start.Depth].segmentTime
	if s.stop.Time.After(s.start.Time) {
		s.duration = s.stop.Time.Sub(s.start.Time)
	}
	if s.duration > children {
		s.exclusive = s.duration - children
	}

	// Note that we expect (depth == (t.currentDepth - 1)).  However, if
	// (depth < (t.currentDepth - 1)), that's ok: could be a panic popped
	// some stack frames (and the consumer was not using defer).
	t.currentDepth = start.Depth

	if 0 == t.currentDepth {
		t.finishedChildren += s.duration
	} else {
		t.stack[t.currentDepth-1].children += s.duration
	}
	return s
}

// EndBasicSegment ends a basic segment.
func EndBasicSegment(t *Tracer, start SegmentStartTime, now time.Time, name string) {
	end := endSegment(t, start, now)
	if !end.valid {
		return
	}
	if nil == t.customSegments {
		t.customSegments = make(map[string]*metricData)
	}
	m := metricDataFromDuration(end.duration, end.exclusive)
	if data, ok := t.customSegments[name]; ok {
		data.aggregate(m)
	} else {
		// Use `new` in place of &m so that m is not
		// automatically moved to the heap.
		cpy := new(metricData)
		*cpy = m
		t.customSegments[name] = cpy
	}

	if t.TxnTrace.considerNode(end) {
		t.TxnTrace.witnessNode(end, customSegmentMetric(name), nil)
	}
}

// EndExternalSegment ends an external segment.
func EndExternalSegment(t *Tracer, start SegmentStartTime, now time.Time, u *url.URL) {
	end := endSegment(t, start, now)
	if !end.valid {
		return
	}
	host := HostFromURL(u)
	if "" == host {
		host = "unknown"
	}
	key := externalMetricKey{
		Host: host,
		ExternalCrossProcessID:  "",
		ExternalTransactionName: "",
	}
	if nil == t.externalSegments {
		t.externalSegments = make(map[externalMetricKey]*metricData)
	}
	t.externalCallCount++
	t.externalDuration += end.duration
	m := metricDataFromDuration(end.duration, end.exclusive)
	if data, ok := t.externalSegments[key]; ok {
		data.aggregate(m)
	} else {
		// Use `new` in place of &m so that m is not
		// automatically moved to the heap.
		cpy := new(metricData)
		*cpy = m
		t.externalSegments[key] = cpy
	}

	if t.TxnTrace.considerNode(end) {
		t.TxnTrace.witnessNode(end, externalHostMetric(key), map[string]interface{}{
			"uri": SafeURL(u),
		})
	}
}

// EndDatastoreSegment ends a datastore segment.
func EndDatastoreSegment(t *Tracer, start SegmentStartTime, now time.Time, key DatastoreMetricKey) {
	end := endSegment(t, start, now)
	if !end.valid {
		return
	}
	if key.Operation == "" {
		key.Operation = datastoreOperationUnknown
	}
	if key.Product == "" {
		key.Product = datastoreProductUnknown
	}
	if nil == t.datastoreSegments {
		t.datastoreSegments = make(map[DatastoreMetricKey]*metricData)
	}
	t.datastoreCallCount++
	t.datastoreDuration += end.duration
	m := metricDataFromDuration(end.duration, end.exclusive)
	if data, ok := t.datastoreSegments[key]; ok {
		data.aggregate(m)
	} else {
		// Use `new` in place of &m so that m is not
		// automatically moved to the heap.
		cpy := new(metricData)
		*cpy = m
		t.datastoreSegments[key] = cpy
	}

	if t.TxnTrace.considerNode(end) {
		var name string
		if "" != key.Collection {
			name = datastoreStatementMetric(key)
		} else {
			name = datastoreOperationMetric(key)
		}
		t.TxnTrace.witnessNode(end, name, nil)
	}
}

// MergeBreakdownMetrics creates segment metrics.
func MergeBreakdownMetrics(t *Tracer, metrics *metricTable, scope string, isWeb bool) {
	// Custom Segment Metrics
	for key, data := range t.customSegments {
		name := customSegmentMetric(key)
		// Unscoped
		metrics.add(name, "", *data, unforced)
		// Scoped
		metrics.add(name, scope, *data, unforced)
	}

	// External Segment Metrics
	for key, data := range t.externalSegments {
		metrics.add(externalAll, "", *data, forced)
		if isWeb {
			metrics.add(externalWeb, "", *data, forced)
		} else {
			metrics.add(externalOther, "", *data, forced)
		}
		hostMetric := externalHostMetric(key)
		metrics.add(hostMetric, "", *data, unforced)
		if "" != key.ExternalCrossProcessID && "" != key.ExternalTransactionName {
			txnMetric := externalTransactionMetric(key)

			// Unscoped CAT metrics
			metrics.add(externalAppMetric(key), "", *data, unforced)
			metrics.add(txnMetric, "", *data, unforced)

			// Scoped External Metric
			metrics.add(txnMetric, scope, *data, unforced)
		} else {
			// Scoped External Metric
			metrics.add(hostMetric, scope, *data, unforced)
		}
	}

	// Datastore Segment Metrics
	for key, data := range t.datastoreSegments {
		metrics.add(datastoreAll, "", *data, forced)

		product := datastoreProductMetric(key)
		metrics.add(product.All, "", *data, forced)
		if isWeb {
			metrics.add(datastoreWeb, "", *data, forced)
			metrics.add(product.Web, "", *data, forced)
		} else {
			metrics.add(datastoreOther, "", *data, forced)
			metrics.add(product.Other, "", *data, forced)
		}

		operation := datastoreOperationMetric(key)
		metrics.add(operation, "", *data, unforced)

		if "" != key.Collection {
			statement := datastoreStatementMetric(key)

			metrics.add(statement, "", *data, unforced)
			metrics.add(statement, scope, *data, unforced)
		} else {
			metrics.add(operation, scope, *data, unforced)
		}
	}
}
