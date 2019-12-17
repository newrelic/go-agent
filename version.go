package newrelic

import (
	"runtime"

	"github.com/newrelic/go-agent/internal"
)

const (
	major = "2"
	minor = "16"
	patch = "4"

	// Version is the full string version of this Go Agent.
	Version = major + "." + minor + "." + patch
)

func init() {
	internal.TrackUsage("Go", "Version", Version)
	internal.TrackUsage("Go", "Runtime", "Version", internal.MinorVersion(runtime.Version()))
}
