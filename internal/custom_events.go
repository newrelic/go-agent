package internal

import (
	"math/rand"
	"time"
)

type customEvents struct {
	events *analyticsEvents
}

func newCustomEvents(max int) *customEvents {
	return &customEvents{
		events: newAnalyticsEvents(max),
	}
}

func (cs *customEvents) Add(e *customEvent) {
	stamp := eventStamp(rand.Float32())
	cs.events.AddEvent(analyticsEvent{stamp, e})
}

func (cs *customEvents) mergeIntoHarvest(h *harvest) {
	h.customEvents.events.MergeFailed(cs.events)
}

func (cs *customEvents) Data(agentRunID string, harvestStart time.Time) ([]byte, error) {
	return cs.events.CollectorJSON(agentRunID)
}

func (cs *customEvents) numSeen() float64  { return cs.events.NumSeen() }
func (cs *customEvents) numSaved() float64 { return cs.events.NumSaved() }
