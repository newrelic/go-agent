// Package nrlogrus forwards go-agent log messages to logrus.  If you are using
// logrus for your application and would like the go-agent log messages to end
// up in the same place, simply import this package for the side effects:
//
//	import _ "github.com/newrelic/go-agent/log/_nrlogrus"
//
package nrlogrus

import (
	"github.com/Sirupsen/logrus"
	"github.com/newrelic/go-agent/log"
)

type shim struct {
	e *logrus.Entry
}

func (s *shim) Fire(e log.Entry) {
	wf := s.e.WithFields(logrus.Fields(e.Context))

	switch e.Level {
	case log.LevelError:
		wf.Error(e.Event)
	case log.LevelWarning:
		wf.Warning(e.Event)
	case log.LevelInfo:
		wf.Info(e.Event)
	case log.LevelDebug:
		wf.Debug(e.Event)
	}
}

func init() {
	log.Logger = &shim{
		e: logrus.WithFields(logrus.Fields{
			"component": "newrelic",
		}),
	}
}
