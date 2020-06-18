// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"encoding/json"
	"testing"
	"time"
)

var (
	samplePayload = Payload{
		payloadCaller: payloadCaller{
			Type:    CallerType,
			Account: "123",
			App:     "456",
		},
		ID:        "myid",
		TracedID:  "mytrip",
		Priority:  0.12345,
		Timestamp: timestampMillis(time.Now()),
	}
)

func TestPayloadRaw(t *testing.T) {
	out, err := AcceptPayload(samplePayload)
	if err != nil || out == nil {
		t.Fatal(err, out)
	}
	if samplePayload != *out {
		t.Fatal(samplePayload, out)
	}
}

func TestPayloadNil(t *testing.T) {
	out, err := AcceptPayload(nil)
	if err != nil || out != nil {
		t.Fatal(err, out)
	}
}

func TestPayloadText(t *testing.T) {
	out, err := AcceptPayload(samplePayload.Text())
	if err != nil || out == nil {
		t.Fatal(err, out)
	}
	out.Timestamp = samplePayload.Timestamp // account for timezone differences
	if samplePayload != *out {
		t.Fatal(samplePayload, out)
	}
}

func TestPayloadTextByteSlice(t *testing.T) {
	out, err := AcceptPayload([]byte(samplePayload.Text()))
	if err != nil || out == nil {
		t.Fatal(err, out)
	}
	out.Timestamp = samplePayload.Timestamp // account for timezone differences
	if samplePayload != *out {
		t.Fatal(samplePayload, out)
	}
}

func TestPayloadHTTPSafe(t *testing.T) {
	out, err := AcceptPayload(samplePayload.HTTPSafe())
	if err != nil || nil == out {
		t.Fatal(err, out)
	}
	out.Timestamp = samplePayload.Timestamp // account for timezone differences
	if samplePayload != *out {
		t.Fatal(samplePayload, out)
	}
}

func TestPayloadHTTPSafeByteSlice(t *testing.T) {
	out, err := AcceptPayload([]byte(samplePayload.HTTPSafe()))
	if err != nil || nil == out {
		t.Fatal(err, out)
	}
	out.Timestamp = samplePayload.Timestamp // account for timezone differences
	if samplePayload != *out {
		t.Fatal(samplePayload, out)
	}
}

func TestPayloadInvalidBase64(t *testing.T) {
	out, err := AcceptPayload("======")
	if _, ok := err.(ErrPayloadParse); !ok {
		t.Fatal(err)
	}
	if nil != out {
		t.Fatal(out)
	}
}

func TestPayloadEmptyString(t *testing.T) {
	out, err := AcceptPayload("")
	if err != nil {
		t.Fatal(err)
	}
	if nil != out {
		t.Fatal(out)
	}
}

func TestPayloadUnexpectedType(t *testing.T) {
	out, err := AcceptPayload(1)
	if err != nil {
		t.Fatal(err)
	}
	if nil != out {
		t.Fatal(out)
	}
}

func TestPayloadBadVersion(t *testing.T) {
	futuristicVersion := distTraceVersion([2]int{
		currentDistTraceVersion[0] + 1,
		currentDistTraceVersion[1] + 1,
	})
	out, err := AcceptPayload(samplePayload.text(futuristicVersion))
	if _, ok := err.(ErrUnsupportedPayloadVersion); !ok {
		t.Fatal(err)
	}
	if out != nil {
		t.Fatal(out)
	}
}

func TestPayloadBadEnvelope(t *testing.T) {
	out, err := AcceptPayload("{")
	if _, ok := err.(ErrPayloadParse); !ok {
		t.Fatal(err)
	}
	if out != nil {
		t.Fatal(out)
	}
}

func TestPayloadBadPayload(t *testing.T) {
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(samplePayload.Text()), &envelope); nil != err {
		t.Fatal(err)
	}
	envelope["d"] = "123"
	payload, err := json.Marshal(envelope)
	if nil != err {
		t.Fatal(err)
	}
	out, err := AcceptPayload(payload)
	if _, ok := err.(ErrPayloadParse); !ok {
		t.Fatal(err)
	}
	if out != nil {
		t.Fatal(out)
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
		samplePayload.Text()
	}
}

func TestEmptyPayloadData(t *testing.T) {
	// does an empty payload json blob result in an invalid payload
	var payload Payload
	fixture := []byte(`{}`)

	if err := json.Unmarshal(fixture, &payload); nil != err {
		t.Log("Could not marshall fixture data into payload")
		t.Error(err)
	}

	if err := payload.IsValid(); err == nil {
		t.Log("Expected error from empty payload data")
		t.Fail()
	}
}

func TestRequiredFieldsPayloadData(t *testing.T) {
	var payload Payload
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

	if err := payload.IsValid(); err != nil {
		t.Log("Expected valid payload if ty, ac, ap, id, tr, and ti are set")
		t.Error(err)
	}
}

func TestRequiredFieldsMissingType(t *testing.T) {
	var payload Payload
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

	if err := payload.IsValid(); err == nil {
		t.Log("Expected error from missing Type (ty)")
		t.Fail()
	}
}

func TestRequiredFieldsMissingAccount(t *testing.T) {
	var payload Payload
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

	if err := payload.IsValid(); err == nil {
		t.Log("Expected error from missing Account (ac)")
		t.Fail()
	}
}

func TestRequiredFieldsMissingApp(t *testing.T) {
	var payload Payload
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

	if err := payload.IsValid(); err == nil {
		t.Log("Expected error from missing App (ap)")
		t.Fail()
	}
}

func TestRequiredFieldsMissingTimestamp(t *testing.T) {
	var payload Payload
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

	if err := payload.IsValid(); err == nil {
		t.Log("Expected error from missing Timestamp (ti)")
		t.Fail()
	}
}

func TestRequiredFieldsZeroTimestamp(t *testing.T) {
	var payload Payload
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

	if err := payload.IsValid(); err == nil {
		t.Log("Expected error from missing Timestamp (ti)")
		t.Fail()
	}
}
