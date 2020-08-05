// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"testing"
	"time"
)

func TestNilApplication(t *testing.T) {
	// Ensure using a nil application does not panic
	var app *Application
	app.StartTransaction("txn")
	app.RecordCustomEvent("Event", nil)
	app.RecordCustomMetric("Metric", 42)
	app.WaitForConnection(time.Nanosecond)
	app.Shutdown(time.Nanosecond)
}

func TestEmptyApplication(t *testing.T) {
	// Ensure using an empty application does not panic
	app := new(Application)
	app.StartTransaction("txn")
	app.RecordCustomEvent("Event", nil)
	app.RecordCustomMetric("Metric", 42)
	app.WaitForConnection(time.Nanosecond)
	app.Shutdown(time.Nanosecond)
}
