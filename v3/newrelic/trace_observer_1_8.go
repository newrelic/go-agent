// +build !go1.9

package newrelic

import (
	"github.com/newrelic/go-agent/v3/internal"
)

func newTraceObserver(runID internal.AgentRunID, cfg observerConfig) (traceObserver, error) {
	return nil, errUnsupportedVersion
}

// versionSupports8T records whether we are using a supported version of Go for
// Infinite Tracing
const versionSupports8T = false

func expectObserverEvents(v internal.Validator, events *analyticsEvents, expect []internal.WantEvent, extraAttributes map[string]interface{}) {
}
