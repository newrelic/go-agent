package newrelic

import "time"

type traceBox struct {
	messages chan *spanEvent
}

const (
	apiKeyMetadataKey        = "api_key"
	traceboxMessageQueueSize = 1000
)

var (
	traceBoxBackoffStrategy = []time.Duration{
		15 * time.Second,
		15 * time.Second,
		30 * time.Second,
		60 * time.Second,
		120 * time.Second,
		300 * time.Second,
	}
)
