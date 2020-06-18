// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"
)

func TestBrowserAttributesNil(t *testing.T) {
	expected := `{"u":{},"a":{}}`
	actual := string(BrowserAttributes(nil))
	if expected != actual {
		t.Errorf("unexpected browser attributes: expected %s; got %s", expected, actual)
	}
}

func TestBrowserAttributes(t *testing.T) {
	a := NewAttributes(CreateAttributeConfig(sampleAttributeConfigInput, true))
	AddUserAttribute(a, "user", "thing", destBrowser)
	AddUserAttribute(a, "not", "shown", destError)
	a.Agent.Add(AttributeHostDisplayName, "host", nil)

	expected := `{"u":{"user":"thing"},"a":{}}`
	actual := string(BrowserAttributes(a))
	if expected != actual {
		t.Errorf("unexpected browser attributes: expected %s; got %s", expected, actual)
	}
}
