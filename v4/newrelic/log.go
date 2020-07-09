// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

// Logger is the interface that is used for logging in the Go Agent.  Assign
// the Config.Logger field to the Logger you wish to use.  Loggers must be safe
// for use in multiple goroutines.  Two Logger implementations are included:
// NewLogger, which logs at info level, and NewDebugLogger which logs at debug
// level.  logrus, logxi, and zap are supported by the integration packages
// https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrlogrus,
// https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrlogxi,
// and https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrzap
// respectively.
type Logger interface {
	Error(msg string, context map[string]interface{})
	Warn(msg string, context map[string]interface{})
	Info(msg string, context map[string]interface{})
	Debug(msg string, context map[string]interface{})
	DebugEnabled() bool
}
