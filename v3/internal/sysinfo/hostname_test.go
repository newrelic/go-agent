package sysinfo

import (
	"testing"
)

func TestGetDynoName(t *testing.T) {
	testcases := []struct {
		useDynoNames     bool
		dynoNamePrefixes []string
		envVarValue      string
		expected         string
	}{
		{
			useDynoNames: false,
			envVarValue:  "dynoname",
			expected:     "",
		},
		{
			useDynoNames: true,
			envVarValue:  "",
			expected:     "",
		},
		{
			useDynoNames: true,
			envVarValue:  "dynoname",
			expected:     "dynoname",
		},
		{
			useDynoNames:     true,
			dynoNamePrefixes: []string{"example"},
			envVarValue:      "dynoname",
			expected:         "dynoname",
		},
		{
			useDynoNames:     true,
			dynoNamePrefixes: []string{""},
			envVarValue:      "dynoname",
			expected:         "dynoname",
		},
		{
			useDynoNames:     true,
			dynoNamePrefixes: []string{"example", "ex"},
			envVarValue:      "example.asdfasdfasdf",
			expected:         "example.*",
		},
		{
			useDynoNames:     true,
			dynoNamePrefixes: []string{"example", "ex"},
			envVarValue:      "exampleasdfasdfasdf",
			expected:         "exampleasdfasdfasdf",
		},
	}

	for _, test := range testcases {
		getenv := func(string) string { return test.envVarValue }
		if actual := getDynoName(getenv, test.useDynoNames, test.dynoNamePrefixes); actual != test.expected {
			t.Errorf("unexpected output: actual=%s expected=%s", actual, test.expected)
		}
	}
}
