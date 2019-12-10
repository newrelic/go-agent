package sysinfo

import (
	"os"
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
			envVarValue:      "exampleasdfasdfasdf",
			expected:         "example.*",
		},
	}

	for _, test := range testcases {
		getenv := func(string) string { return test.envVarValue }
		if actual := getDynoName(getenv, test.useDynoNames, test.dynoNamePrefixes); actual != test.expected {
			t.Errorf("unexpected output: actual=%s expected=%s", actual, test.expected)
		}
	}
}

func TestGetHostname(t *testing.T) {
	getenv := func(string) string { return "dynoname" }

	// dyno name takes precedence
	ResetHostname()
	host, err := getHostname(getenv, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if host != "dynoname" {
		t.Errorf("dyno name is not set as hostname, actual=%s", host)
	}

	// os hostname used when getting dyno name failes
	ResetHostname()
	host, err = getHostname(getenv, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	oshost, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	if host != oshost {
		t.Errorf("os.Hostname is not set as hostname, actual=%s", host)
	}

	// cached value of hostname is used
	host, err = getHostname(getenv, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if host != oshost {
		t.Errorf("cached hostname is not being used, actual=%s", host)
	}
}
