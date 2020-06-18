// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"
)

var (
	samplePayload = payload{
		Type:                 callerTypeApp,
		Account:              "123",
		App:                  "456",
		ID:                   "myid",
		TracedID:             "mytrip",
		Priority:             0.12345,
		Timestamp:            timestampMillis(time.Now()),
		HasNewRelicTraceInfo: true,
	}
)

func TestPayloadNil(t *testing.T) {
	var support distributedTracingSupport
	out, err := acceptPayload(nil, "123", &support)
	if err != nil || out != nil {
		t.Fatal(err, out)
	}
	if !support.isEmpty() {
		t.Error("support flags expected to be empty", support)
	}
}

func TestPayloadText(t *testing.T) {
	hdrs := http.Header{}
	hdrs.Set(DistributedTraceNewRelicHeader, samplePayload.NRText())
	var support distributedTracingSupport
	out, err := acceptPayload(hdrs, "123", &support)
	if err != nil || out == nil {
		t.Fatal(err, out)
	}
	if !support.AcceptPayloadSuccess {
		t.Error("unexpected support flags", support)
	}
	out.Timestamp = samplePayload.Timestamp // account for timezone differences
	if samplePayload != *out {
		t.Fatal(samplePayload, out)
	}
}

func TestPayloadHTTPSafe(t *testing.T) {
	hdrs := http.Header{}
	hdrs.Set(DistributedTraceNewRelicHeader, samplePayload.NRHTTPSafe())
	var support distributedTracingSupport
	out, err := acceptPayload(hdrs, "123", &support)
	if err != nil || nil == out {
		t.Fatal(err, out)
	}
	if !support.AcceptPayloadSuccess {
		t.Error("unexpected support flags", support)
	}
	out.Timestamp = samplePayload.Timestamp // account for timezone differences
	if samplePayload != *out {
		t.Fatal(samplePayload, out)
	}
}

func TestTimestampMillisMarshalUnmarshal(t *testing.T) {
	var sec int64 = 111
	var millis int64 = 222
	var micros int64 = 333
	var nsecWithMicros = 1000*1000*millis + 1000*micros
	var nsecWithoutMicros = 1000 * 1000 * millis

	input := time.Unix(sec, nsecWithMicros)
	expectOutput := time.Unix(sec, nsecWithoutMicros)

	var tm timestampMillis
	tm.Set(input)
	js, err := json.Marshal(tm)
	if nil != err {
		t.Fatal(err)
	}
	var out timestampMillis
	err = json.Unmarshal(js, &out)
	if nil != err {
		t.Fatal(err)
	}
	if out.Time() != expectOutput {
		t.Fatal(out.Time(), expectOutput)
	}
}

func BenchmarkPayloadText(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		samplePayload.NRText()
	}
}

