// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"io"

	"github.com/newrelic/go-agent/v4/internal/logger"
)

// Logger is the interface that is used for logging in the Go Agent.  Assign
// the Config.Logger field to the Logger you wish to use.  Loggers must be safe
// for use in multiple goroutines.  logrus, logxi, and zap are supported by the
// integration packages
// https://godoc.org/github.com/newrelic/go-agent/v4/integrations/nrlogrus,
// https://godoc.org/github.com/newrelic/go-agent/v4/integrations/nrlogxi,
// and https://godoc.org/github.com/newrelic/go-agent/v4/integrations/nrzap
// respectively.
type Logger interface {
	Error(msg string, context map[string]interface{})
	Warn(msg string, context map[string]interface{})
	Info(msg string, context map[string]interface{})
	Debug(msg string, context map[string]interface{})
	DebugEnabled() bool
}

// newLogger creates a basic Logger at info level.
func newLogger(w io.Writer) Logger {
	return logger.New(w, false)
}

// newDebugLogger creates a basic Logger at debug level.
func newDebugLogger(w io.Writer) Logger {
	return logger.New(w, true)
}
