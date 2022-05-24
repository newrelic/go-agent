// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrzerolog"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

var (
	TestNRZL         = "Zerolog Plugin"
	TestZerolog      = "Zerolog"
	TestCustomEvents = "Custom Events"
)

// Zerolog Test
func Zerolog(numEvents, numRuns int) Benchmark {
	return Benchmark{
		TestZerolog,
		numEvents,
		numRuns,
		make([]int64, numRuns),
	}
}

func (bench *Benchmark) timeZerologSet() int64 {
	// Init logger
	logger := zerolog.New(nil)

	// Time Consumption
	start := time.Now()
	for i := 0; i < bench.numEvents; i++ {
		logger.Info().Msg("Message " + fmt.Sprint(i))
	}
	return time.Since(start).Microseconds()
}

// NR Zerolog Plugin Test
func NRZerolog(numEvents, numRuns int) Benchmark {
	return Benchmark{
		TestNRZL,
		numEvents,
		numRuns,
		make([]int64, numRuns),
	}
}

func (bench *Benchmark) timeZerologPluginSet(app *newrelic.Application) int64 {
	// Init Logger

	nrHook := nrzerolog.Hook{
		App: app,
	}

	logger := zerolog.New(nil).Hook(nrHook)

	// Time Consumption
	start := time.Now()
	for i := 0; i < bench.numEvents; i++ {
		logger.Info().Msg("Message " + fmt.Sprint(i))
	}
	return time.Since(start).Microseconds()
}

// Custom Events Test
func CustomEvent(numEvents, numRuns int) Benchmark {
	return Benchmark{
		TestCustomEvents,
		numEvents,
		numRuns,
		make([]int64, numRuns),
	}
}

func (bench *Benchmark) timeCustomEventSet(app *newrelic.Application) int64 {
	// Time Consumption
	start := time.Now()
	for i := 0; i < bench.numEvents; i++ {
		message := "Message " + fmt.Sprint(i)
		app.RecordCustomEvent("TEST EVENT", map[string]interface{}{
			"Message": message,
		})
	}
	return time.Since(start).Microseconds()
}

// Benchmark Framework
type Benchmark struct {
	eventType string
	numEvents int
	sets      int
	runTimes  []int64
}

func (bench *Benchmark) Sprint() string {
	output := fmt.Sprintf("Time taken to record %d %s:\n", bench.numEvents, bench.eventType)
	for _, time := range bench.runTimes {
		output += fmt.Sprintf("\t\tMicroseconds: %d\n", time)
	}

	validTimes, sum := normalize(bench.runTimes)
	average := float64(sum) / float64(len(validTimes))
	output += fmt.Sprintf("\t\tAverage Microseconds: %.3f\n", average)
	return output
}

func (bench *Benchmark) Benchmark(app *newrelic.Application) {
	for set := 0; set < bench.sets; set++ {
		switch bench.eventType {
		case TestZerolog:
			bench.runTimes[set] = bench.timeZerologSet()
		case TestNRZL:
			bench.runTimes[set] = bench.timeZerologPluginSet(app)
		case TestCustomEvents:
			bench.runTimes[set] = bench.timeCustomEventSet(app)
		}
	}
}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("ApplicationLogging Stress Test Golang"),
		newrelic.ConfigZerologPluginEnabled(true),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigInfoLogger(os.Stdout),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	tests := []Benchmark{
		Zerolog(10, 10),
		Zerolog(100, 10),
		Zerolog(1000, 10),
		Zerolog(10000, 10),

		NRZerolog(10, 10),
		NRZerolog(100, 10),
		NRZerolog(1000, 10),
		NRZerolog(1000, 10),

		CustomEvent(10, 10),
		CustomEvent(100, 10),
		CustomEvent(1000, 10),
		CustomEvent(10000, 10),
	}

	// Wait for the application to connect.
	if err := app.WaitForConnection(5 * time.Second); nil != err {
		fmt.Println(err)
	}

	for _, test := range tests {
		test.Benchmark(app)
	}

	// Make sure the metrics get sent
	time.Sleep(60 * time.Second)
	app.Shutdown(10 * time.Second)

	// Compile metrics data as pretty printed strings
	var metrics string
	for _, test := range tests {
		metrics += test.Sprint()
	}

	fmt.Println(metrics)
}
