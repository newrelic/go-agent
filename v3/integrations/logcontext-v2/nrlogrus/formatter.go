// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrlogrus

import (
	"bytes"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sirupsen/logrus"
)

//func init() { internal.TrackUsage("integration", "logcontext-v2", "logrus") }

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

	logData := newrelic.LogData{
		Severity: e.Level.String(),
		Message:  e.Message,
	}

	ctx := e.Context
	var txn *newrelic.Transaction
	if ctx != nil {
		txn = newrelic.FromContext(ctx)
	}

	/*
		if e.HasCaller() {
			data[logcontext.KeyFile] = e.Caller.File
			data[logcontext.KeyLine] = e.Caller.Line
			data[logcontext.KeyMethod] = e.Caller.Function
		}*/

	var b *bytes.Buffer
	if e.Buffer != nil {
		b = e.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	err := logData.AppendLog(b, txn)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
