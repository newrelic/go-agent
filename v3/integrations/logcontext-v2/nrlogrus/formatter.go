// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrlogrus

import (
	"bytes"

	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sirupsen/logrus"
)

func init() { internal.TrackUsage("integration", "logcontext-v2", "logrus") }

// ContextFormatter is a `logrus.Formatter` that will format logs for sending
// to New Relic.
type ContextFormatter struct {
	app       *newrelic.Application
	formatter logrus.Formatter
}

func NewFormatter(app *newrelic.Application, formatter logrus.Formatter) ContextFormatter {
	return ContextFormatter{
		app:       app,
		formatter: formatter,
	}
}

// recordLog records the log data to the transaction or application
func (f ContextFormatter) recordLog(logData newrelic.LogData, txn *newrelic.Transaction) {
	if txn != nil {
		txn.RecordLog(logData)
	} else {
		f.app.RecordLog(logData)
	}
}

// enrichLog enriches the buffer with linking metadata
func (f ContextFormatter) enrichLog(buf *bytes.Buffer, txn *newrelic.Transaction) error {
	if txn != nil {
		return newrelic.EnrichLog(buf, newrelic.FromTxn(txn))
	}
	return newrelic.EnrichLog(buf, newrelic.FromApp(f.app))
}

// Format renders a single log entry.
func (f ContextFormatter) Format(e *logrus.Entry) ([]byte, error) {
	logData := newrelic.LogData{
		Severity:   e.Level.String(),
		Message:    e.Message,
		Attributes: e.Data,
	}

	ctx := e.Context
	var txn *newrelic.Transaction
	if ctx != nil {
		txn = newrelic.FromContext(ctx)
	}

	f.recordLog(logData, txn)

	cfg, _ := f.app.Config()

	if cfg.ApplicationLogging.LocalDecorating.WithinMessageField {
		msgBuf := bytes.NewBufferString(e.Message)
		if err := f.enrichLog(msgBuf, txn); err != nil {
			return nil, err
		}
		e.Message = msgBuf.String()
	}

	logBytes, err := f.formatter.Format(e)
	if err != nil {
		return nil, err
	}

	b := bytes.NewBuffer(bytes.TrimRight(logBytes, "\n"))
	if !cfg.ApplicationLogging.LocalDecorating.WithinMessageField {
		if err := f.enrichLog(b, txn); err != nil {
			return nil, err
		}
	}

	b.WriteString("\n")
	return b.Bytes(), nil
}
