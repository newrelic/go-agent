package internal

import (
	"encoding/json"
	"testing"
	"time"

	ats "github.com/newrelic/go-sdk/attributes"
)

func testErrorEventJSON(t *testing.T, e *errorEvent, expect string) {
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

func TestErrorEventMarshal(t *testing.T) {
	testErrorEventJSON(t, &errorEvent{
		klass:    "*errors.errorString",
		msg:      "hello",
		when:     time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
		txnName:  "myName",
		duration: 3 * time.Second,
		attrs:    nil,
	}, `[
		{
			"type":"TransactionError",
			"error.class":"*errors.errorString",
			"error.message":"hello",
			"timestamp":1.41713646e+09,
			"transactionName":"myName",
			"duration":3
		},
		{},
		{}
	]`)
	testErrorEventJSON(t, &errorEvent{
		klass:    "*errors.errorString",
		msg:      "hello",
		when:     time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
		txnName:  "myName",
		duration: 3 * time.Second,
		queuing:  5 * time.Second,
		attrs:    nil,
	}, `[
		{
			"type":"TransactionError",
			"error.class":"*errors.errorString",
			"error.message":"hello",
			"timestamp":1.41713646e+09,
			"transactionName":"myName",
			"duration":3,
			"queueDuration":5
		},
		{},
		{}
	]`)
}

func TestErrorEventAttributes(t *testing.T) {
	aci := sampleAttributeConfigInput
	aci.errorCollector.Exclude = append(aci.errorCollector.Exclude, "zap")
	aci.errorCollector.Exclude = append(aci.errorCollector.Exclude, ats.HostDisplayName)
	cfg := createAttributeConfig(aci)
	attr := newAttributes(cfg)
	attr.agent.HostDisplayName = "exclude me"
	attr.agent.RequestMethod = "GET"
	addUserAttribute(attr, "zap", 123, destAll)
	addUserAttribute(attr, "zip", 456, destAll)

	testErrorEventJSON(t, &errorEvent{
		klass:    "*errors.errorString",
		msg:      "hello",
		when:     time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
		txnName:  "myName",
		duration: 3 * time.Second,
		attrs:    attr,
	}, `[
		{
			"type":"TransactionError",
			"error.class":"*errors.errorString",
			"error.message":"hello",
			"timestamp":1.41713646e+09,
			"transactionName":"myName",
			"duration":3
		},
		{
			"zip":456
		},
		{
			"request.method":"GET"
		}
	]`)
}
