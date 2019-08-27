// Package nrlogrus decorates logs for sending to the New Relic backend.
//
// Use this package if you are wishing to enable the New Relic logging product
// and see your log messages in the New Relic UI.
//
// To enable, set your log's formatter to the `nrlogrus.NewFormatter()`
//
//	logger := logrus.New()
//	logger.Formatter = nrlogrus.NewFormatter()
//
// The logger will now look for a newrelic.Transaction inside its context and
// decorate logs accordingly.  Therefore, the Transaction must be added to the
// context and passed to the logger.
//
//	ctx := newrelic.NewContext(context.Background(), txn)
//	logger.WithContext(ctx).Info("Hello New Relic!")
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

// NewFormatter creates a new `logrus.Formatter` that will format logs for
// sending to New Relic.
func NewFormatter() logrus.Formatter {
	return nrFormatter{
		jsonFormatter: logrus.JSONFormatter{
			DisableTimestamp: true,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyMsg:   logcontext.KeyMessage,
				logrus.FieldKeyLevel: logcontext.KeyLevel,
				logrus.FieldKeyFunc:  logcontext.KeyMethod,
				logrus.FieldKeyFile:  logcontext.KeyFile,
			},
		},
	}
}
