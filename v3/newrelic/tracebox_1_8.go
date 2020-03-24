// +build !go1.9

package newrelic

import (
	"errors"

	"github.com/newrelic/go-agent/v3/internal"
)

func newTraceBox(endpoint, apiKey string, runID internal.AgentRunID, lg Logger, connected chan<- bool) (*traceBox, error) {
	return nil, errors.New("Non supported Go version - to use Magic Trace Box, " +
		"you must use at least version 1.9 or higher of Go.")
}

func (tb *traceBox) consumeSpan(span *spanEvent) bool {
	return false
}
