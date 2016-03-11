package internal

import (
	"encoding/json"
	"strings"
	"testing"

	"go.datanerd.us/p/will/newrelic/internal/crossagent"
)

func TestCrossAgentSegmentTerms(t *testing.T) {
	var tcs []struct {
		Testname string       `json:"testname"`
		Rules    SegmentRules `json:"transaction_segment_terms"`
		Tests    []struct {
			Input    string `json:"input"`
			Expected string `json:"expected"`
		} `json:"tests"`
	}

	err := crossagent.ReadJSON("transaction_segment_terms.json", &tcs)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tcs {
		for _, test := range tc.Tests {
			out := tc.Rules.Apply(test.Input)
			if out != test.Expected {
				t.Fatal(tc.Testname, test.Input, out, test.Expected)
			}
		}
	}
}

func TestSegmentTerms(t *testing.T) {
	js := `[
      {
         "prefix":"WebTransaction\/Uri",
         "terms":[
            "two",
            "Users",
            "willhf",
            "dev",
            "php",
            "one",
            "alpha",
            "zap"
         ]
      }
   ]`
	var rules SegmentRules
	if err := json.Unmarshal([]byte(js), &rules); nil != err {
		t.Fatal(err)
	}

	out := rules.Apply("WebTransaction/Uri/pen/two/pencil/dev/paper")
	if out != "WebTransaction/Uri/*/two/*/dev/*" {
		t.Fatal(out)
	}
}

func TestEmptySegmentTerms(t *testing.T) {
	var rules SegmentRules

	input := "my/name"
	out := rules.Apply(input)
	if out != input {
		t.Error(input, out)
	}
}

func BenchmarkSegmentTerms(b *testing.B) {
	js := `[
      {
         "prefix":"WebTransaction\/Uri",
         "terms":[
            "two",
            "Users",
            "willhf",
            "dev",
            "php",
            "one",
            "alpha",
            "zap"
         ]
      }
   ]`
	var rules SegmentRules
	if err := json.Unmarshal([]byte(js), &rules); nil != err {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	input := "WebTransaction/Uri/pen/two/pencil/dev/paper"
	expected := "WebTransaction/Uri/*/two/*/dev/*"
	for i := 0; i < b.N; i++ {
		out := rules.Apply(input)
		if out != expected {
			b.Fatal(out, expected)
		}
	}
}

func TestCollapsePlaceholders(t *testing.T) {
	testcases := []struct {
		input  string
		expect string
	}{
		{input: "", expect: ""},
		{input: "/", expect: "/"},
		{input: "*", expect: "*"},
		{input: "*/*", expect: "*"},
		{input: "a/b/c", expect: "a/b/c"},
		{input: "*/*/*", expect: "*"},
		{input: "a/*/*/*/b", expect: "a/*/b"},
		{input: "a/b/*/*/*/", expect: "a/b/*/"},
		{input: "a/b/*/*/*", expect: "a/b/*"},
		{input: "*/*/a/b/*/*/*", expect: "*/a/b/*"},
		{input: "*/*/a/b/*/c/*/*/d/e/*/*/*", expect: "*/a/b/*/c/*/d/e/*"},
		{input: "a/*/b", expect: "a/*/b"},
	}

	for _, tc := range testcases {
		segments := strings.Split(tc.input, "/")
		segments = collapsePlaceholders(segments)
		out := strings.Join(segments, "/")
		if out != tc.expect {
			t.Error(tc.input, tc.expect, out)
		}
	}
}
