package newrelic

import (
	"runtime"

	"github.com/newrelic/go-agent/v3/internal"
)

const (
	// Version is the full string version of this Go Agent.
	Version = "3.6.0"
)

var (
	goVersionSimple = minorVersion(runtime.Version())
)

func init() {
	internal.TrackUsage("Go", "Version", Version)
	internal.TrackUsage("Go", "Runtime", "Version", goVersionSimple)
}
