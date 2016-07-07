package internal

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	ats "github.com/newrelic/go-agent/api/attributes"
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

func TestErrorsLifecycle(t *testing.T) {
	ers := newTxnErrors(5)

	when := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	e1 := txnErrorFromError(errors.New("hello"))
	e2 := txnErrorFromResponseCode(400)
	e3 := txnErrorFromPanic(errors.New("oh no panic"))
	e4 := txnErrorFromPanic(123)
	e5 := txnErrorFromError(errors.New("too many errors, dropped in harvest"))
	e6 := txnErrorFromError(errors.New("too many errors, dropped in transaction"))
	e1.when = when
	e2.when = when
	e3.when = when
	e4.when = when
	e5.when = when
	e6.when = when
	ers.Add(&e1)
	ers.Add(&e2)
	ers.Add(&e3)
	ers.Add(&e4)
	ers.Add(&e5)
	ers.Add(&e6)

	he := newHarvestErrors(4)
	mergeTxnErrors(he, ers, "txnName", "requestURI", nil)
	js, err := he.Data("agentRunID", time.Now())
	if nil != err {
		t.Error(err)
	}
	expect := compactJSONString(`
[
   "agentRunID",
   [
      [
         1.41713646e+12,
         "txnName",
         "hello",
         "*errors.errorString",
         {
            "stack_trace":null,
            "agentAttributes":{},
            "userAttributes":{},
            "intrinsics":{},
            "request_uri":"requestURI"
         }
      ],
      [
         1.41713646e+12,
         "txnName",
         "Bad Request",
         "400",
         {
            "stack_trace":null,
            "agentAttributes":{},
            "userAttributes":{},
            "intrinsics":{},
            "request_uri":"requestURI"
         }
      ],
      [
         1.41713646e+12,
         "txnName",
         "oh no panic",
         "panic",
         {
            "stack_trace":null,
            "agentAttributes":{},
            "userAttributes":{},
            "intrinsics":{},
            "request_uri":"requestURI"
         }
      ],
      [
         1.41713646e+12,
         "txnName",
         "123",
         "panic",
         {
            "stack_trace":null,
            "agentAttributes":{},
            "userAttributes":{},
            "intrinsics":{},
            "request_uri":"requestURI"
         }
      ]
   ]
]`)
	if string(js) != expect {
		t.Error(string(js))
	}
}
