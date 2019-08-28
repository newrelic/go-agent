// Package nrlogrusplugin decorates logs for sending to the New Relic backend.
//
// Use this package if you want to enable the New Relic logging product and see
// your log messages in the New Relic UI.
//
// Since Logrus is completely api-compatible with the stdlib logger, you can
// replace your `"log"` imports with `log "github.com/sirupsen/logrus"` and
// follow the steps below to enable the logging product for use with the stdlib
// Go logger.
//
// To enable, set your log's formatter to the `nrlogrusplugin.NewFormatter()`
//
//	logger := logrus.New()
//	logger.SetFormatter(nrlogrusplugin.NewFormatter())
//
// The logger will now look for a newrelic.Transaction inside its context and
// decorate logs accordingly.  Therefore, the Transaction must be added to the
// context and passed to the logger.  For example, this logging call
//
//	logger.Info("Hello New Relic!")
//
// must be transformed to include the context, such as:
//
//	ctx := newrelic.NewContext(context.Background(), txn)
//	logger.WithContext(ctx).Info("Hello New Relic!")
//
// Using `logger.WithField`
// (https://godoc.org/github.com/sirupsen/logrus#Logger.WithField) and
// `logger.WithFields`
// (https://godoc.org/github.com/sirupsen/logrus#Logger.WithFields) is
// supported.  However, if the field key collides with one of the keys used by
// the New Relic Formatter, the value will be overwritten.  Reserved keys are
// those returned from `txn.GetLinkingMetadata().Map()`
// (https://godoc.org/github.com/newrelic/go-agent/#LinkingMetadata.Map) and
// those found in the `logcontext` package
// (https://godoc.org/github.com/newrelic/go-agent/_integrations/log-plugins/#pkg-constants).
//
// Supported types for `logger.WithField` and `logger.WithFields` field values
// are numbers, booleans, strings, and errors.  Func types are dropped and all
// other types are converted to strings.
//
// Requires v1.4.0 of the Logrus package or newer.
package nrlogrusplugin

import (
	"bytes"
	"fmt"

	newrelic "github.com/newrelic/go-agent"
	logcontext "github.com/newrelic/go-agent/_integrations/log-plugins"
	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/jsonx"
	"github.com/sirupsen/logrus"
)

func init() { internal.TrackUsage("integration", "log-context", "logrus") }

type logFields map[string]interface{}

type nrFormatter struct{}

func (f nrFormatter) Format(e *logrus.Entry) ([]byte, error) {
	data := make(logFields, len(e.Data)+12)
	for k, v := range e.Data {
		data[k] = v
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

	if e.HasCaller() {
		data[logcontext.KeyFile] = e.Caller.File
		data[logcontext.KeyLine] = e.Caller.Line
		data[logcontext.KeyMethod] = e.Caller.Function
	}

	var b *bytes.Buffer
	if e.Buffer != nil {
		b = e.Buffer
	} else {
		b = &bytes.Buffer{}
	}
	writeDataJSON(b, data)
	return b.Bytes(), nil
}

// NewFormatter creates a new `logrus.Formatter` that will format logs for
// sending to New Relic.
func NewFormatter() logrus.Formatter {
	return nrFormatter{}
}

func writeDataJSON(buf *bytes.Buffer, data logFields) {
	buf.WriteByte('{')
	var needsComma bool
	for k, v := range data {
		if needsComma {
			buf.WriteByte(',')
		} else {
			needsComma = true
		}
		jsonx.AppendString(buf, k)
		buf.WriteByte(':')
		writeValue(buf, v)
	}
	buf.WriteByte('}')
	buf.WriteByte('\n')
}

func writeValue(buf *bytes.Buffer, val interface{}) {
	switch v := val.(type) {
	case string:
		jsonx.AppendString(buf, v)
	case bool:
		if v {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case uint8:
		jsonx.AppendInt(buf, int64(v))
	case uint16:
		jsonx.AppendInt(buf, int64(v))
	case uint32:
		jsonx.AppendInt(buf, int64(v))
	case uint64:
		jsonx.AppendInt(buf, int64(v))
	case uint:
		jsonx.AppendInt(buf, int64(v))
	case uintptr:
		jsonx.AppendInt(buf, int64(v))
	case int8:
		jsonx.AppendInt(buf, int64(v))
	case int16:
		jsonx.AppendInt(buf, int64(v))
	case int32:
		jsonx.AppendInt(buf, int64(v))
	case int:
		jsonx.AppendInt(buf, int64(v))
	case int64:
		jsonx.AppendInt(buf, v)
	case float32:
		jsonx.AppendFloat(buf, float64(v))
	case float64:
		jsonx.AppendFloat(buf, v)
	case logrus.Level:
		jsonx.AppendString(buf, v.String())
	case error:
		jsonx.AppendString(buf, v.Error())
	default:
		jsonx.AppendString(buf, fmt.Sprintf("%#v", v))
	}
}
