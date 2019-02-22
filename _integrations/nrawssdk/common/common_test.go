package common

import (
	"testing"
)

func TestGetTableName(t *testing.T) {
	var str string = "this is a string"
	var emptyStr string
	strPtr := &str
	emptyStrPtr := &emptyStr

	testcases := []struct {
		params   interface{}
		expected string
	}{
		{params: nil, expected: ""},
		{params: str, expected: ""},
		{params: strPtr, expected: ""},
		{params: struct{ other string }{other: str}, expected: ""},
		{params: &struct{ other string }{other: str}, expected: ""},
		{params: struct{ TableName bool }{TableName: true}, expected: ""},
		{params: &struct{ TableName bool }{TableName: true}, expected: ""},
		{params: struct{ TableName string }{TableName: str}, expected: ""},
		{params: &struct{ TableName string }{TableName: str}, expected: ""},
		{params: struct{ TableName *string }{TableName: nil}, expected: ""},
		{params: &struct{ TableName *string }{TableName: nil}, expected: ""},
		{params: struct{ TableName *string }{TableName: emptyStrPtr}, expected: ""},
		{params: &struct{ TableName *string }{TableName: emptyStrPtr}, expected: ""},
		{params: struct{ TableName *string }{TableName: strPtr}, expected: ""},
		{params: &struct{ TableName *string }{TableName: strPtr}, expected: str},
	}

	for i, test := range testcases {
		if out := getTableName(test.params); test.expected != out {
			t.Error(i, out, test.params, test.expected)
		}
	}
}
