// Package nrlog15 forwards go-agent log messages to inconshreveable/log15.  If you would
// like to use inconshreveable/log15 for go-agent log messages, wrap your log15 Logger
// using nrlog15.New to create a newrelic.Logger.
//
//	l := log.New("component", "newrelic")
//	cfg.Logger = nrlog15.New(l, true)
//
package nrlog15

import (
	log "github.com/inconshreveable/log15"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

func init() { internal.TrackUsage("integration", "logging", "log15") }

type shim struct {
	e     log.Logger
	debug bool // Unfortunately necessary as log15 does not provide a way to get the current log level
}

func (l *shim) Error(msg string, context map[string]interface{}) {
	l.e.Error(msg, log.Ctx(context))
}
func (l *shim) Warn(msg string, context map[string]interface{}) {
	l.e.Warn(msg, log.Ctx(context))
}
func (l *shim) Info(msg string, context map[string]interface{}) {
	l.e.Info(msg, log.Ctx(context))
}
func (l *shim) Debug(msg string, context map[string]interface{}) {
	l.e.Debug(msg, log.Ctx(context))
}
func (l *shim) DebugEnabled() bool {
	return l.debug
}

// New returns a newrelic.Logger which forwards agent log messages to the
// provided log15 Logger.
func New(logger log.Logger, debug bool) newrelic.Logger {
	return &shim{
		e:     logger,
		debug: debug,
	}
}

func StandardLogger() newrelic.Logger {
	logger := log.New("component", "newrelic")

	return &shim{
		e:     logger,
		debug: true,
	}
}
