package internal

import (
	"encoding/json"
	"testing"
)

func TestGetStackTrace(t *testing.T) {
	stack := GetStackTrace(0)
	js, err := json.Marshal(stack)
	if nil != err {
		t.Fatal(err)
	}
	if nil == js {
		t.Fatal(string(js))
	}
}

func TestSimplifyStackTraceFilename(t *testing.T) {
	tcs := []struct {
		Input  string
		Expect string
	}{
		{"", ""},
		{"zop.go", "zop.go"},
		{"/zip/zop.go", "/zip/zop.go"},
		{"/gopath/src/zip/zop.go", "zip/zop.go"},
		{"/gopath/src/zip/src/zop.go", "zip/src/zop.go"},
		{"/gopath/src/zip/zop.go", "zip/zop.go"},
		{"/日本/src/日本/zop.go", "日本/zop.go"},
		{"/src/", ""},
		{"/src/zop.go", "zop.go"},
	}

	for _, tc := range tcs {
		out := simplifyStackTraceFilename(tc.Input)
		if out != tc.Expect {
			t.Error(tc.Input, tc.Expect, out)
		}
	}
}
