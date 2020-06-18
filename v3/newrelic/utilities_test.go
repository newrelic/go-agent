// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"net/http"
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

func TestTimeToFloatMilliseconds(t *testing.T) {
	tm := time.Unix(123, 456789000)
	if ms := timeToFloatMilliseconds(tm); ms != 123456.789 {
		t.Error(ms)
	}
}

func TestCompactJSON(t *testing.T) {
	in := `
	{   "zip":	1}`
	out := compactJSONString(in)
	if out != `{"zip":1}` {
		t.Fatal(in, out)
	}
}

func TestGetContentLengthFromHeader(t *testing.T) {
	// Nil header.
	if cl := getContentLengthFromHeader(nil); cl != -1 {
		t.Errorf("unexpected content length: expected -1; got %d", cl)
	}

	// Empty header.
	header := make(http.Header)
	if cl := getContentLengthFromHeader(header); cl != -1 {
		t.Errorf("unexpected content length: expected -1; got %d", cl)
	}

	// Invalid header.
	header.Set("Content-Length", "foo")
	if cl := getContentLengthFromHeader(header); cl != -1 {
		t.Errorf("unexpected content length: expected -1; got %d", cl)
	}

	// Zero header.
	header.Set("Content-Length", "0")
	if cl := getContentLengthFromHeader(header); cl != 0 {
		t.Errorf("unexpected content length: expected 0; got %d", cl)
	}

	// Valid, non-zero header.
	header.Set("Content-Length", "1024")
	if cl := getContentLengthFromHeader(header); cl != 1024 {
		t.Errorf("unexpected content length: expected 1024; got %d", cl)
	}
}

func TestStringLengthByteLimit(t *testing.T) {
	testcases := []struct {
		input  string
		limit  int
		expect string
	}{
		{"", 255, ""},
		{"awesome", -1, ""},
		{"awesome", 0, ""},
		{"awesome", 1, "a"},
		{"awesome", 7, "awesome"},
		{"awesome", 20, "awesome"},
		{"日本\x80語", 10, "日本\x80語"}, // bad unicode
		{"日本", 1, ""},
		{"日本", 2, ""},
		{"日本", 3, "日"},
		{"日本", 4, "日"},
		{"日本", 5, "日"},
		{"日本", 6, "日本"},
		{"日本", 7, "日本"},
	}

	for _, tc := range testcases {
		out := stringLengthByteLimit(tc.input, tc.limit)
		if out != tc.expect {
			t.Error(tc.input, tc.limit, tc.expect, out)
		}
	}
}

func TestTimeToAndFromUnixMilliseconds(t *testing.T) {
	t1 := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	millis := timeToUnixMilliseconds(t1)
	if millis != 1417136460000 {
		t.Fatal(millis)
	}
	t2 := timeFromUnixMilliseconds(millis)
	if t1.UnixNano() != t2.UnixNano() {
		t.Fatal(t1, t2)
	}
}

func TestMinorVersion(t *testing.T) {
	testcases := []struct {
		input  string
		expect string
	}{
		{"go1.13", "go1.13"},
		{"go1.13.1", "go1.13"},
		{"go1.13.1.0", "go1.13"},
		{"purple", "purple"},
	}

	for _, test := range testcases {
		if actual := minorVersion(test.input); actual != test.expect {
			t.Errorf("incorrect result: expect=%s actual=%s", test.expect, actual)
		}
	}
}
