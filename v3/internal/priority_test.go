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

func TestTraceStateFormat(t *testing.T) {
	testcases := []struct {
		input    float64
		expected string
	}{
		{input: 0, expected: "0"},
		{input: 0.1, expected: "0.1"},
		{input: 0.7654321, expected: "0.765432"},
		{input: 10.7654321, expected: "10.765432"},
		{input: 0.99999999999, expected: "1"},
	}

	for _, tc := range testcases {
		p := Priority(tc.input)
		if out := p.traceStateFormat(); out != tc.expected {
			t.Errorf("wrong priority format for %f: expected=%s actual=%s", tc.input, tc.expected, out)
		}
	}
}
