package internal

import (
	"runtime"
	"syscall"
	"time"

	"github.com/newrelic/go-agent/log"
)

type sample struct {
	when         time.Time
	memStats     runtime.MemStats
	userTime     time.Duration
	systemTime   time.Duration
	numGoroutine int
	numCPU       int
}

func timevalToDuration(tv syscall.Timeval) time.Duration {
	return time.Duration(tv.Nano()) * time.Nanosecond
}

func bytesToMebibytesFloat(bts uint64) float64 {
	return float64(bts) / ((float64)(1024 * 1024))
}

func getSample(now time.Time) *sample {
	s := sample{
		when:         now,
		numGoroutine: runtime.NumGoroutine(),
		numCPU:       runtime.NumCPU(),
	}

	// Gather CPU Usage
	ru := syscall.Rusage{}
	err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	if nil == err {
		s.userTime = timevalToDuration(ru.Utime)
		s.systemTime = timevalToDuration(ru.Stime)
	} else {
		log.Warn("unable to getrusage", log.Context{
			"error": err.Error(),
		})
	}

	// Gather MemStats
	runtime.ReadMemStats(&s.memStats)

	return &s
}

type samples struct {
	previous *sample
	current  *sample
}

type cpuStats struct {
	used     time.Duration
	fraction float64 // used / (elapsed * numCPU)
}

type stats struct {
	numGoroutine    int
	allocBytes      uint64
	user            cpuStats
	system          cpuStats
	gcPauseFraction float64
	deltaNumGC      uint32
	deltaPauseTotal time.Duration
	minPause        time.Duration
	maxPause        time.Duration
}

func getStats(ss samples) stats {
	cur := ss.current
	prev := ss.previous
	elapsed := cur.when.Sub(prev.when)

	s := stats{
		numGoroutine: cur.numGoroutine,
		allocBytes:   cur.memStats.Alloc,
	}

	// CPU Utilization
	totalCPUSeconds := elapsed.Seconds() * float64(cur.numCPU)
	if prev.userTime != 0 && cur.userTime > prev.userTime {
		s.user.used = cur.userTime - prev.userTime
		s.user.fraction = s.user.used.Seconds() / totalCPUSeconds
	}
	if prev.systemTime != 0 && cur.systemTime > prev.systemTime {
		s.system.used = cur.systemTime - prev.systemTime
		s.system.fraction = s.system.used.Seconds() / totalCPUSeconds
	}

	// GC Pause Fraction
	deltaPauseTotalNs := cur.memStats.PauseTotalNs - prev.memStats.PauseTotalNs
	frac := float64(deltaPauseTotalNs) / float64(elapsed.Nanoseconds())
	s.gcPauseFraction = frac

	// GC Pauses
	if deltaNumGC := cur.memStats.NumGC - prev.memStats.NumGC; deltaNumGC > 0 {
		var maxPauseNs uint64
		var minPauseNs uint64
		for i := prev.memStats.NumGC + 1; i <= cur.memStats.NumGC; i++ {
			pause := cur.memStats.PauseNs[(i+255)%256]
			if pause > maxPauseNs {
				maxPauseNs = pause
			}
			if 0 == minPauseNs || pause < minPauseNs {
				minPauseNs = pause
			}
		}
		s.deltaPauseTotal = time.Duration(deltaPauseTotalNs) * time.Nanosecond
		s.deltaNumGC = deltaNumGC
		s.minPause = time.Duration(minPauseNs) * time.Nanosecond
		s.maxPause = time.Duration(maxPauseNs) * time.Nanosecond
	}

	return s
}

func (s stats) mergeIntoHarvest(h *harvest) {
	h.metrics.addValue(runGoroutine, "", float64(s.numGoroutine), forced)
	h.metrics.addValueExclusive(memoryPhysical, "", bytesToMebibytesFloat(s.allocBytes), 0, forced)
	h.metrics.addValueExclusive(cpuUserUtilization, "", s.user.fraction, 0, forced)
	h.metrics.addValueExclusive(cpuSystemUtilization, "", s.system.fraction, 0, forced)
	h.metrics.addValue(cpuUserTime, "", s.user.used.Seconds(), forced)
	h.metrics.addValue(cpuSystemTime, "", s.system.used.Seconds(), forced)
	h.metrics.addValueExclusive(gcPauseFraction, "", s.gcPauseFraction, 0, forced)
	h.metrics.add(gcPauses, "", metricData{
		countSatisfied:  float64(s.deltaNumGC),
		totalTolerated:  s.deltaPauseTotal.Seconds(),
		exclusiveFailed: 0,
		min:             s.minPause.Seconds(),
		max:             s.maxPause.Seconds(),
		sumSquares:      s.deltaPauseTotal.Seconds() * s.deltaPauseTotal.Seconds(),
	}, forced)
}

func runSampler(app *App, period time.Duration) {
	previous := getSample(time.Now())

	for now := range time.Tick(period) {
		current := getSample(now)

		run := app.getRun()
		app.consume(run.RunID, getStats(samples{
			previous: previous,
			current:  current,
		}))
		previous = current
	}
}
