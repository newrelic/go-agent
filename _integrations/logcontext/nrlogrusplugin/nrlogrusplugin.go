// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

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
// Using `logger.WithField`
// (https://godoc.org/github.com/sirupsen/logrus#Logger.WithField) and
// `logger.WithFields`
// (https://godoc.org/github.com/sirupsen/logrus#Logger.WithFields) is
// supported.  However, if the field key collides with one of the keys used by
// the New Relic Formatter, the value will be overwritten.  Reserved keys are
// those found in the `logcontext` package
// (https://godoc.org/github.com/newrelic/go-agent/_integrations/logcontext/#pkg-constants).
//
// Supported types for `logger.WithField` and `logger.WithFields` field values
// are numbers, booleans, strings, and errors.  Func types are dropped and all
// other types are converted to strings.
//
// Requires v1.4.0 of the Logrus package or newer.
//
// Configuration
//
// For the best linking experience be sure to enable Distributed Tracing:
//
//	cfg := NewConfig("Example Application", "__YOUR_NEW_RELIC_LICENSE_KEY__")
//	cfg.DistributedTracer.Enabled = true
//
// To enable log decoration, set your log's formatter to the
// `nrlogrusplugin.ContextFormatter`
//
//	logger := log.New()
//	logger.SetFormatter(nrlogrusplugin.ContextFormatter{})
//
// or if you are using the logrus standard logger
//
//	log.SetFormatter(nrlogrusplugin.ContextFormatter{})
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
// Troubleshooting
//
// When properly configured, your log statements will be in JSON format with
// one message per line:
//
//	{"message":"Hello New Relic!","log.level":"info","trace.id":"469a04f6c1278593","span.id":"9f365c71f0f04a98","entity.type":"SERVICE","entity.guid":"MTE3ODUwMHxBUE18QVBQTElDQVRJT058Mjc3MDU2Njc1","hostname":"my.hostname","timestamp":1568917432034,"entity.name":"Example Application"}
//
// If the `trace.id` key is missing, be sure that Distributed Tracing is
// enabled and that the Transaction context has been added to the logger using
// `WithContext` (https://godoc.org/github.com/sirupsen/logrus#Logger.WithContext).
package nrlogrusplugin

import (
	"bytes"
	"encoding/json"
	"fmt"

	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/logcontext"
	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/jsonx"
	"github.com/sirupsen/logrus"
)

func init() { internal.TrackUsage("integration", "logcontext", "logrus") }

type logFields map[string]interface{}

// ContextFormatter is a `logrus.Formatter` that will format logs for sending
// to New Relic.
type ContextFormatter struct{}

// Format renders a single log entry.
func (f ContextFormatter) Format(e *logrus.Entry) ([]byte, error) {
	// 12 = 6 from GetLinkingMetadata + 6 more below
	data := make(logFields, len(e.Data)+12)
	for k, v := range e.Data {
		data[k] = v
	}

	if ctx := e.Context; nil != ctx {
		if txn := newrelic.FromContext(ctx); nil != txn {
			logcontext.AddLinkingMetadata(data, txn.GetLinkingMetadata())
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
		if m, ok := v.(json.Marshaler); ok {
			if js, err := m.MarshalJSON(); nil == err {
				buf.Write(js)
				return
			}
		}
		jsonx.AppendString(buf, fmt.Sprintf("%#v", v))
	}
}
