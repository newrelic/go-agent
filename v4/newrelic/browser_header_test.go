// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"fmt"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
)

func TestNilBrowserTimingHeader(t *testing.T) {
	var h *BrowserTimingHeader

	// The methods on a nil BrowserTimingHeader pointer should not panic.

	if out := h.WithTags(); out != nil {
		t.Errorf("unexpected WithTags output for a disabled header: expected a blank string; got %s", out)
	}

	if out := h.WithoutTags(); out != nil {
		t.Errorf("unexpected WithoutTags output for a disabled header: expected a blank string; got %s", out)
	}
}

func TestEnabled(t *testing.T) {
	// We're not trying to test Go's JSON marshalling here; we just want to
	// ensure that we get the right fields out the other side.
	expectInfo := internal.CompactJSONString(`
    {
      "beacon": "brecon",
      "licenseKey": "12345",
      "applicationID": "app",
      "transactionName": "txn",
      "queueTime": 1,
      "applicationTime": 2,
      "atts": "attrs",
      "errorBeacon": "blah",
      "agent": "bond"
    }
  `)

	h := &BrowserTimingHeader{
		agentLoader: "loader();",
		info: browserInfo{
			Beacon:                "brecon",
			LicenseKey:            "12345",
			ApplicationID:         "app",
			TransactionName:       "txn",
			QueueTimeMillis:       1,
			ApplicationTimeMillis: 2,
			ObfuscatedAttributes:  "attrs",
			ErrorBeacon:           "blah",
			Agent:                 "bond",
		},
	}

	expected := fmt.Sprintf("%s%s%s%s%s", browserStartTag, h.agentLoader, browserInfoPrefix, expectInfo, browserEndTag)
	if actual := h.WithTags(); string(actual) != expected {
		t.Errorf("unexpected WithTags output: expected %s; got %s", expected, string(actual))
	}

	expected = fmt.Sprintf("%s%s%s", h.agentLoader, browserInfoPrefix, expectInfo)
	if actual := h.WithoutTags(); string(actual) != expected {
		t.Errorf("unexpected WithoutTags output: expected %s; got %s", expected, string(actual))
	}
}

func TestBrowserAttributesNil(t *testing.T) {
	expected := `{"u":{},"a":{}}`
	actual := string(browserAttributes(nil))
	if expected != actual {
		t.Errorf("unexpected browser attributes: expected %s; got %s", expected, actual)
	}
}

func TestBrowserAttributes(t *testing.T) {
	config := config{Config: defaultConfig()}
	config.BrowserMonitoring.Attributes.Enabled = true
	a := newAttributes(createAttributeConfig(config, true))
	addUserAttribute(a, "user", "thing", destBrowser)
	addUserAttribute(a, "not", "shown", destError)
	a.Agent.Add(AttributeHostDisplayName, "host", nil)

	expected := `{"u":{"user":"thing"},"a":{}}`
	actual := string(browserAttributes(a))
	if expected != actual {
		t.Errorf("unexpected browser attributes: expected %s; got %s", expected, actual)
	}
}
