// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"
	"time"
)

func TestFloatSecondsToDuration(t *testing.T) {
	if d := FloatSecondsToDuration(0.123); d != 123*time.Millisecond {
		t.Error(d)
	}
	if d := FloatSecondsToDuration(456.0); d != 456*time.Second {
		t.Error(d)
	}
}

func TestCompactJSON(t *testing.T) {
	in := `
	{   "zip":	1}`
	out := CompactJSONString(in)
	if out != `{"zip":1}` {
		t.Fatal(in, out)
	}
}
