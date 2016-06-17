package internal

import (
	"encoding/json"
	"testing"
	"time"

	ats "github.com/newrelic/go-agent/attributes"
)

func testTxnEventJSON(t *testing.T, e *txnEvent, expect string) {
	js, err := json.Marshal(e)
	if nil != err {
		t.Error(err)
		return
	}
	expect = compactJSONString(expect)
	if string(js) != expect {
		t.Error(string(js), expect)
	}
}

func TestTxnEventMarshal(t *testing.T) {
	testTxnEventJSON(t, &txnEvent{
		Name:      "myName",
		Timestamp: time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
		Duration:  2 * time.Second,
		zone:      apdexNone,
		attrs:     nil,
	}, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1.41713646e+09,
		"duration":2
	},
	{},
	{}]`)
	testTxnEventJSON(t, &txnEvent{
		Name:      "myName",
		Timestamp: time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
		Duration:  2 * time.Second,
		zone:      apdexFailing,
		attrs:     nil,
	}, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1.41713646e+09,
		"duration":2,
		"nr.apdexPerfZone":"F"
	},
	{},
	{}]`)
	testTxnEventJSON(t, &txnEvent{
		Name:      "myName",
		Timestamp: time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
		Duration:  2 * time.Second,
		queuing:   5 * time.Second,
		zone:      apdexNone,
		attrs:     nil,
	}, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1.41713646e+09,
		"duration":2,
		"queueDuration":5
	},
	{},
	{}]`)
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

	testTxnEventJSON(t, &txnEvent{
		Name:      "myName",
		Timestamp: time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
		Duration:  2 * time.Second,
		zone:      apdexNone,
		attrs:     attr,
	}, `[
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
}
