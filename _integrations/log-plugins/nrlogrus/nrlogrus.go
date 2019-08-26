package nrlogrus

import (
	newrelic "github.com/newrelic/go-agent"
	logcontext "github.com/newrelic/go-agent/_integrations/log-plugins"
	"github.com/newrelic/go-agent/internal"
	"github.com/sirupsen/logrus"
)

func init() { internal.TrackUsage("integration", "log-context", "logrus") }

type nrFormatter struct {
	jsonFormatter logrus.JSONFormatter
}

func (f nrFormatter) Format(e *logrus.Entry) ([]byte, error) {
	next := e.WithField(logcontext.KeyTimestamp, uint64(e.Time.UnixNano())/uint64(1000*1000))

	if ctx := e.Context; nil != ctx {
		if txn := newrelic.FromContext(ctx); nil != txn {
			next = next.WithFields(logrus.Fields(txn.GetLinkingMetadata().Map()))
		}
	}

	next.Level = e.Level
	next.Message = e.Message
	next.Caller = e.Caller
	next.Buffer = e.Buffer
	return f.jsonFormatter.Format(next)
}

// NewFormatter TODO
func NewFormatter() logrus.Formatter {
	return nrFormatter{
		jsonFormatter: logrus.JSONFormatter{
			DisableTimestamp: true,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyMsg:   logcontext.KeyMessage,
				logrus.FieldKeyLevel: logcontext.KeyLevel,
				logrus.FieldKeyFunc:  logcontext.KeyMethod,
				logrus.FieldKeyFile:  logcontext.KeyFile,
				// TODO: Split file and line number?
			},
		},
	}
}
