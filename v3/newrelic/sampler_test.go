// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/logger"
)

func TestGetSample(t *testing.T) {
	now := time.Now()
	sample := getSystemSample(now, logger.ShimLogger{})
	if nil == sample {
		t.Fatal(sample)
	}
	if now != sample.when {
		t.Error(now, sample.when)
	}
	if sample.numGoroutine <= 0 {
		t.Error(sample.numGoroutine)
	}
	if sample.numCPU <= 0 {
		t.Error(sample.numCPU)
	}
	if sample.memStats.HeapObjects == 0 {
		t.Error(sample.memStats.HeapObjects)
	}
}

func TestMetricsCreated(t *testing.T) {
	now := time.Now()
	h := newHarvest(now, testHarvestCfgr)

	stats := systemStats{
		heapObjects:  5 * 1000,
		numGoroutine: 23,
		allocBytes:   37 * 1024 * 1024,
		user: cpuStats{
			used:     20 * time.Millisecond,
			fraction: 0.01,
		},
		system: cpuStats{
			used:     40 * time.Millisecond,
			fraction: 0.02,
		},
		gcPauseFraction: 3e-05,
		deltaNumGC:      2,
		deltaPauseTotal: 500 * time.Microsecond,
		minPause:        100 * time.Microsecond,
		maxPause:        400 * time.Microsecond,
	}

	stats.MergeIntoHarvest(h)

	expectMetrics(t, h.Metrics, []internal.WantMetric{
		{Name: "Memory/Heap/AllocatedObjects", Scope: "", Forced: true, Data: []float64{1, 5000, 5000, 5000, 5000, 25000000}},
		{Name: "Memory/Physical", Scope: "", Forced: true, Data: []float64{1, 37, 0, 37, 37, 1369}},
		{Name: "CPU/User Time", Scope: "", Forced: true, Data: []float64{1, 0.02, 0.02, 0.02, 0.02, 0.0004}},
		{Name: "CPU/System Time", Scope: "", Forced: true, Data: []float64{1, 0.04, 0.04, 0.04, 0.04, 0.0016}},
		{Name: "CPU/User/Utilization", Scope: "", Forced: true, Data: []float64{1, 0.01, 0, 0.01, 0.01, 0.0001}},
		{Name: "CPU/System/Utilization", Scope: "", Forced: true, Data: []float64{1, 0.02, 0, 0.02, 0.02, 0.0004}},
		{Name: "Go/Runtime/Goroutines", Scope: "", Forced: true, Data: []float64{1, 23, 23, 23, 23, 529}},
		{Name: "GC/System/Pause Fraction", Scope: "", Forced: true, Data: []float64{1, 3e-05, 0, 3e-05, 3e-05, 9e-10}},
		{Name: "GC/System/Pauses", Scope: "", Forced: true, Data: []float64{2, 0.0005, 0, 0.0001, 0.0004, 2.5e-7}},
	})
}

func TestMetricsCreatedEmpty(t *testing.T) {
	now := time.Now()
	h := newHarvest(now, testHarvestCfgr)
	stats := systemStats{}

	stats.MergeIntoHarvest(h)

	expectMetrics(t, h.Metrics, []internal.WantMetric{
		{Name: "Memory/Heap/AllocatedObjects", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Memory/Physical", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "CPU/User Time", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "CPU/System Time", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "CPU/User/Utilization", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "CPU/System/Utilization", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Go/Runtime/Goroutines", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "GC/System/Pause Fraction", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})
}
