// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/newrelic/go-agent/v3/internal/crossagent"
)

type AttributeTestcase struct {
	Testname string `json:"testname"`
	Config   struct {
		AttributesEnabled        bool     `json:"attributes.enabled"`
		AttributesInclude        []string `json:"attributes.include"`
		AttributesExclude        []string `json:"attributes.exclude"`
		BrowserAttributesEnabled bool     `json:"browser_monitoring.attributes.enabled"`
		BrowserAttributesInclude []string `json:"browser_monitoring.attributes.include"`
		BrowserAttributesExclude []string `json:"browser_monitoring.attributes.exclude"`
		ErrorAttributesEnabled   bool     `json:"error_collector.attributes.enabled"`
		ErrorAttributesInclude   []string `json:"error_collector.attributes.include"`
		ErrorAttributesExclude   []string `json:"error_collector.attributes.exclude"`
		EventsAttributesEnabled  bool     `json:"transaction_events.attributes.enabled"`
		EventsAttributesInclude  []string `json:"transaction_events.attributes.include"`
		EventsAttributesExclude  []string `json:"transaction_events.attributes.exclude"`
		TracerAttributesEnabled  bool     `json:"transaction_tracer.attributes.enabled"`
		TracerAttributesInclude  []string `json:"transaction_tracer.attributes.include"`
		TracerAttributesExclude  []string `json:"transaction_tracer.attributes.exclude"`
	} `json:"config"`
	Key                  string   `json:"input_key"`
	InputDestinations    []string `json:"input_default_destinations"`
	ExpectedDestinations []string `json:"expected_destinations"`
}

var (
	destTranslate = map[string]destinationSet{
		"attributes":         destAll,
		"transaction_events": destTxnEvent,
		"transaction_tracer": destTxnTrace,
		"error_collector":    destError,
		"browser_monitoring": destBrowser,
	}
)

func destinationsFromArray(dests []string) destinationSet {
	d := destNone
	for _, s := range dests {
		if x, ok := destTranslate[s]; ok {
			d |= x
		}
	}
	return d
}

func destToString(d destinationSet) string {
	if 0 == d {
		return "none"
	}
	out := ""
	for _, ds := range []struct {
		Name string
		Dest destinationSet
	}{
		{Name: "event", Dest: destTxnEvent},
		{Name: "trace", Dest: destTxnTrace},
		{Name: "error", Dest: destError},
		{Name: "browser", Dest: destBrowser},
		{Name: "span", Dest: destSpan},
		{Name: "segment", Dest: destSegment},
	} {
		if 0 != d&ds.Dest {
			if "" == out {
				out = ds.Name
			} else {
				out = out + "," + ds.Name
			}
		}
	}
	return out
}

func runAttributeTestcase(t *testing.T, js json.RawMessage) {
	var tc AttributeTestcase

	tc.Config.AttributesEnabled = true
	tc.Config.BrowserAttributesEnabled = false
	tc.Config.ErrorAttributesEnabled = true
	tc.Config.EventsAttributesEnabled = true
	tc.Config.TracerAttributesEnabled = true

	if err := json.Unmarshal(js, &tc); nil != err {
		t.Error(err)
		return
	}

	config := config{}
	config.Attributes.Enabled = tc.Config.AttributesEnabled
	config.Attributes.Include = tc.Config.AttributesInclude
	config.Attributes.Exclude = tc.Config.AttributesExclude

	config.ErrorCollector.Attributes.Enabled = tc.Config.ErrorAttributesEnabled
	config.ErrorCollector.Attributes.Include = tc.Config.ErrorAttributesInclude
	config.ErrorCollector.Attributes.Exclude = tc.Config.ErrorAttributesExclude

	config.TransactionEvents.Attributes.Enabled = tc.Config.EventsAttributesEnabled
	config.TransactionEvents.Attributes.Include = tc.Config.EventsAttributesInclude
	config.TransactionEvents.Attributes.Exclude = tc.Config.EventsAttributesExclude

	config.BrowserMonitoring.Attributes.Enabled = tc.Config.BrowserAttributesEnabled
	config.BrowserMonitoring.Attributes.Include = tc.Config.BrowserAttributesInclude
	config.BrowserMonitoring.Attributes.Exclude = tc.Config.BrowserAttributesExclude

	config.TransactionTracer.Attributes.Enabled = tc.Config.TracerAttributesEnabled
	config.TransactionTracer.Attributes.Include = tc.Config.TracerAttributesInclude
	config.TransactionTracer.Attributes.Exclude = tc.Config.TracerAttributesExclude

	cfg := createAttributeConfig(config, true)

	inputDests := destinationsFromArray(tc.InputDestinations)
	expectedDests := destinationsFromArray(tc.ExpectedDestinations)

	out := applyAttributeConfig(cfg, tc.Key, inputDests)

	if out != expectedDests {
		t.Errorf(`name="%s"  input="%s"  expected="%s"  got="%s"`,
			tc.Testname,
			destToString(inputDests),
			destToString(expectedDests),
			destToString(out))
	}
}

