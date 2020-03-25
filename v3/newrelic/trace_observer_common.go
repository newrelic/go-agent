package newrelic

import (
	"time"

	"github.com/newrelic/go-agent/v3/internal"
)

type traceObserver struct {
	messages chan *spanEvent
}

type observerConfig struct {
	endpoint  string
	license   string
	runID     internal.AgentRunID
	log       Logger
	connected chan<- bool
}

const (
	licenseMetadataKey            = "license_key"
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
