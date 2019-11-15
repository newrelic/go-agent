package internal

import (
	"testing"
	"time"
)

func assert(t testing.TB, expectTrue bool) {
	if h, ok := t.(interface {
		Helper()
	}); ok {
		h.Helper()
	}
	if !expectTrue {
		t.Error(expectTrue)
	}
}

func TestDefaultReplyValidSampler(t *testing.T) {
	reply := ConnectReplyDefaults()
	assert(t, !reply.AdaptiveSampler.ComputeSampled(1.0, time.Now()))
}

func TestAdaptiveSampler(t *testing.T) {
	start := time.Now()
	sampler := NewAdaptiveSampler(60*time.Second, 2, start)

	// first period -- we're guaranteed to get 2 sampled
	// due to our target, and we'll send through a total of 4
	assert(t, sampler.ComputeSampled(0.0, start))
	assert(t, sampler.ComputeSampled(0.0, start))
	sampler.ComputeSampled(0.0, start)
	sampler.ComputeSampled(0.0, start)

	// Next period!  4 calls in the last period means a new sample ratio
	// of 1/2.  Nothing with a priority less than the ratio will get through
	now := start.Add(61 * time.Second)
	assert(t, !sampler.ComputeSampled(0.0, now))
	assert(t, !sampler.ComputeSampled(0.0, now))
	assert(t, !sampler.ComputeSampled(0.0, now))
	assert(t, !sampler.ComputeSampled(0.0, now))
	assert(t, !sampler.ComputeSampled(0.49, now))
	assert(t, !sampler.ComputeSampled(0.49, now))

	// but these two will get through, and we'll still be under
	// our target rate so there's no random sampling to deal with
	assert(t, sampler.ComputeSampled(0.55, now))
	assert(t, sampler.ComputeSampled(1.0, now))

	// Next period!  8 calls in the last period means a new sample ratio
	// of 1/4.
	now = start.Add(121 * time.Second)
	assert(t, !sampler.ComputeSampled(0.0, now))
	assert(t, !sampler.ComputeSampled(0.5, now))
	assert(t, !sampler.ComputeSampled(0.7, now))
	assert(t, sampler.ComputeSampled(0.8, now))
}

func TestAdaptiveSamplerSkipPeriod(t *testing.T) {
	start := time.Now()
	sampler := NewAdaptiveSampler(60*time.Second, 2, start)

	// same as the previous test, we know we can get two through
	// and we'll send a total of 4 through
	assert(t, sampler.ComputeSampled(0.0, start))
	assert(t, sampler.ComputeSampled(0.0, start))
	sampler.ComputeSampled(0.0, start)
	sampler.ComputeSampled(0.0, start)

	// Two periods later!  Since there was a period with no samples, priorityMin
	// should be zero

	now := start.Add(121 * time.Second)
	assert(t, sampler.ComputeSampled(0.0, now))
	assert(t, sampler.ComputeSampled(0.0, now))
}

func TestAdaptiveSamplerTarget(t *testing.T) {
	var target uint64
	target = 20
	start := time.Now()
	sampler := NewAdaptiveSampler(60*time.Second, target, start)

	// we should always sample up to the number of target events
	for i := 0; uint64(i) < target; i++ {
		assert(t, sampler.ComputeSampled(0.0, start))
	}

	// but now further calls to ComputeSampled are subject to exponential backoff.
	// this means their sampling is subject to a bit of randomness and we have no
	// guarantee of a true or false sample, just an increasing unlikeliness that
	// things will be sampled
}
