package newrelic

import (
	"errors"
	"sync"

	"github.com/newrelic/go-agent/v3/internal"
)

type traceObserver struct {
	messages chan *spanEvent

	// This mutex protects `connected`, which should be accessed via `getConnectedState` and `setConnectedState`
	sync.Mutex
	connected bool
}

type observerConfig struct {
	endpoint  *observerURL
	license   string
	runID     internal.AgentRunID
	log       Logger
	queueSize int
}

type observerURL struct {
	host   string
	secure bool
}

const (
	licenseMetadataKey = "license_key"
	runIDMetadataKey   = "agent_run_token"
)

var errUnsupportedVersion = errors.New("non supported Go version - to use Infinite Tracing, " +
	"you must use at least version 1.9 or higher of Go")
