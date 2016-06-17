package internal

import (
	"encoding/json"
	"testing"
	"time"

	ats "github.com/newrelic/go-agent/attributes"
)

func TestErrorTraceMarshal(t *testing.T) {
	e := &txnError{
		when:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
		stack: nil,
		msg:   "my_msg",
		klass: "my_class",
	}
	he := harvestErrorFromTxnError(e, "my_txn_name", "my_request_uri", nil)
	js, err := json.Marshal(he)
	if nil != err {
		t.Error(err)
	}
	expect := compactJSONString(`
	[
		1.41713646e+12,
		"my_txn_name",
		"my_msg",
		"my_class",
		{
			"stack_trace":null,
			"agentAttributes":{},
			"userAttributes":{},
			"intrinsics":{},
			"request_uri":"my_request_uri"
		}
	]`)
	if string(js) != expect {
		t.Error(string(js))
	}
}

func TestErrorTraceAttributes(t *testing.T) {
	aci := sampleAttributeConfigInput
	aci.errorCollector.Exclude = append(aci.errorCollector.Exclude, "zap")
	aci.errorCollector.Exclude = append(aci.errorCollector.Exclude, ats.HostDisplayName)
	cfg := createAttributeConfig(aci)
	attr := newAttributes(cfg)
	attr.agent.HostDisplayName = "exclude me"
	attr.agent.RequestMethod = "GET"
	addUserAttribute(attr, "zap", 123, destAll)
	addUserAttribute(attr, "zip", 456, destAll)

	e := &txnError{
		when:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
		stack: nil,
		msg:   "my_msg",
		klass: "my_class",
	}
	he := harvestErrorFromTxnError(e, "my_txn_name", "my_request_uri", attr)
	js, err := json.Marshal(he)
	if nil != err {
		t.Error(err)
	}
	expect := compactJSONString(`
	[
		1.41713646e+12,
		"my_txn_name",
		"my_msg",
		"my_class",
		{
			"stack_trace":null,
			"agentAttributes":{"request.method":"GET"},
			"userAttributes":{"zip":456},
			"intrinsics":{},
			"request_uri":"my_request_uri"
		}
	]`)
	if string(js) != expect {
		t.Error(string(js))
	}
}
