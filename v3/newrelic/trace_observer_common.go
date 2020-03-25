package newrelic

import "time"

type traceObserver struct {
	messages chan *spanEvent
}

const (
	apiKeyMetadataKey             = "api_key"
	runIDMetadataKey              = "agent_run_token"
	traceObserverMessageQueueSize = 1000
)

var (
	infiniteTracingBackoffStrategy = []time.Duration{
		15 * time.Second,
		15 * time.Second,
		30 * time.Second,
		60 * time.Second,
		120 * time.Second,
		300 * time.Second,
	}
)
