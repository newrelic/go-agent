package internal

import (
	"math"
	"sync"
	"time"
)

// AdaptiveSamplerInput holds input fields for the NewAdaptiveSampler function
type AdaptiveSamplerInput struct {
	Period time.Duration
	Target uint64
}

// AdaptiveSampler calculates which transactions should be sampled.
type AdaptiveSampler struct {
	sync.Mutex
	AdaptiveSamplerInput

	// Transactions with priority higher than this are sampled.
	// This is 1 - sampleRatio.
	priorityMin float32

	currentPeriod struct {
		numSampled uint64
		numSeen    uint64
		end        time.Time
	}
}

// NewAdaptiveSampler is a constructor-like function that gives us an AdaptiveSampler
func NewAdaptiveSampler(input AdaptiveSamplerInput, now time.Time) *AdaptiveSampler {
	as := &AdaptiveSampler{}
	as.AdaptiveSamplerInput = input
	as.currentPeriod.end = now.Add(input.Period)

	// Sample the first transactions in the first period.
	as.priorityMin = 0.0
	return as
}

// ComputeSampled calculates if the transaction should be sampled.
func (as *AdaptiveSampler) ComputeSampled(priority float32, now time.Time) bool {
	if nil == as {
		return false
	}

	as.Lock()
	defer as.Unlock()

	// If the current time is after the end of the "currentPeriod".  This is in
	// a `for`/`while` loop in case there's a harvest where no sampling happened.
	// i.e. for situations where a single call to
	//    as.currentPeriod.end = as.currentPeriod.end.Add(as.period)
	// might not catch us up to the current period
	for now.After(as.currentPeriod.end) {
		as.priorityMin = 0.0
		if as.currentPeriod.numSeen > 0 {
			sampledRatio := float32(as.Target) / float32(as.currentPeriod.numSeen)
			as.priorityMin = 1.0 - sampledRatio
		}
		as.currentPeriod.numSampled = 0
		as.currentPeriod.numSeen = 0
		as.currentPeriod.end = as.currentPeriod.end.Add(as.Period)
	}

	as.currentPeriod.numSeen++

	// exponential backoff -- if the number of sampled items is greater than our
	// target, we need to apply the exponential backoff
	if as.currentPeriod.numSampled > as.Target {
		if as.computeSampledBackoff(as.Target, as.currentPeriod.numSeen, as.currentPeriod.numSampled) {
			as.currentPeriod.numSampled++
			return true
		}
		return false
	} else if as.currentPeriod.numSampled > as.Target {
		return false
	}

	if priority >= as.priorityMin {
		as.currentPeriod.numSampled++
		return true
	}

	return false
}

func (as *AdaptiveSampler) computeSampledBackoff(target uint64, decidedCount uint64, sampledTrueCount uint64) bool {
	return float64(RandUint64N(decidedCount)) <
		math.Pow(float64(target), (float64(target)/float64(sampledTrueCount)))-math.Pow(float64(target), 0.5)
}