func TestCrossAgentAttributes(t *testing.T) {
	var tcs []json.RawMessage

	err := crossagent.ReadJSON("attribute_configuration.json", &tcs)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tcs {
		runAttributeTestcase(t, tc)
	}
}

func TestWriteAttributeValueJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	w := jsonFieldsWriter{buf: buf}

	buf.WriteByte('{')
	writeAttributeValueJSON(&w, "a", `escape\me!`)
	writeAttributeValueJSON(&w, "a", true)
	writeAttributeValueJSON(&w, "a", false)
	writeAttributeValueJSON(&w, "a", uint8(1))
	writeAttributeValueJSON(&w, "a", uint16(2))
	writeAttributeValueJSON(&w, "a", uint32(3))
	writeAttributeValueJSON(&w, "a", uint64(4))
	writeAttributeValueJSON(&w, "a", uint(5))
	writeAttributeValueJSON(&w, "a", uintptr(6))
	writeAttributeValueJSON(&w, "a", int8(-1))
	writeAttributeValueJSON(&w, "a", int16(-2))
	writeAttributeValueJSON(&w, "a", int32(-3))
	writeAttributeValueJSON(&w, "a", int64(-4))
	writeAttributeValueJSON(&w, "a", int(-5))
	writeAttributeValueJSON(&w, "a", float32(1.5))
	writeAttributeValueJSON(&w, "a", float64(4.56))
	buf.WriteByte('}')

	expect := compactJSONString(`{
		"a":"escape\\me!",
		"a":true,
		"a":false,
		"a":1,
		"a":2,
		"a":3,
		"a":4,
		"a":5,
		"a":6,
		"a":-1,
		"a":-2,
		"a":-3,
		"a":-4,
		"a":-5,
		"a":1.5,
		"a":4.56
		}`)
	js := buf.String()
	if js != expect {
		t.Error(js, expect)
	}
}

func TestValidAttributeTypes(t *testing.T) {
	testcases := []struct {
		Input interface{}
		Valid bool
	}{
		// Valid attribute types.
		{Input: "string value", Valid: true},
		{Input: true, Valid: true},
		{Input: uint8(0), Valid: true},
		{Input: uint16(0), Valid: true},
		{Input: uint32(0), Valid: true},
		{Input: uint64(0), Valid: true},
		{Input: int8(0), Valid: true},
		{Input: int16(0), Valid: true},
		{Input: int32(0), Valid: true},
		{Input: int64(0), Valid: true},
		{Input: float32(0), Valid: true},
		{Input: float64(0), Valid: true},
		{Input: uint(0), Valid: true},
		{Input: int(0), Valid: true},
		{Input: uintptr(0), Valid: true},
		// Invalid attribute types.
		{Input: nil, Valid: false},
		{Input: struct{}{}, Valid: false},
		{Input: &struct{}{}, Valid: false},
	}

	for _, tc := range testcases {
		val, err := validateUserAttribute("key", tc.Input)
		_, invalid := err.(errInvalidAttributeType)
		if tc.Valid == invalid {
			t.Error(tc.Input, tc.Valid, val, err)
		}
	}
}

