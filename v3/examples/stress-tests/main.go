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

var (
	applicationLoggingEvents = "Application Logging Events"
	customEvents             = "Custom Events"
)

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("ApplicationLogging Stress Test Golang"),
		newrelic.ConfigAppLogForwardingEnabled(true),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigInfoLogger(os.Stdout),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Wait for the application to connect.
	if err := app.WaitForConnection(5 * time.Second); nil != err {
		fmt.Println(err)
	}

	tests := []Benchmark{
		NewLogBenchmark(10, 6),
		NewLogBenchmark(100, 6),
		NewLogBenchmark(1000, 6),

		NewCustomEventBenchmark(10, 6),
		NewCustomEventBenchmark(100, 6),
		NewCustomEventBenchmark(1000, 6),
	}

	for _, test := range tests {
		test.Benchmark(app)
	}

	var metrics string
	for _, test := range tests {
		metrics += test.Sprint()
	}

	// Wait for the application to connect.
	if err := app.WaitForConnection(5 * time.Second); nil != err {
		fmt.Println(err)
	}

	// Shut down the application to flush data to New Relic.
	app.Shutdown(10 * time.Second)

	fmt.Println(metrics)
}

type Benchmark struct {
	eventType string
	numEvents int
	sets      int
	runTimes  []int64
}

func NewLogBenchmark(numEvents, numRuns int) Benchmark {
	return Benchmark{
		applicationLoggingEvents,
		numEvents,
		numRuns,
		make([]int64, numRuns),
	}
}

func NewCustomEventBenchmark(numEvents, numRuns int) Benchmark {
	return Benchmark{
		customEvents,
		numEvents,
		numRuns,
		make([]int64, numRuns),
	}
}

func (bench *Benchmark) Sprint() string {
	sum := int64(0)
	output := fmt.Sprintf("Time taken to record %d %s:\n", bench.numEvents, bench.eventType)
	for _, time := range bench.runTimes {
		output += fmt.Sprintf("\t\tMicroseconds: %d\n", time)
		sum += time
	}

	average := sum / int64(len(bench.runTimes))
	output += fmt.Sprintf("\t\tAverage Microseconds: %d\n", average)
	return output
}

func (bench *Benchmark) Benchmark(app *newrelic.Application) {
	for set := 0; set < bench.sets; set++ {
		start := time.Now()
		for i := 0; i < bench.numEvents; i++ {
			switch bench.eventType {
			case applicationLoggingEvents:
				message := "Message " + fmt.Sprint(i)
				app.RecordLogEvent(context.Background(), message, "INFO", time.Now().UnixMilli())
			case customEvents:
				message := "Message " + fmt.Sprint(i)
				app.RecordCustomEvent("TEST EVENT", map[string]interface{}{
					"Message": message,
				})
			}
		}
		bench.runTimes[set] = time.Since(start).Microseconds()
	}
}
