// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/newrelic/go-agent/internal/crossagent"
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
		"attributes":         DestAll,
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

	input := AttributeConfigInput{
		Attributes: AttributeDestinationConfig{
			Enabled: tc.Config.AttributesEnabled,
			Include: tc.Config.AttributesInclude,
			Exclude: tc.Config.AttributesExclude,
		},
		ErrorCollector: AttributeDestinationConfig{
			Enabled: tc.Config.ErrorAttributesEnabled,
			Include: tc.Config.ErrorAttributesInclude,
			Exclude: tc.Config.ErrorAttributesExclude,
		},
		TransactionEvents: AttributeDestinationConfig{
			Enabled: tc.Config.EventsAttributesEnabled,
			Include: tc.Config.EventsAttributesInclude,
			Exclude: tc.Config.EventsAttributesExclude,
		},
		BrowserMonitoring: AttributeDestinationConfig{
			Enabled: tc.Config.BrowserAttributesEnabled,
			Include: tc.Config.BrowserAttributesInclude,
			Exclude: tc.Config.BrowserAttributesExclude,
		},
		TransactionTracer: AttributeDestinationConfig{
			Enabled: tc.Config.TracerAttributesEnabled,
			Include: tc.Config.TracerAttributesInclude,
			Exclude: tc.Config.TracerAttributesExclude,
		},
	}

	cfg := CreateAttributeConfig(input, true)

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

	expect := CompactJSONString(`{
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
		val, err := ValidateUserAttribute("key", tc.Input)
		_, invalid := err.(ErrInvalidAttributeType)
		if tc.Valid == invalid {
			t.Error(tc.Input, tc.Valid, val, err)
		}
	}
}

func TestUserAttributeValLength(t *testing.T) {
	cfg := CreateAttributeConfig(sampleAttributeConfigInput, true)
	attrs := NewAttributes(cfg)

	atLimit := strings.Repeat("a", attributeValueLengthLimit)
	tooLong := atLimit + "a"

	err := AddUserAttribute(attrs, `escape\me`, tooLong, DestAll)
	if err != nil {
		t.Error(err)
	}
	js := userAttributesStringJSON(attrs, DestAll, nil)
	if `{"escape\\me":"`+atLimit+`"}` != js {
		t.Error(js)
	}
}

func TestUserAttributeKeyLength(t *testing.T) {
	cfg := CreateAttributeConfig(sampleAttributeConfigInput, true)
	attrs := NewAttributes(cfg)

	lengthyKey := strings.Repeat("a", attributeKeyLengthLimit+1)
	err := AddUserAttribute(attrs, lengthyKey, 123, DestAll)
	if _, ok := err.(invalidAttributeKeyErr); !ok {
		t.Error(err)
	}
	js := userAttributesStringJSON(attrs, DestAll, nil)
	if `{}` != js {
		t.Error(js)
	}
}

func TestNumUserAttributesLimit(t *testing.T) {
	cfg := CreateAttributeConfig(sampleAttributeConfigInput, true)
	attrs := NewAttributes(cfg)

	for i := 0; i < attributeUserLimit; i++ {
		s := strconv.Itoa(i)
		err := AddUserAttribute(attrs, s, s, DestAll)
		if err != nil {
			t.Fatal(err)
		}
	}

	err := AddUserAttribute(attrs, "cant_add_me", 123, DestAll)
	if _, ok := err.(userAttributeLimitErr); !ok {
		t.Fatal(err)
	}

	js := userAttributesStringJSON(attrs, DestAll, nil)
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
	err = AddUserAttribute(attrs, "0", "BEEN_REPLACED", DestAll)
	if nil != err {
		t.Fatal(err)
	}
	js = userAttributesStringJSON(attrs, DestAll, nil)
	if !strings.Contains(js, "BEEN_REPLACED") {
		t.Fatal(js)
	}
}

func TestExtraAttributesIncluded(t *testing.T) {
	cfg := CreateAttributeConfig(sampleAttributeConfigInput, true)
	attrs := NewAttributes(cfg)

	err := AddUserAttribute(attrs, "a", 1, DestAll)
	if nil != err {
		t.Error(err)
	}
	js := userAttributesStringJSON(attrs, DestAll, map[string]interface{}{"b": 2})
	if `{"b":2,"a":1}` != js {
		t.Error(js)
	}
}

func TestExtraAttributesPrecedence(t *testing.T) {
	cfg := CreateAttributeConfig(sampleAttributeConfigInput, true)
	attrs := NewAttributes(cfg)

	err := AddUserAttribute(attrs, "a", 1, DestAll)
	if nil != err {
		t.Error(err)
	}
	js := userAttributesStringJSON(attrs, DestAll, map[string]interface{}{"a": 2})
	if `{"a":2}` != js {
		t.Error(js)
	}
}

func TestIncludeDisabled(t *testing.T) {
	input := sampleAttributeConfigInput
	input.Attributes.Include = append(input.Attributes.Include, "include_me")
	cfg := CreateAttributeConfig(input, false)
	attrs := NewAttributes(cfg)

	err := AddUserAttribute(attrs, "include_me", 1, destNone)
	if nil != err {
		t.Error(err)
	}
	js := userAttributesStringJSON(attrs, DestAll, nil)
	if `{}` != js {
		t.Error(js)
	}
}

func agentAttributesMap(attrs *Attributes, d destinationSet) map[string]interface{} {
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
	cfg := CreateAttributeConfig(sampleAttributeConfigInput, true)
	attrs := NewAttributes(cfg)
	RequestAgentAttributes(attrs, "", nil, nil)
	got := agentAttributesMap(attrs, DestAll)
	expectAttributes(t, got, map[string]interface{}{})
}

func TestRequestAgentAttributesPresent(t *testing.T) {
	req, err := http.NewRequest("GET", "http://www.newrelic.com?remove=me", nil)
	if nil != err {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "the-accept")
	req.Header.Set("Content-Type", "the-content-type")
	req.Header.Set("Host", "the-host")
	req.Header.Set("User-Agent", "the-agent")
	req.Header.Set("Referer", "http://www.example.com")
	req.Header.Set("Content-Length", "123")

	cfg := CreateAttributeConfig(sampleAttributeConfigInput, true)

	attrs := NewAttributes(cfg)
	RequestAgentAttributes(attrs, req.Method, req.Header, req.URL)
	got := agentAttributesMap(attrs, DestAll)
	expectAttributes(t, got, map[string]interface{}{
		"request.headers.contentType":   "the-content-type",
		"request.headers.host":          "the-host",
		"request.headers.User-Agent":    "the-agent",
		"request.headers.referer":       "http://www.example.com",
		"request.headers.contentLength": 123,
		"request.method":                "GET",
		"request.uri":                   "http://www.newrelic.com",
		"request.headers.accept":        "the-accept",
	})
}

func BenchmarkAgentAttributes(b *testing.B) {
	cfg := CreateAttributeConfig(sampleAttributeConfigInput, true)

	req, err := http.NewRequest("GET", "http://www.newrelic.com", nil)
	if nil != err {
		b.Fatal(err)
	}

	req.Header.Set("Accept", "zap")
	req.Header.Set("Content-Type", "zap")
	req.Header.Set("Host", "zap")
	req.Header.Set("User-Agent", "zap")
	req.Header.Set("Referer", "http://www.newrelic.com")
	req.Header.Set("Content-Length", "123")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		attrs := NewAttributes(cfg)
		RequestAgentAttributes(attrs, req.Method, req.Header, req.URL)
		buf := bytes.Buffer{}
		agentAttributesJSON(attrs, &buf, destTxnTrace)
	}
}

func TestGetAgentValue(t *testing.T) {
	// Test nil safe
	var attrs *Attributes
	outstr, outother := attrs.GetAgentValue(attributeRequestURI, destTxnTrace)
	if outstr != "" || outother != nil {
		t.Error(outstr, outother)
	}

	c := sampleAttributeConfigInput
	c.TransactionTracer.Exclude = []string{"request.uri"}
	cfg := CreateAttributeConfig(c, true)
	attrs = NewAttributes(cfg)
	attrs.Agent.Add(attributeResponseHeadersContentLength, "", 123)
	attrs.Agent.Add(attributeRequestMethod, "GET", nil)
	attrs.Agent.Add(attributeRequestURI, "/url", nil) // disabled by configuration

	outstr, outother = attrs.GetAgentValue(attributeResponseHeadersContentLength, destTxnTrace)
	if outstr != "" || outother != 123 {
		t.Error(outstr, outother)
	}
	outstr, outother = attrs.GetAgentValue(attributeRequestMethod, destTxnTrace)
	if outstr != "GET" || outother != nil {
		t.Error(outstr, outother)
	}
	outstr, outother = attrs.GetAgentValue(attributeRequestURI, destTxnTrace)
	if outstr != "" || outother != nil {
		t.Error(outstr, outother)
	}
}