func TestEmptyPayloadData(t *testing.T) {
	// does an empty payload json blob result in an invalid payload
	var payload payload
	fixture := []byte(`{}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.validateNewRelicData(); err == nil {
		t.Log("Expected error from empty payload data")
		t.Fail()
	}
}

func TestRequiredFieldsPayloadData(t *testing.T) {
	var payload payload
	fixture := []byte(`{
		"ty":"App",
		"ac":"123",
		"ap":"456",
		"id":"id",
		"tr":"traceID",
		"ti":1488325987402
	}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.validateNewRelicData(); err != nil {
		t.Log("Expected valid payload if ty, ac, ap, id, tr, and ti are set")
		t.Error(err)
	}
}

func TestRequiredFieldsMissingType(t *testing.T) {
	var payload payload
	fixture := []byte(`{
		"ac":"123",
		"ap":"456",
		"id":"id",
		"tr":"traceID",
		"ti":1488325987402
	}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.validateNewRelicData(); err == nil {
		t.Log("Expected error from missing Type (ty)")
		t.Fail()
	}
}

func TestRequiredFieldsMissingAccount(t *testing.T) {
	var payload payload
	fixture := []byte(`{
		"ty":"App",
		"ap":"456",
		"id":"id",
		"tr":"traceID",
		"ti":1488325987402
	}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.validateNewRelicData(); err == nil {
		t.Log("Expected error from missing Account (ac)")
		t.Fail()
	}
}

func TestRequiredFieldsMissingApp(t *testing.T) {
	var payload payload
	fixture := []byte(`{
		"ty":"App",
		"ac":"123",
		"id":"id",
		"tr":"traceID",
		"ti":1488325987402
	}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.validateNewRelicData(); err == nil {
		t.Log("Expected error from missing App (ap)")
		t.Fail()
	}
}

func TestRequiredFieldsMissingTimestamp(t *testing.T) {
	var payload payload
	fixture := []byte(`{
		"ty":"App",
		"ac":"123",
		"ap":"456",
		"tr":"traceID"
	}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.validateNewRelicData(); err == nil {
		t.Log("Expected error from missing Timestamp (ti)")
		t.Fail()
	}
}

func TestRequiredFieldsZeroTimestamp(t *testing.T) {
	var payload payload
	fixture := []byte(`{
		"ty":"App",
		"ac":"123",
		"ap":"456",
		"tr":"traceID",
		"ti":0
	}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.validateNewRelicData(); err == nil {
		t.Log("Expected error from missing Timestamp (ti)")
		t.Fail()
	}
}

func TestPayload_W3CTraceState(t *testing.T) {
	var payload payload
	fixture := []byte(`{
		"ty":"App",
		"ac":"123",
		"ap":"456",
		"tr":"traceID",
		"ti":0,
		"id":"1234567890123456",
		"tx":"6543210987654321",
		"pr":0.24689,
        "tk":"123"
	}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}
	cases := map[string]string{
		"":        "123@nr=0-0-123-456-1234567890123456-6543210987654321-0-0.24689-0",
		"a=1,b=2": "123@nr=0-0-123-456-1234567890123456-6543210987654321-0-0.24689-0,a=1,b=2",
	}
	for k, v := range cases {
		payload.NonTrustedTraceState = k
		if act := payload.W3CTraceState(); act != v {
			t.Errorf("Unexpected trace state - expected %s but got %s", v, act)
		}
	}
}

func TestProcessTraceParent(t *testing.T) {
	traceParentHdr := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"},
	}
	payload, err := processTraceParent(traceParentHdr)
	if nil != err {
		t.Errorf("Unexpected error for trace parent %s: %v", traceParentHdr, err)
	}
	traceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	if payload.TracedID != traceID {
		t.Errorf("Unexpected Trace ID in trace parent - expected %s, got %v", traceID, payload.TracedID)
	}
	spanID := "00f067aa0ba902b7"
	if payload.ID != spanID {
		t.Errorf("Unexpected Span ID in trace parent - expected %s, got %v", spanID, payload.ID)
	}
	if payload.Sampled != nil {
		t.Errorf("Expected traceparent %s sampled to be unset, but it is not", traceParentHdr)
	}
}

func TestProcessTraceParentInvalidFormat(t *testing.T) {
	cases := []string{
		"000-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"0X-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"0-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d-00f067aa0ba902b7-01",
		"0-4bf92f3577b34da6a3ce929d0e0e47366666666-00f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4MMM-00f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b711111-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba9TTT7-01",
		"00-12345678901234567890123456789012-1234567890123456-.0",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-0T",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-0",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-031",
	}
	for _, traceParent := range cases {
		traceParentHdr := http.Header{DistributedTraceW3CTraceParentHeader: []string{traceParent}}
		_, err := processTraceParent(traceParentHdr)
		if nil == err {
			t.Errorf("No error reported for trace parent %s", traceParent)
		}
	}
}

func TestProcessTraceState(t *testing.T) {
	var payload payload
	traceStateHdr := http.Header{
		DistributedTraceW3CTraceStateHeader: []string{"190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035,rojo=00f067aa0ba902b7"},
	}
	processTraceState(traceStateHdr, "190", &payload)
	if payload.TrustedAccountKey != "190" {
		t.Errorf("Wrong trusted account key: expected 190 but got %s", payload.TrustedAccountKey)
	}
	if payload.Type != "Mobile" {
		t.Errorf("Wrong payload type: expected Mobile but got %s", payload.Type)
	}
	if payload.Account != "332029" {
		t.Errorf("Wrong account: expected 332029 but got %s", payload.Account)
	}
	if payload.App != "2827902" {
		t.Errorf("Wrong app ID: expected 2827902 but got %s", payload.App)
	}
	if payload.TrustedParentID != "5f474d64b9cc9b2a" {
		t.Errorf("Wrong Trusted Parent ID: expected 5f474d64b9cc9b2a but got %s", payload.ID)
	}
	if payload.TransactionID != "7d3efb1b173fecfa" {
		t.Errorf("Wrong transaction ID: expected 7d3efb1b173fecfa but got %s", payload.TransactionID)
	}
	if nil != payload.Sampled {
		t.Errorf("Payload sampled field was set when it should not be")
	}
	if payload.Priority != 0.0 {
		t.Errorf("Wrong priority: expected 0.0 but got %f", payload.Priority)
	}
	if payload.Timestamp != timestampMillis(timeFromUnixMilliseconds(1518469636035)) {
		t.Errorf("Wrong timestamp: expected 1518469636035 but got %v", payload.Timestamp)
	}
}

func TestExtractNRTraceStateEntry(t *testing.T) {
	trustedAccountID := "12345"
	trustedNRVal := "0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277"
	trustedNR := "12345@nr=" + trustedNRVal
	nonTrustedNR := "190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035"
	cases := map[string]string{
		"rojo=00f06": "",
		// comma separator
		trustedNR + ",rojo=00f06,congo=t61":                      trustedNRVal,
		"congo=t61," + trustedNR + ",rojo=00f06":                 trustedNRVal,
		trustedNR + "," + nonTrustedNR:                           trustedNRVal,
		"rojo=00f06," + nonTrustedNR + ",congo=t61," + trustedNR: trustedNRVal,
		"rojo=00f06," + nonTrustedNR + ",congo=t61":              "",
		// comma space separator
		trustedNR + ", rojo=00f06, congo=t61":                       trustedNRVal,
		"congo=t61, " + trustedNR + ", rojo=00f06":                  trustedNRVal,
		trustedNR + ", " + nonTrustedNR:                             trustedNRVal,
		"rojo=00f06, " + nonTrustedNR + ", congo=t61, " + trustedNR: trustedNRVal,
		"rojo=00f06, " + nonTrustedNR + ", congo=t61":               "",
		// comma tab separator
		trustedNR + ",\trojo=00f06,congo=t61":                          trustedNRVal,
		"congo=t61,\t" + trustedNR + ",\trojo=00f06":                   trustedNRVal,
		trustedNR + ",\t" + nonTrustedNR:                               trustedNRVal,
		"rojo=00f06,\t" + nonTrustedNR + ",\tcongo=t61,\t" + trustedNR: trustedNRVal,
		"rojo=00f06,\t" + nonTrustedNR + ",\tcongo=t61":                "",
	}

	for test, expected := range cases {
		_, _, result := parseTraceState(test, trustedAccountID)
		if result != expected {
			t.Errorf("Expected %s but got %s", expected, result)
		}
	}
}

func TestParseTraceState(t *testing.T) {
	cases := []struct {
		// Input
		trustedAccount string
		full           string
		// Expect
		trusted          string
		expVendors       string
		expNonTrustState string
	}{
		{
			trustedAccount:   "12345",
			full:             "12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277,rojo=00f067aa0ba902b7,congo=t61rcWkgMzE",
			trusted:          "0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277",
			expVendors:       "rojo,congo",
			expNonTrustState: "rojo=00f067aa0ba902b7,congo=t61rcWkgMzE",
		},
		{
			trustedAccount:   "12345",
			full:             "congo=t61rcWkgMzE,12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277,rojo=00f067aa0ba902b7",
			trusted:          "0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277",
			expVendors:       "congo,rojo",
			expNonTrustState: "congo=t61rcWkgMzE,rojo=00f067aa0ba902b7",
		},
		{
			trustedAccount:   "12345",
			full:             "12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035",
			trusted:          "0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277",
			expVendors:       "190@nr",
			expNonTrustState: "190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035",
		},
		{
			trustedAccount:   "12345",
			full:             "atd@rojo=00f067aa0ba902b7,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035,congo=t61rcWkgMzE,12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277",
			trusted:          "0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277",
			expVendors:       "atd@rojo,190@nr,congo",
			expNonTrustState: "atd@rojo=00f067aa0ba902b7,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035,congo=t61rcWkgMzE",
		},
		{
			trustedAccount:   "12345",
			full:             "rojo=00f067aa0ba902b7,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035,fff@congo=t61rcWkgMzE",
			trusted:          "",
			expVendors:       "rojo,190@nr,fff@congo",
			expNonTrustState: "rojo=00f067aa0ba902b7,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035,fff@congo=t61rcWkgMzE",
		},
		{
			trustedAccount:   "12345",
			full:             "rojo=00f067aa0ba902b7",
			trusted:          "",
			expVendors:       "rojo",
			expNonTrustState: "rojo=00f067aa0ba902b7",
		},
		{
			trustedAccount:   "12345",
			full:             "",
			trusted:          "",
			expVendors:       "",
			expNonTrustState: "",
		},
		{
			trustedAccount:   "12345",
			full:             "abcdefghijklmnopqrstuvwxyz0123456789_-*/@a-z0-9_-*/= !\"#$%&'()*+-./0123456789:;<>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz",
			trusted:          "",
			expVendors:       "abcdefghijklmnopqrstuvwxyz0123456789_-*/@a-z0-9_-*/",
			expNonTrustState: "abcdefghijklmnopqrstuvwxyz0123456789_-*/@a-z0-9_-*/= !\"#$%&'()*+-./0123456789:;<>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz",
		},
	}

	for idx, tc := range cases {
		vendors, state, trusted := parseTraceState(tc.full, tc.trustedAccount)
		if vendors != tc.expVendors {
			t.Errorf("testcase %d: wrong value for vendors returned, expected=%s actual=%s",
				idx, tc.expVendors, vendors)
		}
		if state != tc.expNonTrustState {
			t.Errorf("testcase %d: wrong value for state returned, expected=%s actual=%s",
				idx, tc.expNonTrustState, state)
		}
		if trusted != tc.trusted {
			t.Errorf("testcase %d: wrong value for trust returned, expected=%s actual=%s",
				idx, tc.trusted, trusted)
		}
	}
}

// Our code assumes that the keys we are using are canoncial header keys, so we should make sure
// we don't accidentally change that.
func TestW3CKeysAreCannoncial(t *testing.T) {
	if DistributedTraceW3CTraceParentHeader != http.CanonicalHeaderKey(DistributedTraceW3CTraceParentHeader) {
		t.Error(DistributedTraceW3CTraceParentHeader + " is not canonical")
	}
	if DistributedTraceW3CTraceStateHeader != http.CanonicalHeaderKey(DistributedTraceW3CTraceStateHeader) {
		t.Error(DistributedTraceW3CTraceParentHeader + " is not canonical")
	}
}

func TestTransactionIDTraceStateField(t *testing.T) {
	// Test that tracestate headers transactionId accepts varying vales
	trustKey := "33"
	testcases := []struct {
		tracestate string
		expect     string
	}{
		{tracestate: "33@nr=0-0-33-5-1234567890123456--0-0.0-0", expect: ""},
		// TODO: support this use case which is called out specifically in the spec
		// {tracestate: "33@nr=0-0-33-5-1234567890123456-meatballs!-0-0.0-0", expect: "meatballs!"},
	}

	for _, tc := range testcases {
		p := &payload{}
		h := http.Header{
			DistributedTraceW3CTraceStateHeader: []string{tc.tracestate},
		}
		processTraceState(h, trustKey, p)
		if p.TransactionID != tc.expect {
			t.Errorf("wrong transactionId gathered: expect=%s actual=%s", tc.expect, p.TransactionID)
		}
	}
}

func TestSpanIDTraceStateField(t *testing.T) {
	// Test that tracestate headers spanId accepts varying vales
	trustKey := "33"
	testcases := []struct {
		tracestate string
		expect     string
	}{
		{tracestate: "33@nr=0-0-33-5--0123456789012345-0-0.0-0", expect: ""},
		// TODO: support this use case which is called out specifically in the spec
		// {tracestate: "33@nr=0-0-33-5-meatballs!-0123456789012345-0-0.0-0", expect: "meatballs!"},
	}

	for _, tc := range testcases {
		p := &payload{}
		h := http.Header{
			DistributedTraceW3CTraceStateHeader: []string{tc.tracestate},
		}
		processTraceState(h, trustKey, p)
		if p.TrustedParentID != tc.expect {
			t.Errorf("wrong transactionId gathered: expect=%s actual=%s", tc.expect, p.TrustedParentID)
		}
	}
}

func TestVersionTraceStateField(t *testing.T) {
	// Test that tracestate headers version accepts varying values
	trustKey := "33"
	testcases := []struct {
		tracestate string
		expAppID   string
	}{
		{
			tracestate: "33@nr=0-0-33-5-0123456789012345-5432109876543210-1-0.5-123",
			expAppID:   "5",
		},
		{
			// when version is too high we still try to parse what we can
			tracestate: "33@nr=1-0-33-5-0123456789012345-5432109876543210-1-0.5-123-extra-fields",
			expAppID:   "5",
		},
	}

	for _, tc := range testcases {
		p := &payload{}
		h := http.Header{
			DistributedTraceW3CTraceStateHeader: []string{tc.tracestate},
		}
		processTraceState(h, trustKey, p)
		if p.App != tc.expAppID {
			t.Errorf("wrong application id set on payload: expect=%s actual=%s", tc.expAppID, p.App)
		}
	}
}

func TestPayloadIsSampled(t *testing.T) {
	p := &payload{}
	if s := p.isSampled(); s {
		t.Error(s)
	}
	p.SetSampled(true)
	if s := p.isSampled(); !s {
		t.Error(s)
	}
	p.SetSampled(false)
	if s := p.isSampled(); s {
		t.Error(s)
	}
}

func TestTraceStateSpanTxnIDs(t *testing.T) {
	// Test that we cover this case as stated in the spec for the tracestate
	// header:
	// Conforming agents should not require any particular format of this
	// string on inbound payloads beyond receiving non-delimiter characters
	// that are valid in a tracestate header entry. meatball! is an acceptable
	// spanId or transactionId.

	hdrs := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456-meatball!-meatballs!-1-0.43771-1577830891900"},
	}
	var support distributedTracingSupport
	p, err := acceptPayload(hdrs, "123", &support)
	if err != nil {
		t.Error("failure to AcceptPayload:", err)
	}
	if !support.TraceContextAcceptSuccess {
		t.Error("unexpected support flags", support)
	}
	if p.TrustedParentID != "meatball!" {
		t.Error("wrong payload ID", p.ID)
	}
	if p.TransactionID != "meatballs!" {
		t.Error("wrong payload TransactionID", p.TransactionID)
	}
}

func TestAcceptMultipleTraceParentHeaders(t *testing.T) {
	// Test that when multiple traceparent headers are received, we discard the
	// headers all together.  From
	// https://github.com/w3c/trace-context/blob/3d02cfc15778ef850df9bc4e9d2740a4a2627fd5/test/test.py#L134
	sup := new(distributedTracingSupport)
	hdrs := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{
			"00-01234567890123456789012345678901-0123456789012345-01",
			"00-01234567890123456789012345678902-0123456789012346-01",
		},
	}
	_, err := acceptPayload(hdrs, "123", sup)
	if err == nil {
		t.Error("error should have been returned")
	}
}

