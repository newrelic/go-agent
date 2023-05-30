// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrlogxi_test

import (
	"bytes"
	"strings"

	"testing"

	log "github.com/mgutz/logxi/v1"
	nrlogxi "github.com/newrelic/go-agent/v3/integrations/nrlogxi"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"

	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
)

func bufferToStringAndReset(buf *bytes.Buffer) string {
	s := buf.String()
	buf.Reset()
	return s
}

func createLoggerWithBuffer() (newrelic.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	l := log.NewLogger(buf, "LoggerName")
	l.SetLevel(log.LevelDebug)
	logger := nrlogxi.New(l)

	return logger, buf
}

func TestLogxiDebug(t *testing.T) {
	l, buf := createLoggerWithBuffer()
	l.Debug("elephant", map[string]interface{}{"color": "gray"})
	s := bufferToStringAndReset(buf)

	// check to see if the level is set to debug
	if !l.DebugEnabled() {
		t.Error("Debug mode not enabled")
	}

	if !strings.Contains(s, "DBG") {
		t.Error(s)
	}
	if !strings.Contains(s, "elephant") || !strings.Contains(s, "gray") {
		t.Error(s)
	}
}

func TestLogxiInfo(t *testing.T) {
	l, buf := createLoggerWithBuffer()
	l.Info("tiger", map[string]interface{}{"color": "orange"})
	s := bufferToStringAndReset(buf)

	// check to see if the level is set to info
	if !strings.Contains(s, "INF") {
		t.Error(s)
	}
	if !strings.Contains(s, "tiger") || !strings.Contains(s, "orange") {
		t.Error(s)
	}
}

func TestLogxiError(t *testing.T) {
	l, buf := createLoggerWithBuffer()
	l.Error("tiger", map[string]interface{}{"color": "orange"})
	s := bufferToStringAndReset(buf)

	// check to see if the level is set to error
	if !strings.Contains(s, "ERR") {
		t.Error(s)
	}
	if !strings.Contains(s, "tiger") || !strings.Contains(s, "orange") {
		t.Error(s)
	}
}

func TestLogxiWarn(t *testing.T) {
	l, buf := createLoggerWithBuffer()
	l.Warn("tiger", map[string]interface{}{"color": "orange"})
	s := bufferToStringAndReset(buf)

	// check to see if the level is set to warning
	if !strings.Contains(s, "WRN") {
		t.Error(s)
	}
	if !strings.Contains(s, "tiger") || !strings.Contains(s, "orange") {
		t.Error(s)
	}
}
func TestConfigLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	l := log.NewLogger(buf, "LoggerName")
	l.SetLevel(log.LevelDebug)

	integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		nrlogxi.ConfigLogger(l),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	s := bufferToStringAndReset(buf)

	if !strings.Contains(s, "application created") || !strings.Contains(s, "my app") {
		t.Error(s)
	}
}
