package internal

import (
	"testing"
	"time"
)

func TestMetricsCreated(t *testing.T) {
	now := time.Now()
	h := NewHarvest(now)

	stats := Stats{
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

	ExpectMetrics(t, h.Metrics, []WantMetric{
		{"Memory/Physical", "", true, []float64{1, 37, 0, 37, 37, 1369}},
		{"CPU/User Time", "", true, []float64{1, 0.02, 0.02, 0.02, 0.02, 0.0004}},
		{"CPU/System Time", "", true, []float64{1, 0.04, 0.04, 0.04, 0.04, 0.0016}},
		{"CPU/User/Utilization", "", true, []float64{1, 0.01, 0, 0.01, 0.01, 0.0001}},
		{"CPU/System/Utilization", "", true, []float64{1, 0.02, 0, 0.02, 0.02, 0.0004}},
		{"Go/Runtime/Goroutines", "", true, []float64{1, 23, 23, 23, 23, 529}},
		{"GC/System/Pause Fraction", "", true, []float64{1, 3e-05, 0, 3e-05, 3e-05, 9e-10}},
		{"GC/System/Pauses", "", true, []float64{2, 0.0005, 0, 0.0001, 0.0004, 2.5e-7}},
	})
}

func TestMetricsCreatedEmpty(t *testing.T) {
	now := time.Now()
	h := NewHarvest(now)
	stats := Stats{}

	stats.MergeIntoHarvest(h)

	ExpectMetrics(t, h.Metrics, []WantMetric{
		{"Memory/Physical", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"CPU/User Time", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"CPU/System Time", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"CPU/User/Utilization", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"CPU/System/Utilization", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Go/Runtime/Goroutines", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"GC/System/Pause Fraction", "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
}
