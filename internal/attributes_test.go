package internal

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"go.datanerd.us/p/will/newrelic/internal/crossagent"
)

type AttributeTestcase struct {
	Testname             string                      `json:"testname"`
	Config               map[string]*json.RawMessage `json:"config"`
	Key                  string                      `json:"input_key"`
	InputDestinations    []string                    `json:"input_default_destinations"`
	ExpectedDestinations []string                    `json:"expected_destinations"`
}

func destinationsFromArray(dests []string) destination {
	d := destinationNone
	for _, s := range dests {
		switch s {
		case "transaction_events":
			d |= destinationEvent
		case "transaction_tracer":
			d |= destinationTrace
		case "error_collector":
			d |= destinationError
		case "browser_monitoring":
			d |= destinationBrowser
		}
	}
	return d
}

func getActualDestinations(attributes *attributes, key string) destination {
	dests := []destination{
		destinationEvent,
		destinationTrace,
		destinationError,
		destinationBrowser,
	}
	actual := destinationNone

	for _, d := range dests {
		out := attributes.GetAgent(d)
		if _, ok := out[key]; ok {
			actual |= d
		}
	}
	return actual
}

func (tc *AttributeTestcase) disable(t *testing.T, val *json.RawMessage, setting *bool) {
	var b bool

	if err := json.Unmarshal(*val, &b); nil != err {
		t.Fatal(tc.Testname, err)
	}
	*setting = b
}

func (tc *AttributeTestcase) includeExclude(t *testing.T, val *json.RawMessage) []string {
	var matches []string

	if err := json.Unmarshal(*val, &matches); nil != err {
		t.Fatal(tc.Testname, err)
	}
	return matches
}

func (tc *AttributeTestcase) processSetting(t *testing.T, cfg *attributeConfig, key string, val *json.RawMessage) {

	segments := strings.Split(key, ".")
	if len(segments) < 2 {
		t.Fatal(tc.Testname, key)
	}

	var dc *destinationConfig
	switch segments[0] {
	case "attributes":
		dc = &cfg.All
	case "transaction_events":
		dc = &cfg.TransactionEvents
	case "transaction_tracer":
		dc = &cfg.TransactionTracer
	case "error_collector":
		dc = &cfg.ErrorCollector
	case "browser_monitoring":
		dc = &cfg.BrowserMonitoring
	default:
		t.Fatal(tc.Testname, key)
	}

	switch segments[len(segments)-1] {
	case "enabled":
		tc.disable(t, val, &dc.Enabled)
	case "include":
		dc.Include = tc.includeExclude(t, val)
	case "exclude":
		dc.Exclude = tc.includeExclude(t, val)
	default:
		t.Fatal(tc.Testname, key)
	}
}

func (tc *AttributeTestcase) Run(t *testing.T) {
	inputDests := destinationsFromArray(tc.InputDestinations)
	expectedDests := destinationsFromArray(tc.ExpectedDestinations)

	config := defaultAttributeConfig

	for key, val := range tc.Config {
		tc.processSetting(t, &config, key, val)
	}

	attributes := createAttributes(&config)

	attributes.addAgent(tc.Key, interface{}(1), inputDests)
	actualDests := getActualDestinations(attributes, tc.Key)

	if actualDests != expectedDests {
		t.Error(tc.Testname, expectedDests, actualDests)
	}
}

func TestCrossAgentAttributes(t *testing.T) {
	var tcs []AttributeTestcase

	err := crossagent.ReadJSON("attribute_configuration.json", &tcs)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tcs {
		tc.Run(t)
	}
}

func toJSON(x interface{}) string {
	out, err := json.Marshal(x)
	if nil != err {
		return ""
	}
	return string(out)
}

// Only works if there is a single attribute, since map iteration order is
// undefined.
func testUserAttributesAsJSON(t *testing.T, attributes *attributes, expected string) {
	out := attributes.GetUser(destinationAll)
	json := toJSON(out)
	if json != expected {
		t.Fatal(json, expected)
	}
}

const (
	a324 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

func TestStringLengthLimits(t *testing.T) {
	// Value is too long
	attributes := createAttributes(&defaultAttributeConfig)
	attributes.addUser("alpha", a324, destinationAll)
	testUserAttributesAsJSON(t, attributes, `{"alpha":"`+
		`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`+
		`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`+
		`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`+
		`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)

	// Key is too long
	attributes = createAttributes(&defaultAttributeConfig)
	attributes.addUser(a324, "alpha", destinationAll)
	testUserAttributesAsJSON(t, attributes, `{}`)
}

func TestBadValue(t *testing.T) {
	attributes := createAttributes(&defaultAttributeConfig)
	attributes.addUser("alpha", make(map[string]string), destinationAll)
	testUserAttributesAsJSON(t, attributes, `{}`)
}

func TestNullValue(t *testing.T) {
	attributes := createAttributes(&defaultAttributeConfig)
	attributes.addUser("alpha", nil, destinationAll)
	testUserAttributesAsJSON(t, attributes, `{"alpha":null}`)
}

func TestIntValue(t *testing.T) {
	attributes := createAttributes(&defaultAttributeConfig)
	attributes.addUser("alpha", 123, destinationAll)
	testUserAttributesAsJSON(t, attributes, `{"alpha":123}`)
}

func TestFloatValue(t *testing.T) {
	attributes := createAttributes(&defaultAttributeConfig)
	attributes.addUser("alpha", 44.55, destinationAll)
	testUserAttributesAsJSON(t, attributes, `{"alpha":44.55}`)
}

func TestStringValue(t *testing.T) {
	attributes := createAttributes(&defaultAttributeConfig)
	attributes.addUser("alpha", "beta", destinationAll)
	testUserAttributesAsJSON(t, attributes, `{"alpha":"beta"}`)
}

func TestBoolValue(t *testing.T) {
	attributes := createAttributes(&defaultAttributeConfig)
	attributes.addUser("alpha", true, destinationAll)
	testUserAttributesAsJSON(t, attributes, `{"alpha":true}`)
}

func TestReplacement(t *testing.T) {
	attributes := createAttributes(&defaultAttributeConfig)
	attributes.addUser("alpha", 1, destinationAll)
	attributes.addUser("alpha", 2, destinationAll)
	attributes.addUser("alpha", 3, destinationAll)
	testUserAttributesAsJSON(t, attributes, `{"alpha":3}`)
}

func TestUserLimit(t *testing.T) {
	attributes := createAttributes(&defaultAttributeConfig)

	for i := 0; i < attributeUserLimit; i++ {
		s := strconv.Itoa(i)
		attributes.addUser(s, s, destinationAll)
	}

	attributes.addUser("cant_add_me", 123, destinationAll)

	out := attributes.GetUser(destinationAll)
	js := toJSON(out)
	if len(out) != attributeUserLimit {
		t.Fatal(len(out), attributeUserLimit)
	}
	if strings.Contains(js, "cant_add_me") {
		t.Fatal(js)
	}

	// Now test that replacement works when the limit is reached
	attributes.addUser("0", "BEEN_REPLACED", destinationAll)
	out = attributes.GetUser(destinationAll)
	js = toJSON(out)
	if len(out) != attributeUserLimit {
		t.Fatal(len(out), attributeUserLimit)
	}
	if !strings.Contains(js, "BEEN_REPLACED") {
		t.Fatal(js)
	}
}
