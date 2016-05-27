package internal

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	ats "github.com/newrelic/go-sdk/attributes"
)

func TestErrorEventMarshal(t *testing.T) {
	e := txnErrorFromError(errors.New("hello"))
	e.when = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	e.stack = getStackTrace(0)
	event := createErrorEvent(&e, "myName", 3*time.Second, nil)

	js, err := json.Marshal(event)
	if nil != err {
		t.Error(err)
	}
	expect := compactJSONString(`
	[
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
	if string(js) != expect {
		t.Error(string(js))
	}
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

	e := txnErrorFromError(errors.New("hello"))
	e.when = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	e.stack = getStackTrace(0)
	event := createErrorEvent(&e, "myName", 3*time.Second, attr)

	js, err := json.Marshal(event)
	if nil != err {
		t.Error(err)
	}
	expect := compactJSONString(`[
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
	}]`)
	if string(js) != expect {
		t.Error(string(js), expect)
	}
}
