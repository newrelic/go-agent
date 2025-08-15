// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"runtime"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/logger"
	"github.com/newrelic/go-agent/v3/internal/sysinfo"
)

func TestGetSystemSample(t *testing.T) {
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

func TestGetSystemStats(t *testing.T) {
	baseTime := time.Now()

	// Test with meaningful differences between samples
	prev := &systemSample{
		when:         baseTime,
		numGoroutine: 10,
		numCPU:       4,
		memStats: runtime.MemStats{
			Alloc:        50 * 1024 * 1024, // 50MB
			HeapObjects:  1000,
			PauseTotalNs: 1000000, // 1ms
			NumGC:        5,
			PauseNs:      [256]uint64{}, // Will be set below
		},
		usage: sysinfo.Usage{
			User:   100 * time.Millisecond,
			System: 50 * time.Millisecond,
		},
	}

	current := &systemSample{
		when:         baseTime.Add(1 * time.Second),
		numGoroutine: 15,
		numCPU:       4,
		memStats: runtime.MemStats{
			Alloc:        75 * 1024 * 1024, // 75MB
			HeapObjects:  1500,
			PauseTotalNs: 2000000,       // 2ms total
			NumGC:        7,             // 2 more GCs
			PauseNs:      [256]uint64{}, // Will be set below
		},
		usage: sysinfo.Usage{
			User:   150 * time.Millisecond,
			System: 80 * time.Millisecond,
		},
	}

	// Set pause data at correct indices for NumGC transitions from 5 to 7
	// The new GCs (6 and 7) should have their pause data at indices (6-1)%256=5 and (7-1)%256=6
	current.memStats.PauseNs[5] = 400000 // 400µs for 6th GC
	current.memStats.PauseNs[6] = 500000 // 500µs for 7th GC

	stats := getSystemStats(systemSamples{Previous: prev, Current: current})

	// Verify basic metrics
	if stats.numGoroutine != 15 {
		t.Errorf("Expected numGoroutine=15, got %d", stats.numGoroutine)
	}
	if stats.allocBytes != 75*1024*1024 {
		t.Errorf("Expected allocBytes=78643200, got %d", stats.allocBytes)
	}
	if stats.heapObjects != 1500 {
		t.Errorf("Expected heapObjects=1500, got %d", stats.heapObjects)
	}

	// Verify CPU utilization (50ms user over 1s with 4 CPUs = 50ms / 4000ms = 0.0125)
	expectedUserFraction := 0.0125
	if abs(stats.user.fraction-expectedUserFraction) > 0.001 {
		t.Errorf("Expected user.fraction=%f, got %f", expectedUserFraction, stats.user.fraction)
	}
	if stats.user.used != 50*time.Millisecond {
		t.Errorf("Expected user.used=50ms, got %v", stats.user.used)
	}

	// Verify system CPU (30ms system over 1s with 4 CPUs = 30ms / 4000ms = 0.0075)
	expectedSystemFraction := 0.0075
	if abs(stats.system.fraction-expectedSystemFraction) > 0.001 {
		t.Errorf("Expected system.fraction=%f, got %f", expectedSystemFraction, stats.system.fraction)
	}
	if stats.system.used != 30*time.Millisecond {
		t.Errorf("Expected system.used=30ms, got %v", stats.system.used)
	}

	// Verify GC pause fraction (1ms pause over 1s = 0.001)
	expectedGCFraction := 0.001
	if abs(stats.gcPauseFraction-expectedGCFraction) > 0.0001 {
		t.Errorf("Expected gcPauseFraction=%f, got %f", expectedGCFraction, stats.gcPauseFraction)
	}

	// Verify GC pause stats
	if stats.deltaNumGC != 2 {
		t.Errorf("Expected deltaNumGC=2, got %d", stats.deltaNumGC)
	}
	if stats.deltaPauseTotal != 1*time.Millisecond {
		t.Errorf("Expected deltaPauseTotal=1ms, got %v", stats.deltaPauseTotal)
	}
	if stats.minPause != 400*time.Microsecond {
		t.Errorf("Expected minPause=400µs, got %v", stats.minPause)
	}
	if stats.maxPause != 500*time.Microsecond {
		t.Errorf("Expected maxPause=500µs, got %v", stats.maxPause)
	}

	// Test case with pause greater than average to cover maxPauseNs branch
	prevMaxTest := &systemSample{
		when: baseTime,
		memStats: runtime.MemStats{
			NumGC:        10,
			PauseTotalNs: 300000,        // 300µs total
			PauseNs:      [256]uint64{}, // Will be set below
		},
		usage: sysinfo.Usage{},
	}

	currentMaxTest := &systemSample{
		when: baseTime.Add(1 * time.Second),
		memStats: runtime.MemStats{
			NumGC:        12,            // 2 new GCs
			PauseTotalNs: 900000,        // 900µs total (600µs delta)
			PauseNs:      [256]uint64{}, // Will be set below
		},
		usage: sysinfo.Usage{},
	}

	// Set pause data at correct indices for NumGC transitions from 10 to 12
	// The new GCs (11 and 12) should have their pause data at indices (11-1)%256=10 and (12-1)%256=11
	currentMaxTest.memStats.PauseNs[10] = 100000 // 100µs for 11th GC
	currentMaxTest.memStats.PauseNs[11] = 500000 // 500µs for 12th GC

	statsMaxTest := getSystemStats(systemSamples{Previous: prevMaxTest, Current: currentMaxTest})

	// Average pause would be 600µs / 2 = 300µs
	// The 500µs pause should trigger the maxPauseNs > pause condition
	if statsMaxTest.maxPause != 500*time.Microsecond {
		t.Errorf("Expected maxPause=500µs for max test, got %v", statsMaxTest.maxPause)
	}
	if statsMaxTest.minPause != 100*time.Microsecond {
		t.Errorf("Expected minPause=100µs for max test, got %v", statsMaxTest.minPause)
	}

}

func TestGetSystemStatsNoGC(t *testing.T) {
	baseTime := time.Now()

	prev := &systemSample{
		when:     baseTime,
		memStats: runtime.MemStats{NumGC: 5},
		usage:    sysinfo.Usage{},
	}

	current := &systemSample{
		when:     baseTime.Add(1 * time.Second),
		memStats: runtime.MemStats{NumGC: 5}, // No new GCs
		usage:    sysinfo.Usage{},
	}

	stats := getSystemStats(systemSamples{Previous: prev, Current: current})

	// When no GC occurred, these should be zero
	if stats.deltaNumGC != 0 {
		t.Errorf("Expected deltaNumGC=0, got %d", stats.deltaNumGC)
	}
	if stats.deltaPauseTotal != 0 {
		t.Errorf("Expected deltaPauseTotal=0, got %v", stats.deltaPauseTotal)
	}
}

func TestGetSystemStatsNoCPUUsage(t *testing.T) {
	baseTime := time.Now()

	// For the initial sample (or if prior usage data is unavailable or zero) the calculated CPU utilization will be zero
	// to ensure accuracy and prevent misleading spikes.
	prev := &systemSample{
		when:  baseTime,
		usage: sysinfo.Usage{User: 0, System: 0},
	}

	current := &systemSample{
		when:  baseTime.Add(1 * time.Second),
		usage: sysinfo.Usage{User: 100 * time.Millisecond, System: 50 * time.Millisecond},
	}

	stats := getSystemStats(systemSamples{Previous: prev, Current: current})

	// CPU stats should be zero when previous usage is 0
	if stats.user.used != 0 {
		t.Errorf("Expected user.used=0, got %v", stats.user.used)
	}
	if stats.system.used != 0 {
		t.Errorf("Expected system.used=0, got %v", stats.system.used)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
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
