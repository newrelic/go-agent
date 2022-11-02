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

// Format renders a single log entry.
func (f ContextFormatter) Format(e *logrus.Entry) ([]byte, error) {
	logData := newrelic.LogData{
		Severity: e.Level.String(),
		Message:  e.Message,
	}

	logBytes, err := f.formatter.Format(e)
	if err != nil {
		return nil, err
	}
	logBytes = bytes.TrimRight(logBytes, "\n")
	b := bytes.NewBuffer(logBytes)

	ctx := e.Context
	var txn *newrelic.Transaction
	if ctx != nil {
		txn = newrelic.FromContext(ctx)
	}
	if txn != nil {
		txn.RecordLog(logData)
		err := newrelic.EnrichLog(b, newrelic.FromTxn(txn))
		if err != nil {
			return nil, err
		}
	} else {
		f.app.RecordLog(logData)
		err := newrelic.EnrichLog(b, newrelic.FromApp(f.app))
		if err != nil {
			return nil, err
		}
	}
	b.WriteString("\n")
	return b.Bytes(), nil
}
