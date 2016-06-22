package internal

import (
	"testing"
	"time"
)

func TestMetricsCreated(t *testing.T) {
	now := time.Now()
	h := newHarvest(now)

	stats := stats{
		numGoroutine:         23,
		allocMIB:             3740000,
		cpuUserUtilization:   3.82e-04,
		cpuSystemUtilization: 6.99e-05,
		gcPauseFraction:      3.03e-05,
		deltaNumGC:           2,
		deltaPauseTotal:      time.Duration(606000),
		minPause:             time.Duration(264000),
		maxPause:             time.Duration(342000),
	}

	stats.mergeIntoHarvest(h)

	expectMetrics(t, h.metrics, []WantMetric{
		{"Memory/Physical", "", true, []float64{1, 3.566741943359375, 0, 3.566741943359375, 3.566741943359375, 12.721648090519011}},
		{"CPU/User/Utilization", "", true, []float64{1, 0.000382, 0, 0.000382, 0.000382, 1.45924e-07}},
		{"CPU/System/Utilization", "", true, []float64{1, 6.99e-05, 0, 6.99e-05, 6.99e-05, 4.8860100000000006e-09}},
		{"Go/Runtime/Goroutines", "", true, []float64{1, 23, 23, 23, 23, 529}},
		{"GC/System/Pause Fraction", "", true, []float64{1, 3.03e-05, 0, 3.03e-05, 3.03e-05, 9.180900000000001e-10}},
		{"GC/System/Pauses", "", true, []float64{2, 0.000606, 0, 0.000264, 0.000342, 3.67236e-07}},
	})
}

func TestMetricsCreatedEmpty(t *testing.T) {
	now := time.Now()
	h := newHarvest(now)
	stats := stats{}

	stats.mergeIntoHarvest(h)

	expectMetrics(t, h.metrics, []WantMetric{
		{"Memory/Physical", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"CPU/User/Utilization", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"CPU/System/Utilization", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Go/Runtime/Goroutines", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"GC/System/Pause Fraction", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"GC/System/Pauses", "", true, []float64{0, 0, 0, 0, 0, 0}},
	})
}
