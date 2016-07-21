package internal

import (
	"time"

	"github.com/newrelic/go-agent/datastore"
)

// Token tracks segments.
type Token uint64

const (
	// Maximum number of traced function calls is 2^tokenStampBits.
	// Maximum number stack depth is 2^(64-tokenStampBits)
	tokenStampBits          = 40
	invalidToken            = Token(0)
	startingStackDepthAlloc = 128

	datastoreProductUnknown = datastore.Product("Unknown")
)

func createToken(depth int, stamp uint64) Token {
	token := (uint64(depth) << tokenStampBits) | stamp
	return Token(token)
}

func parseToken(token Token) (depth int, stamp uint64) {
	stamp = uint64((1<<tokenStampBits)-1) & uint64(token)
	depth = int(token >> tokenStampBits)
	return
}

type segmentFrame struct {
	stamp    uint64
	start    time.Time
	children time.Duration
}

// Tracer tracks segments.
type Tracer struct {
	finishedChildren time.Duration
	stamp            uint64
	currentDepth     int
	stack            []segmentFrame

	customSegments    map[string]*metricData
	datastoreSegments map[datastoreMetricKey]*metricData
	externalSegments  map[externalMetricKey]*metricData

	DatastoreExternalTotals
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
func StartSegment(t *Tracer, now time.Time) Token {
	if nil == t.stack {
		t.stack = make([]segmentFrame, startingStackDepthAlloc)
	}
	if cap(t.stack) == t.currentDepth {
		newLimit := 2 * t.currentDepth
		newStack := make([]segmentFrame, newLimit)
		copy(newStack, t.stack)
		t.stack = newStack
	}

	// Update the stamp before using it so that a 0 stamp can be special.
	t.stamp++

	idx := t.currentDepth
	stamp := t.stamp
	t.currentDepth++

	t.stack[idx].start = now
	t.stack[idx].children = 0
	t.stack[idx].stamp = stamp

	return createToken(idx, stamp)
}

type segmentEnd struct {
	valid     bool
	start     time.Time
	stop      time.Time
	duration  time.Duration
	exclusive time.Duration
}

func endSegment(t *Tracer, token Token, now time.Time) segmentEnd {
	var s segmentEnd
	depth, stamp := parseToken(token)
	if 0 == stamp {
		return s
	}
	if depth >= t.currentDepth {
		return s
	}
	if depth < 0 {
		return s
	}
	if stamp != t.stack[depth].stamp {
		return s
	}

	var children time.Duration
	for i := depth; i < t.currentDepth; i++ {
		children += t.stack[i].children
	}
	s.valid = true
	s.stop = now
	s.start = t.stack[depth].start
	if s.stop.After(s.start) {
		s.duration = s.stop.Sub(s.start)
	}
	if s.duration > children {
		s.exclusive = s.duration - children
	}

	// Note that we expect (depth == (t.currentDepth - 1)).  However, if
	// (depth < (t.currentDepth - 1)), that's ok: could be a panic popped
	// some stack frames (and the consumer was not using defer).
	t.currentDepth = depth

	if 0 == t.currentDepth {
		t.finishedChildren += s.duration
	} else {
		t.stack[t.currentDepth-1].children += s.duration
	}
	return s
}

// EndBasicSegment ends a basic segment.
func EndBasicSegment(t *Tracer, token Token, now time.Time, name string) {
	end := endSegment(t, token, now)
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
}

// EndExternalSegment ends an external segment.
func EndExternalSegment(t *Tracer, token Token, now time.Time, host string) {
	end := endSegment(t, token, now)
	if !end.valid {
		return
	}
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
}

// EndDatastoreSegment ends a datastore segment.
func EndDatastoreSegment(t *Tracer, token Token, now time.Time, s datastore.Segment) {
	end := endSegment(t, token, now)
	if !end.valid {
		return
	}
	key := datastoreMetricKey{
		Product:    s.Product,
		Collection: s.Collection,
		Operation:  s.Operation,
	}
	if key.Operation == "" {
		key.Operation = "other"
	}
	if key.Product == "" {
		key.Product = datastoreProductUnknown
	}
	if nil == t.datastoreSegments {
		t.datastoreSegments = make(map[datastoreMetricKey]*metricData)
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
}

// MergeBreakdownMetrics creates segment metrics.
func MergeBreakdownMetrics(t *Tracer, metrics *metricTable, scope string, isWeb bool) {
	// Custom Segment Metrics
	for key, data := range t.customSegments {
		name := customSegmentPrefix + key
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
