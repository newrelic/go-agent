// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Example Short Lived Process"),
		newrelic.ConfigAppLogForwardingEnabled(true),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigAppLogMetricsEnabled(true),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	// Wait for the application to connect.
	if err := app.WaitForConnection(5 * time.Second); nil != err {
		fmt.Println(err)
	}

	app.RecordLogEvent(context.Background(), "App Started", "INFO", time.Now().UnixMilli())

	// Do the tasks at hand.  Perhaps record them using transactions and/or
	// custom events.
	tasks := []string{"white", "black", "red", "blue", "green", "yellow"}
	for _, task := range tasks {
		txn := app.StartTransaction("task")
		time.Sleep(10 * time.Millisecond)
		txn.End()
		app.RecordCustomEvent("task", map[string]interface{}{
			"color": task,
		})
	}

	app.RecordLogEvent(context.Background(), "A warning log occured!", "WARN", time.Now().UnixMilli())
	app.RecordLogEvent(context.Background(), "App Executed Succesfully", "INFO", time.Now().UnixMilli())

	time.Sleep(60 * time.Second)

	// Shut down the application to flush data to New Relic.
	app.Shutdown(10 * time.Second)
}
