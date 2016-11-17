package internal

import (
	"bytes"
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

func TestStackTrace_WriteJSON(t *testing.T) {
	st := StackTrace{
		{
			File:     "foo.go",
			Line:     42,
			Function: "DoStuff",
		},
		{
			File:     "main.go",
			Line:     6,
			Function: "main",
		},
	}

	b := &bytes.Buffer{}
	st.WriteJSON(b)
	t.Log(b.String())

	actual := []map[string]interface{}{}
	err := json.NewDecoder(b).Decode(&actual)
	if nil != err {
		t.Fatal(err)
	}

	if 2 != len(actual) {
		t.Fatalf("want 2 elements but got %v", len(actual))
	}

	requireField := func(i int, field string, want interface{}) {
		have, ok := actual[i][field]
		if !ok {
			t.Errorf("JSON encoded stack frame should have field %q", field)
		}

		if have != want {
			t.Errorf("Field %q index %d: want %v (%T) have %+v (%T)", field, i, want, want, have, have)
		}
	}

	for i := range actual {
		requireField(i, "filepath", st[i].File)
		requireField(i, "name", st[i].Function)
		requireField(i, "line", float64(st[i].Line)) // unmarshalling an integer from JSON into interface{} becomes a float
	}
}