func TestUserAttributeValLength(t *testing.T) {
	cfg := createAttributeConfig(config{Config: defaultConfig()}, true)
	attrs := newAttributes(cfg)

	atLimit := strings.Repeat("a", attributeValueLengthLimit)
	tooLong := atLimit + "a"

	err := addUserAttribute(attrs, `escape\me`, tooLong, destAll)
	if err != nil {
		t.Error(err)
	}
	js := userAttributesStringJSON(attrs, destAll, nil)
	if `{"escape\\me":"`+atLimit+`"}` != js {
		t.Error(js)
	}
}

func TestUserAttributeKeyLength(t *testing.T) {
	cfg := createAttributeConfig(config{Config: defaultConfig()}, true)
	attrs := newAttributes(cfg)

	lengthyKey := strings.Repeat("a", attributeKeyLengthLimit+1)
	err := addUserAttribute(attrs, lengthyKey, 123, destAll)
	if _, ok := err.(invalidAttributeKeyErr); !ok {
		t.Error(err)
	}
	js := userAttributesStringJSON(attrs, destAll, nil)
	if `{}` != js {
		t.Error(js)
	}
}

func TestNumUserAttributesLimit(t *testing.T) {
	cfg := createAttributeConfig(config{Config: defaultConfig()}, true)
	attrs := newAttributes(cfg)

	for i := 0; i < attributeUserLimit; i++ {
		s := strconv.Itoa(i)
		err := addUserAttribute(attrs, s, s, destAll)
		if err != nil {
			t.Fatal(err)
		}
	}

	err := addUserAttribute(attrs, "cant_add_me", 123, destAll)
	if _, ok := err.(userAttributeLimitErr); !ok {
		t.Fatal(err)
	}

	js := userAttributesStringJSON(attrs, destAll, nil)
	var out map[string]string
	err = json.Unmarshal([]byte(js), &out)
	if nil != err {
		t.Fatal(err)
	}
	if len(out) != attributeUserLimit {
		t.Error(len(out))
	}
	if strings.Contains(js, "cant_add_me") {
		t.Fatal(js)
	}

	// Now test that replacement works when the limit is reached.
	err = addUserAttribute(attrs, "0", "BEEN_REPLACED", destAll)
	if nil != err {
		t.Fatal(err)
	}
	js = userAttributesStringJSON(attrs, destAll, nil)
	if !strings.Contains(js, "BEEN_REPLACED") {
		t.Fatal(js)
	}
}

func TestExtraAttributesIncluded(t *testing.T) {
	cfg := createAttributeConfig(config{Config: defaultConfig()}, true)
	attrs := newAttributes(cfg)

	err := addUserAttribute(attrs, "a", 1, destAll)
	if nil != err {
		t.Error(err)
	}
	js := userAttributesStringJSON(attrs, destAll, map[string]interface{}{"b": 2})
	if `{"b":2,"a":1}` != js {
		t.Error(js)
	}
}

func TestExtraAttributesPrecedence(t *testing.T) {
	cfg := createAttributeConfig(config{Config: defaultConfig()}, true)
	attrs := newAttributes(cfg)

	err := addUserAttribute(attrs, "a", 1, destAll)
	if nil != err {
		t.Error(err)
	}
	js := userAttributesStringJSON(attrs, destAll, map[string]interface{}{"a": 2})
	if `{"a":2}` != js {
		t.Error(js)
	}
}

func TestIncludeDisabled(t *testing.T) {
	input := config{Config: defaultConfig()}
	input.Attributes.Include = append(input.Attributes.Include, "include_me")
	cfg := createAttributeConfig(input, false)
	attrs := newAttributes(cfg)

	err := addUserAttribute(attrs, "include_me", 1, destNone)
	if nil != err {
		t.Error(err)
	}
	js := userAttributesStringJSON(attrs, destAll, nil)
	if `{}` != js {
		t.Error(js)
	}
}

