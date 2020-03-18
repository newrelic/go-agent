// +build !go1.9

package newrelic

import "errors"

func newTraceBox(endpoint, apiKey string, lg Logger) (*traceBox, error) {
	return nil, errors.New("Non supported Go version - to use Magic Trace Box, " +
		"you must use at least version 1.9 or higher of Go.")
}

func (tb *traceBox) consumeSpan(span *spanEvent) bool {
	return false
}
