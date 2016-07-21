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
