// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrsqlite3

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetPortPathOrID(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(here)

	testcases := []struct {
		dsn      string
		expected string
	}{
		{":memory:", ":memory:"},
		{"test.db", filepath.Join(currentDir, "test.db")},
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
