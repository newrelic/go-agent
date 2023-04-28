// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrzap_test

import (
	"testing"

	"github.com/newrelic/go-agent/v3/integrations/nrzap"
	"github.com/newrelic/go-agent/v3/newrelic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func Example() {
	// Create a new zap logger:
	z, _ := zap.NewProduction()

	newrelic.NewApplication(
		newrelic.ConfigAppName("Example App"),
		newrelic.ConfigLicense("__YOUR_NEWRELIC_LICENSE_KEY__"),
		// Use nrzap to register the logger with the agent:
		nrzap.ConfigLogger(z.Named("newrelic")),
	)
}

func TestLogs(t *testing.T) {
	tests := []struct {
		name    string
		logFunc func(logger newrelic.Logger, message string, attrs map[string]interface{})
		level   zapcore.LevelEnabler
	}{
		{
			name:    "Error",
			logFunc: newrelic.Logger.Error,
			level:   zap.ErrorLevel,
		},
		{
			name:    "Warn",
			logFunc: newrelic.Logger.Warn,
			level:   zap.WarnLevel,
		},
		{
			name:    "Info",
			logFunc: newrelic.Logger.Info,
			level:   zap.InfoLevel,
		},
		{
			name:    "Debug",
			logFunc: newrelic.Logger.Debug,
			level:   zap.DebugLevel,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create an observer to record logs at the specified level:
			observedZapCore, observedLogs := observer.New(test.level)
			observedLogger := zap.New(observedZapCore)

			// Create a test logger using nrzap.Transform:
			testLogger := nrzap.Transform(observedLogger)

			// Define a message and attributes for the test log message:
			message := test.name
			attrs := map[string]interface{}{
				"key": "val",
			}

			// Log the message and attributes using the test logger:
			test.logFunc(testLogger, message, attrs)

			// Check if observed log matches the expected message and attributes:
			logs := observedLogs.All()
			if len(logs) == 0 {
				t.Errorf("no log messages produced")
			} else {
				log := logs[0]
				if message != log.Message {
					t.Errorf("incorrect log message; expected: %s, got: %s", message, log.Message)
				}
				context := log.ContextMap()
				val, ok := context["key"]
				if !ok || val != "val" {
					t.Errorf("incorrect log attributes for key, \"key\"; expected \"val\", got: %s", val.(string))
				}
			}
		})
	}
}

func TestDebugEnabled(t *testing.T) {
	observedZapCore, _ := observer.New(zap.DebugLevel)
	observedLogger := zap.New(observedZapCore)

	testLogger := nrzap.Transform(observedLogger)

	if !testLogger.DebugEnabled() {
		t.Errorf("debug logging is not enabled")
	}
}
