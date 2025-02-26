// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

const MB = 1024 * 1024

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("OOM Response High Water Mark App"),
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Wait for the application to connect.
	if err := app.WaitForConnection(5 * time.Second); err != nil {
		fmt.Println(err)
	}

	app.HeapHighWaterMarkAlarmSet(1*MB, megabyte)
	app.HeapHighWaterMarkAlarmSet(10*MB, tenMegabyte)
	app.HeapHighWaterMarkAlarmSet(100*MB, hundredMegabyte)
	app.HeapHighWaterMarkAlarmEnable(2 * time.Second)

	var a [][]byte
	for _ = range 100 {
		a = append(a, make([]byte, MB, MB))
		time.Sleep(1 * time.Second)
	}

	// Shut down the application to flush data to New Relic.
	app.Shutdown(10 * time.Second)
}

func megabyte(limit uint64, stats *runtime.MemStats) {
	fmt.Printf("*** 1M *** threshold %v alloc %v (%v)\n", limit, stats.Alloc, stats.TotalAlloc)
}
func tenMegabyte(limit uint64, stats *runtime.MemStats) {
	fmt.Printf("*** 10M *** threshold %v alloc %v (%v)\n", limit, stats.Alloc, stats.TotalAlloc)
}
func hundredMegabyte(limit uint64, stats *runtime.MemStats) {
	fmt.Printf("*** 100M *** threshold %v alloc %v (%v)\n", limit, stats.Alloc, stats.TotalAlloc)
}
