// Package nrlogapex forwards go-agent log messages to apex/log.  If you would
// like to use apex/log for go-agent log messages, wrap your Logger using
// nrlogapex.New to create a newrelic.Logger.
//
//  l := log.Logger{}
//	l.SetLevel(log.LevelInfo)
//	cfg.Logger = nrlogapex.New(l)
//
//  Or
//
//  cfg.Logger = nrlogapex.StandardLogger()
//
package nrlogapex

import (
	"github.com/apex/log"
	newrelic "github.com/newrelic/go-agent"
)

type shim struct {
	l log.Interface
}

func (n *shim) Error(msg string, context map[string]interface{}) {
	n.l.WithFields(log.Fields(context)).Error(msg)
}

func (n *shim) Warn(msg string, context map[string]interface{}) {
	n.l.WithFields(log.Fields(context)).Warn(msg)
}

func (n *shim) Info(msg string, context map[string]interface{}) {
	n.l.WithFields(log.Fields(context)).Info(msg)
}

func (n *shim) Debug(msg string, context map[string]interface{}) {
	n.l.WithFields(log.Fields(context)).Debug(msg)
}

func (n *shim) DebugEnabled() bool {
	switch n.l.(type) {
	case *log.Entry:
		l := n.l.(*log.Entry)
		return l.Level == log.DebugLevel
	case *log.Logger:
		l := n.l.(*log.Logger)
		return l.Level == log.DebugLevel
	}

	return false
}

// New returns a newrelic.Logger which forwards agent log messages to the
// provided apex log.Interface (*log.Entry / *log.Logger).
func New(l log.Interface) newrelic.Logger {
	return &shim{
		l: l.WithField("component", "newrelic"),
	}
}

// StandardLogger returns a newrelic.Logger which forwards agent log messages to
// the apex/log package-level exported logger.
func StandardLogger() newrelic.Logger {
	return &shim{
		l: log.Log.WithField("component", "newrelic"),
	}
}
