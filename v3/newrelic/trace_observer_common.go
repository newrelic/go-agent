package newrelic

import (
	"errors"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
)

type traceObserver interface {
	// restart TODO
	restart(internal.AgentRunID)
	// shutdown TODO
	shutdown(time.Duration) error
	// consumeSpan TODO
	consumeSpan(*spanEvent)
	// dumpSupportabilityMetrics TODO
	dumpSupportabilityMetrics() map[string]float64
	// initialConnCompleted TODO - does NOT indicate current state of connection
	initialConnCompleted() bool
}

type observerConfig struct {
	endpoint    *observerURL
	license     string
	log         Logger
	queueSize   int
	appShutdown chan struct{}
	dialer      internal.DialerFunc
	// removeBackoff sets the recordSpanBackoff to 0 and is useful for testing
	removeBackoff bool
}

type observerURL struct {
	host   string
	secure bool
}

const (
	localTestingHost      = "localhost"
	infTracingDefaultPort = 443
)

var (
	errUnsupportedVersion = errors.New("non supported Go version - to use Infinite Tracing, " +
		"you must use at least version 1.9 or higher of Go")

	errSpanOrDTDisabled = errors.New("in order to enable Infinite Tracing, you must have both " +
		"Distributed Tracing and Span Events enabled")
)
