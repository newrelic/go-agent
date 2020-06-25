// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"
)

func TestIsLowerPriority(t *testing.T) {
	low := Priority(0.0)
	middle := Priority(0.1)
	high := Priority(0.999999)

	if !low.isLowerPriority(middle) {
		t.Error(low, middle)
	}

	if high.isLowerPriority(middle) {
		t.Error(high, middle)
	}

	if high.isLowerPriority(high) {
		t.Error(high, high)
	}
}
