package newrelic

import (
	"testing"
)

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
