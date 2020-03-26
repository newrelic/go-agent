package newrelic

import (
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
