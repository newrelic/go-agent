// Package nrlogxi forwards go-agent log messages to mgutz/logxi.  If you
// are using mgutz/logxi for your application and would like the go-agent log
// messages to end up in the same place, create a new logging.Logger and
// initialize the  NR integration.
//
//  l := log.New("newrelic")
//	cfg.Logger = nrlogxi.New(l)
//
package nrlogxi

import (
	"github.com/mgutz/logxi/v1"
	newrelic "github.com/newrelic/go-agent"
)

type shim struct {
	e log.Logger
}

func (l *shim) Error(msg string, context map[string]interface{}) {
	c := convert(context)
	l.e.Error(msg, c...)
}
func (l *shim) Warn(msg string, context map[string]interface{}) {
	c := convert(context)
	l.e.Warn(msg, c...)
}
func (l *shim) Info(msg string, context map[string]interface{}) {
	c := convert(context)
	l.e.Info(msg, c...)
}
func (l *shim) Debug(msg string, context map[string]interface{}) {
	c := convert(context)
	l.e.Debug(msg, c...)
}
func (l *shim) DebugEnabled() bool {
	return l.e.IsDebug()
}

// With the context provided by the agent, convert to key/value and
// append to a slice of interfaces.
func convert(c map[string]interface{}) (output []interface{}) {
	output = make([]interface{}, 0, len(c))
	for k, v := range c {
		output = append(output, k, v)
	}
	return output
}

// New returns a newrelic.Logger which forwards agent log messages to the
// logging package-level exported logger.
func New(l log.Logger) newrelic.Logger {
	return &shim{
		e: l,
	}
}
