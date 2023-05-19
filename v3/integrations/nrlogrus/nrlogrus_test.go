// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrlogrus

import (
	"bytes"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sirupsen/logrus"
	"strings"
	"testing"

	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
)

func bufferToStringAndReset(buf *bytes.Buffer) string {
	s := buf.String()
	buf.Reset()
	return s
}

func createLoggerWithBuffer(level logrus.Level) (*logrus.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	l := logrus.New()
	l.SetOutput(buf)
	l.SetLevel(logrus.DebugLevel)
	return l, buf
}

func TestLogrusDebug(t *testing.T) {
	l, buf := createLoggerWithBuffer(logrus.DebugLevel)
	lg := Transform(l)
	lg.Debug("elephant", map[string]interface{}{"color": "gray"})
	s := bufferToStringAndReset(buf)
	// check to see if the level is set to debug
	if !strings.Contains(s, "level=debug") {
		t.Error(s)
	}
	if !strings.Contains(s, "elephant") || !strings.Contains(s, "gray") {
		t.Error(s)
	}
	if enabled := lg.DebugEnabled(); !enabled {
		t.Error(enabled)
	}

}
func TestLogrusInfo(t *testing.T) {
	l, buf := createLoggerWithBuffer(logrus.InfoLevel)
	lg := Transform(l)
	lg.Info("tiger", map[string]interface{}{"color": "orange"})
	s := bufferToStringAndReset(buf)
	// check to see if the level is set to info
	if !strings.Contains(s, "level=info") {
		t.Error(s)
	}

	if !strings.Contains(s, "tiger") || !strings.Contains(s, "orange") {
		t.Error(s)
	}
}

func TestLogrusError(t *testing.T) {
	l, buf := createLoggerWithBuffer(logrus.ErrorLevel)
	lg := Transform(l)
	lg.Error("tiger", map[string]interface{}{"color": "orange"})
	s := bufferToStringAndReset(buf)
	// check to see if the level is set to error
	if !strings.Contains(s, "level=error") {
		t.Error(s)
	}
	if !strings.Contains(s, "tiger") || !strings.Contains(s, "orange") {
		t.Error(s)
	}
}

func TestLogrusWarn(t *testing.T) {
	l, buf := createLoggerWithBuffer(logrus.WarnLevel)
	lg := Transform(l)
	lg.Warn("tiger", map[string]interface{}{"color": "orange"})
	s := bufferToStringAndReset(buf)
	// check to see if the level is set to warning
	if !strings.Contains(s, "level=warn") {
		t.Error(s)
	}
	if !strings.Contains(s, "tiger") || !strings.Contains(s, "orange") {
		t.Error(s)
	}
}

func TestConfigLogger(t *testing.T) {
	l, buf := createLoggerWithBuffer(logrus.InfoLevel)

	integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		ConfigLogger(l),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	s := bufferToStringAndReset(buf)

	if !strings.Contains(s, "application created") || !strings.Contains(s, "my app") {
		t.Error(s)
	}
}

func TestConfigStandardLogger(t *testing.T) {
	buf := &bytes.Buffer{}

	integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		ConfigStandardLogger(),
		newrelic.ConfigDebugLogger(buf),
	)

	s := bufferToStringAndReset(buf)

	if !strings.Contains(s, "application created") || !strings.Contains(s, "my app") {
		t.Error(s)
	}

}
