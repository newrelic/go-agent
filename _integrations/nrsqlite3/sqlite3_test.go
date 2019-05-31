package nrsqlite3

import (
	"path/filepath"
	"testing"
)

func TestGetPortPathOrID(t *testing.T) {
	testdbAbsPath, err := filepath.Abs("test.db")
	if nil != err {
		t.Fatal(err)
	}

	testcases := []struct {
		dsn      string
		expected string
	}{
		{":memory:", ":memory:"},
		{"test.db", testdbAbsPath},
		{"file:/test.db?cache=shared&mode=memory", "/test.db"},
		{"file::memory:", ":memory:"},
		{"", ""},
	}

	for _, test := range testcases {
		if actual := getPortPathOrID(test.dsn); actual != test.expected {
			t.Errorf(`incorrect port path or id: dsn="%s", actual="%s"`, test.dsn, actual)
		}
	}
}