func agentAttributesMap(attrs *attributes, d destinationSet) map[string]interface{} {
	buf := &bytes.Buffer{}
	agentAttributesJSON(attrs, buf, d)
	var m map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &m)
	if err != nil {
		panic(err)
	}
	return m
}

func TestRequestAgentAttributesEmptyInput(t *testing.T) {
	cfg := createAttributeConfig(config{Config: defaultConfig()}, true)
	attrs := newAttributes(cfg)
	requestAgentAttributes(attrs, "", nil, nil, "")
	got := agentAttributesMap(attrs, destAll)
	expectAttributes(t, got, map[string]interface{}{})
}

func TestRequestAgentAttributesPresent(t *testing.T) {
	req, err := http.NewRequest("GET", "http://www.newrelic.com?remove=me", nil)
	if nil != err {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "the-accept")
	req.Header.Set("Content-Type", "the-content-type")
	req.Header.Set("User-Agent", "the-agent")
	req.Header.Set("Referer", "http://www.example.com")
	req.Header.Set("Content-Length", "123")

	req.Host = "the-host"

	cfg := createAttributeConfig(config{Config: defaultConfig()}, true)

	attrs := newAttributes(cfg)
	requestAgentAttributes(attrs, req.Method, req.Header, req.URL, req.Host)
	got := agentAttributesMap(attrs, destAll)
	expectAttributes(t, got, map[string]interface{}{
		"request.headers.contentType":   "the-content-type",
		"request.headers.host":          "the-host",
		"request.headers.User-Agent":    "the-agent",
		"request.headers.userAgent":     "the-agent",
		"request.headers.referer":       "http://www.example.com",
		"request.headers.contentLength": 123,
		"request.method":                "GET",
		"request.uri":                   "http://www.newrelic.com",
		"request.headers.accept":        "the-accept",
	})
}

func BenchmarkAgentAttributes(b *testing.B) {
	cfg := createAttributeConfig(config{Config: defaultConfig()}, true)

	req, err := http.NewRequest("GET", "http://www.newrelic.com", nil)
	if nil != err {
		b.Fatal(err)
	}

	req.Header.Set("Accept", "zap")
	req.Header.Set("Content-Type", "zap")
	req.Header.Set("User-Agent", "zap")
	req.Header.Set("Referer", "http://www.newrelic.com")
	req.Header.Set("Content-Length", "123")

	req.Host = "zap"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		attrs := newAttributes(cfg)
		requestAgentAttributes(attrs, req.Method, req.Header, req.URL, req.Host)
		buf := bytes.Buffer{}
		agentAttributesJSON(attrs, &buf, destTxnTrace)
	}
}

func TestGetAgentValue(t *testing.T) {
	// Test nil safe
	var attrs *attributes
	outstr, outother := attrs.GetAgentValue(AttributeRequestURI, destTxnTrace)
	if outstr != "" || outother != nil {
		t.Error(outstr, outother)
	}

	c := config{Config: defaultConfig()}
	c.TransactionTracer.Attributes.Exclude = []string{"request.uri"}
	cfg := createAttributeConfig(c, true)
	attrs = newAttributes(cfg)
	attrs.Agent.Add(AttributeResponseContentLength, "", 123)
	attrs.Agent.Add(AttributeRequestMethod, "GET", nil)
	attrs.Agent.Add(AttributeRequestURI, "/url", nil) // disabled by configuration

	outstr, outother = attrs.GetAgentValue(AttributeResponseContentLength, destTxnTrace)
	if outstr != "" || outother != 123 {
		t.Error(outstr, outother)
	}
	outstr, outother = attrs.GetAgentValue(AttributeRequestMethod, destTxnTrace)
	if outstr != "GET" || outother != nil {
		t.Error(outstr, outother)
	}
	outstr, outother = attrs.GetAgentValue(AttributeRequestURI, destTxnTrace)
	if outstr != "" || outother != nil {
		t.Error(outstr, outother)
	}
}
