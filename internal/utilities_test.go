package internal

import (
	"testing"
	"time"
)

func TestRemoveFirstSegment(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{input: "no_seperators", expected: "no_seperators"},
		{input: "heyo/zip/zap", expected: "zip/zap"},
		{input: "ends_in_slash/", expected: ""},
		{input: "☃☃☃/✓✓✓/heyo", expected: "✓✓✓/heyo"},
		{input: "☃☃☃/", expected: ""},
		{input: "/", expected: ""},
		{input: "", expected: ""},
	}

	for _, tc := range testcases {
		out := removeFirstSegment(tc.input)
		if out != tc.expected {
			t.Fatal(tc.input, out, tc.expected)
		}
	}
}

func TestfloatSecondsToDuration(t *testing.T) {
	if d := floatSecondsToDuration(0.123); d != 123*time.Millisecond {
		t.Error(d)
	}
	if d := floatSecondsToDuration(456.0); d != 456*time.Second {
		t.Error(d)
	}
}

func TestAbsTimeDiff(t *testing.T) {
	diff := 5 * time.Second
	before := time.Now()
	after := before.Add(5 * time.Second)

	if out := absTimeDiff(before, after); out != diff {
		t.Error(out, diff)
	}
	if out := absTimeDiff(after, before); out != diff {
		t.Error(out, diff)
	}
	if out := absTimeDiff(after, after); out != 0 {
		t.Error(out)
	}
}

func TestTimeToFloatMilliseconds(t *testing.T) {
	tm := time.Unix(123, 456789000)
	if ms := timeToFloatMilliseconds(tm); ms != 123456.789 {
		t.Error(ms)
	}
}
