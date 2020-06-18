// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrlogrus

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func bufferToStringAndReset(buf *bytes.Buffer) string {
	s := buf.String()
	buf.Reset()
	return s
}

func TestLogrus(t *testing.T) {
	buf := &bytes.Buffer{}
	l := logrus.New()
	l.SetOutput(buf)
	l.SetLevel(logrus.DebugLevel)
	lg := Transform(l)
	lg.Debug("elephant", map[string]interface{}{"color": "gray"})
	s := bufferToStringAndReset(buf)
	if !strings.Contains(s, "elephant") || !strings.Contains(s, "gray") {
		t.Error(s)
	}
	if enabled := lg.DebugEnabled(); !enabled {
		t.Error(enabled)
	}
	// Now switch the level and test that debug is no longer enabled.
	l.SetLevel(logrus.InfoLevel)
	lg.Debug("lion", map[string]interface{}{"color": "yellow"})
	s = bufferToStringAndReset(buf)
	if strings.Contains(s, "lion") || strings.Contains(s, "yellow") {
		t.Error(s)
	}
	if enabled := lg.DebugEnabled(); enabled {
		t.Error(enabled)
	}
	lg.Info("tiger", map[string]interface{}{"color": "orange"})
	s = bufferToStringAndReset(buf)
	if !strings.Contains(s, "tiger") || !strings.Contains(s, "orange") {
		t.Error(s)
	}
}
