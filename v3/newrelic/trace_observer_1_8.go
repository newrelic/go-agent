// +build !go1.9

package newrelic

import (
	"errors"

	"github.com/newrelic/go-agent/v3/internal"
)

func newTraceObserver(cfg observerConfig) (*traceObserver, error) {
	return nil, errors.New("Non supported Go version - to use Infinite Tracing, " +
		"you must use at least version 1.9 or higher of Go.")
}

func (to *traceObserver) consumeSpan(span *spanEvent) bool {
	return false
}

func (to *traceObserver) getConnectedState() bool {
	return false
}

func expectObserverEvents(v internal.Validator, events *analyticsEvents, expect []internal.WantEvent, extraAttributes map[string]interface{}) {
}
