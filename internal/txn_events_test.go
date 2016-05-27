package internal

import (
	"encoding/json"
	"testing"
	"time"

	ats "github.com/newrelic/go-sdk/attributes"
)

func TestTxnEventMarshal(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)

	event := createTxnEvent(apdexNone, "myName", 2*time.Second, start, nil)
	js, err := json.Marshal(event)
	if nil != err {
		t.Error(err)
	}
	expect := compactJSONString(`[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1.41713646e+09,
		"duration":2
	},
	{},
	{}]`)
	if string(js) != expect {
		t.Error(string(js), expect)
	}

	event = createTxnEvent(apdexFailing, "myName", 2*time.Second, start, nil)
	js, err = json.Marshal(event)
	if nil != err {
		t.Error(err)
	}
	expect = compactJSONString(`[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1.41713646e+09,
		"duration":2,
		"nr.apdexPerfZone":"F"
	},
	{},
	{}]`)
	if string(js) != expect {
		t.Error(string(js), expect)
	}
}

func TestTxnEventAttributes(t *testing.T) {
	aci := sampleAttributeConfigInput
	aci.transactionEvents.Exclude = append(aci.transactionEvents.Exclude, "zap")
	aci.transactionEvents.Exclude = append(aci.transactionEvents.Exclude, ats.HostDisplayName)
	cfg := createAttributeConfig(aci)
	attr := newAttributes(cfg)
	attr.agent.HostDisplayName = "exclude me"
	attr.agent.RequestMethod = "GET"
	addUserAttribute(attr, "zap", 123, destAll)
	addUserAttribute(attr, "zip", 456, destAll)

	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	event := createTxnEvent(apdexNone, "myName", 2*time.Second, start, attr)
	js, err := json.Marshal(event)
	if nil != err {
		t.Error(err)
	}
	expect := compactJSONString(`[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1.41713646e+09,
		"duration":2
	},
	{
		"zip":456
	},
	{
		"request.method":"GET"
	}]`)
	if string(js) != expect {
		t.Error(string(js), expect)
	}
}
