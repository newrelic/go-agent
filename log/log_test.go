package log

import "testing"

func TestMergeContexts(t *testing.T) {
	out := mergeContexts()
	if 0 != len(out) {
		t.Fatal(out)
	}
	one := Context{"one": "zip"}
	two := Context{"two": "zap"}

	out = mergeContexts(one)
	if 1 != len(out) {
		t.Fatal(out)
	}
	if x := out["one"]; x != "zip" {
		t.Fatal(out["one"])
	}

	out = mergeContexts(one, two)
	if 2 != len(out) {
		t.Fatal(out)
	}
	if x := out["one"]; x != "zip" {
		t.Fatal(out["one"])
	}
	if x := out["two"]; x != "zap" {
		t.Fatal(out["two"])
	}
}
