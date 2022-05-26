// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"container/heap"
	"time"

	"github.com/newrelic/go-agent/v3/internal/jsonx"
)

type commonAttributes struct {
	entityGUID string
	entityName string
	hostname   string
}

type logEventHeap []logEvent

type logEvents struct {
	numSeen        int
	failedHarvests int
	severityCount  map[string]int
	commonAttributes
	config loggingConfig
	logs   logEventHeap
}

// NumSeen returns the number of events seen
func (events *logEvents) NumSeen() float64 { return float64(events.numSeen) }

// NumSaved returns the number of events that will be harvested for this cycle
func (events *logEvents) NumSaved() float64 { return float64(len(events.logs)) }

func (events *logEvents) RecordLoggingMetrics(metrics *metricTable, forced metricForce) {
	if events.config.collectMetrics && metrics != nil {
		metrics.addCount(logsSeen, events.NumSeen(), forced)
		for k, v := range events.severityCount {
			severitySeen := logsSeen + "/" + k
			metrics.addCount(severitySeen, float64(v), forced)
		}
	}

	if events.config.collectEvents {
		metrics.addCount(logsDropped, events.NumSeen()-events.NumSaved(), forced)
	}
}

// TODO: when go 1.18 becomes the minimum supported version, re-write to make a generic heap implementation
// for all event heaps, to de-duplicate this code
//func (events *logEvents)
func (h logEventHeap) Len() int           { return len(h) }
func (h logEventHeap) Less(i, j int) bool { return h[i].priority.isLowerPriority(h[j].priority) }
func (h logEventHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

// Push and Pop are unused: only heap.Init and heap.Fix are used.
func (h logEventHeap) Push(x interface{}) {}
func (h logEventHeap) Pop() interface{}   { return nil }

func newLogEvents(ca commonAttributes, loggingConfig loggingConfig) *logEvents {
	return &logEvents{
		commonAttributes: ca,
		config:           loggingConfig,
		severityCount:    map[string]int{},
		logs:             make(logEventHeap, 0, loggingConfig.maxLogEvents),
	}
}

func (events *logEvents) capacity() int {
	return events.config.maxLogEvents
}

func (events *logEvents) Add(e *logEvent) {
	// always collect this but do not report logging metrics when disabled
	events.numSeen++
	events.severityCount[e.severity] += 1

	// Do not collect log events when the harvest capacity is intentionally set to 0
	// or the collection of events is explicitly disabled
	if events.capacity() == 0 || !events.config.collectEvents {
		// Configurable event harvest limits may be zero.
		return
	}

	if len(events.logs) < cap(events.logs) {
		// copy log event onto event heap
		events.logs = append(events.logs, *e)
		if len(events.logs) == cap(events.logs) {
			// Delay heap initialization so that we can have
			// deterministic ordering for integration tests (the max
			// is not being reached).
			heap.Init(events.logs)
		}
		return
	}

	if e.priority.isLowerPriority((events.logs)[0].priority) {
		return
	}

	events.logs[0] = *e
	heap.Fix(events.logs, 0)
}

func (events *logEvents) mergeFailed(other *logEvents) {
	fails := other.failedHarvests + 1
	if fails >= failedEventsAttemptsLimit {
		return
	}
	events.failedHarvests = fails
	events.Merge(other)
}

func (events *logEvents) Merge(other *logEvents) {
	allSeen := events.numSeen + other.numSeen

	for _, e := range other.logs {
		events.Add(&e)
	}
	events.numSeen = allSeen
}

func (events *logEvents) CollectorJSON(agentRunID string) ([]byte, error) {
	if 0 == len(events.logs) {
		return nil, nil
	}

	estimate := 256 * len(events.logs)
	buf := bytes.NewBuffer(make([]byte, 0, estimate))

	if events.numSeen == 0 {
		return nil, nil
	}

	buf.WriteByte('[')
	buf.WriteByte('{')
	buf.WriteString(`"common":`)
	buf.WriteByte('{')
	buf.WriteString(`"attributes":`)
	buf.WriteByte('{')
	buf.WriteString(`"entity.guid":`)
	jsonx.AppendString(buf, events.entityGUID)
	buf.WriteByte(',')
	buf.WriteString(`"entity.name":`)
	jsonx.AppendString(buf, events.entityName)
	buf.WriteByte(',')
	buf.WriteString(`"hostname":`)
	jsonx.AppendString(buf, events.hostname)
	buf.WriteByte('}')
	buf.WriteByte('}')
	buf.WriteByte(',')
	buf.WriteString(`"logs":`)
	buf.WriteByte('[')
	for i, e := range events.logs {
		// If severity is empty string, then this is not a user provided entry, and is empty.
		// Do not write json to buffer in this case.
		if e.severity != "" {
			e.WriteJSON(buf)
			if i != len(events.logs)-1 {
				buf.WriteByte(',')
			}
		}

	}
	buf.WriteByte(']')
	buf.WriteByte('}')
	buf.WriteByte(']')
	return buf.Bytes(), nil

}

// split splits the events into two.  NOTE! The two event pools are not valid
// priority queues, and should only be used to create JSON, not for adding any
// events.
func (events *logEvents) split() (*logEvents, *logEvents) {
	// numSeen is conserved: e1.numSeen + e2.numSeen == events.numSeen.
	sc1, sc2 := splitSeverityCount(events.severityCount)
	e1 := &logEvents{
		numSeen:          len(events.logs) / 2,
		failedHarvests:   events.failedHarvests / 2,
		severityCount:    sc1,
		commonAttributes: events.commonAttributes,
		logs:             make([]logEvent, len(events.logs)/2),
	}
	e2 := &logEvents{
		numSeen:          events.numSeen - e1.numSeen,
		failedHarvests:   events.failedHarvests - e1.failedHarvests,
		severityCount:    sc2,
		commonAttributes: events.commonAttributes,
		logs:             make([]logEvent, len(events.logs)-len(e1.logs)),
	}
	// Note that slicing is not used to ensure that length == capacity for
	// e1.events and e2.events.
	copy(e1.logs, events.logs)
	copy(e2.logs, events.logs[len(events.logs)/2:])

	return e1, e2
}

// splits the contents and counts of the severity map
func splitSeverityCount(severityCount map[string]int) (map[string]int, map[string]int) {
	count1 := map[string]int{}
	count2 := map[string]int{}
	for k, v := range severityCount {
		count1[k] = v / 2
		count2[k] = v - count1[k]
	}
	return count1, count2
}

func (events *logEvents) MergeIntoHarvest(h *harvest) {
	h.LogEvents.mergeFailed(events)
}

func (events *logEvents) Data(agentRunID string, harvestStart time.Time) ([]byte, error) {
	return events.CollectorJSON(agentRunID)
}

func (events *logEvents) EndpointMethod() string {
	return cmdLogEvents
}
