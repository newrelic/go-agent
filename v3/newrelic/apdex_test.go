// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"testing"
	"time"
)

func dur(d int) time.Duration {
	return time.Duration(d)
}

func TestCalculateApdexZone(t *testing.T) {
	if z := calculateApdexZone(dur(10), dur(1)); z != apdexSatisfying {
		t.Fatal(z)
	}
	if z := calculateApdexZone(dur(10), dur(10)); z != apdexSatisfying {
		t.Fatal(z)
	}
	if z := calculateApdexZone(dur(10), dur(11)); z != apdexTolerating {
		t.Fatal(z)
	}
	if z := calculateApdexZone(dur(10), dur(40)); z != apdexTolerating {
		t.Fatal(z)
	}
	if z := calculateApdexZone(dur(10), dur(41)); z != apdexFailing {
		t.Fatal(z)
	}
	if z := calculateApdexZone(dur(10), dur(100)); z != apdexFailing {
		t.Fatal(z)
	}
}

func TestApdexLabel(t *testing.T) {
	if out := apdexSatisfying.label(); "S" != out {
		t.Fatal(out)
	}
	if out := apdexTolerating.label(); "T" != out {
		t.Fatal(out)
	}
	if out := apdexFailing.label(); "F" != out {
		t.Fatal(out)
	}
	if out := apdexNone.label(); "" != out {
		t.Fatal(out)
	}
}