func TestAcceptW3CSuccess(t *testing.T) {
	hdrs := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{
			"00-11223344556677889900aabbccddeeff-0aaabbbcccdddeee-01",
		},
		DistributedTraceW3CTraceStateHeader: []string{
			"atd@rojo=00f067aa0ba902b7,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035,congo=t61rcWkgMzE,12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277",
		},
	}
	trustedAccountKey := "12345"
	support := distributedTracingSupport{}
	p, err := acceptPayload(hdrs, trustedAccountKey, &support)
	if err != nil {
		t.Fatal(err)
	}
	truePtr := true
	expect := &payload{
		Type:                 "App",
		App:                  "41346604",
		Account:              "1349956",
		TransactionID:        "b28be285632bbc0a",
		ID:                   "0aaabbbcccdddeee",
		TracedID:             "11223344556677889900aabbccddeeff",
		Priority:             0.24689,
		Sampled:              &truePtr,
		Timestamp:            timestampMillis(timeFromUnixMilliseconds(1569367663277)),
		TransportDuration:    0,
		TrustedParentID:      "27ddd2d8890283b4",
		TracingVendors:       "atd@rojo,190@nr,congo",
		HasNewRelicTraceInfo: true,
		TrustedAccountKey:    "12345",
		NonTrustedTraceState: "atd@rojo=00f067aa0ba902b7,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035,congo=t61rcWkgMzE",
		OriginalTraceState:   "atd@rojo=00f067aa0ba902b7,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035,congo=t61rcWkgMzE,12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277",
	}
	if !reflect.DeepEqual(p, expect) {
		t.Errorf("%#v", p)
	}
}

func BenchmarkAcceptW3C(b *testing.B) {
	hdrs := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{
			"00-11223344556677889900aabbccddeeff-0aaabbbcccdddeee-01",
		},
		DistributedTraceW3CTraceStateHeader: []string{
			"atd@rojo=00f067aa0ba902b7,190@nr=0-2-332029-2827902-5f474d64b9cc9b2a-7d3efb1b173fecfa---1518469636035,congo=t61rcWkgMzE,12345@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277",
		},
	}
	trustedAccountKey := "12345"
	support := distributedTracingSupport{}

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, err := acceptPayload(hdrs, trustedAccountKey, &support)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func Test_processTraceState_invalidEntry(t *testing.T) {
	payload := payload{}
	hdrs := http.Header{
		DistributedTraceW3CTraceStateHeader: []string{"33@nr=-0-33-2827902-b4a146e3237b4df1-e8b91a159289ff74-1-1.23456-1518469636035"},
	}

	err := processTraceState(hdrs, "33", &payload)
	if err == nil || err != errInvalidNRTraceState {
		t.Errorf("expected invalidNRTraceState error but got %v", err)
	}
}
