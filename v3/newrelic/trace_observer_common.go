package newrelic

import (
	"errors"
	"sync"

	"github.com/newrelic/go-agent/v3/internal"
)

type traceObserver struct {
	messages chan *spanEvent
	// once protects messages from being closed multiple times.
	once sync.Once

	initialConnSuccess chan struct{}
	restart            chan internal.AgentRunID
	initiateShutdown   chan struct{}
	shutdownComplete   chan struct{}
	runID              internal.AgentRunID

	supportability *observerSupport

	observerConfig
}

type observerConfig struct {
	endpoint    *observerURL
	license     string
	log         Logger
	queueSize   int
	appShutdown <-chan struct{}
}

type observerURL struct {
	host   string
	secure bool
}

type observerSupport struct {
	increment chan string
	dump      chan map[string]float64
}

const (
	licenseMetadataKey    = "license_key"
	runIDMetadataKey      = "agent_run_token"
	localTestingHost      = "localhost"
	infTracingDefaultPort = 443
)

var (
	errUnsupportedVersion = errors.New("non supported Go version - to use Infinite Tracing, " +
		"you must use at least version 1.9 or higher of Go")

	errSpanOrDTDisabled = errors.New("in order to enable Infinite Tracing, you must have both " +
		"Distributed Tracing and Span Events enabled")
)
