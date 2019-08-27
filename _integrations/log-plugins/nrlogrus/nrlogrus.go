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
	"encoding/json"

	newrelic "github.com/newrelic/go-agent"
	logcontext "github.com/newrelic/go-agent/_integrations/log-plugins"
	"github.com/newrelic/go-agent/internal"
	"github.com/sirupsen/logrus"
)

func init() { internal.TrackUsage("integration", "log-context", "logrus") }

type logFields map[string]interface{}

type nrFormatter struct{}

func (f nrFormatter) Format(e *logrus.Entry) ([]byte, error) {
	data := make(logFields, len(e.Data)+12) // TODO: how much to add?
	for k, v := range e.Data {
		switch v := v.(type) {
		case error:
			// TODO: test this
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/sirupsen/logrus/issues/137
			data[k] = v.Error()
		default:
			data[k] = v
		}
	}

	if ctx := e.Context; nil != ctx {
		if txn := newrelic.FromContext(ctx); nil != txn {
			for k, v := range txn.GetLinkingMetadata().Map() {
				data[k] = v
			}
		}
	}

	data[logcontext.KeyTimestamp] = uint64(e.Time.UnixNano()) / uint64(1000*1000)
	data[logcontext.KeyMessage] = e.Message
	data[logcontext.KeyLevel] = e.Level
	// TODO: cannot record e.err since it is private, document this
	// TODO: test for when key names collide, document this

	// TODO: test when the caller is disabled
	if e.HasCaller() {
		data[logcontext.KeyFile] = e.Caller.File
		data[logcontext.KeyLine] = e.Caller.Line
		data[logcontext.KeyMethod] = e.Caller.Function
	}

	return json.Marshal(data)
}

// NewFormatter creates a new `logrus.Formatter` that will format logs for
// sending to New Relic.
func NewFormatter() logrus.Formatter {
	return nrFormatter{}
}
