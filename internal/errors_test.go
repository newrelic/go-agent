package internal

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestErrorTraceMarshal(t *testing.T) {
	e := &TxnError{
		When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
		Stack: nil,
		Msg:   "my_msg",
		Klass: "my_class",
	}
	he := harvestErrorFromTxnError(e, "my_txn_name", "my_request_uri", nil)
	js, err := json.Marshal(he)
	if nil != err {
		t.Error(err)
	}
	expect := CompactJSONString(`
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
	aci.ErrorCollector.Exclude = append(aci.ErrorCollector.Exclude, "zap")
	aci.ErrorCollector.Exclude = append(aci.ErrorCollector.Exclude, hostDisplayName)
	cfg := CreateAttributeConfig(aci)
	attr := NewAttributes(cfg)
	attr.Agent.HostDisplayName = "exclude me"
	attr.Agent.RequestMethod = "GET"
	AddUserAttribute(attr, "zap", 123, DestAll)
	AddUserAttribute(attr, "zip", 456, DestAll)

	e := &TxnError{
		When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
		Stack: nil,
		Msg:   "my_msg",
		Klass: "my_class",
	}
	he := harvestErrorFromTxnError(e, "my_txn_name", "my_request_uri", attr)
	js, err := json.Marshal(he)
	if nil != err {
		t.Error(err)
	}
	expect := CompactJSONString(`
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
	ers := NewTxnErrors(5)

	when := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	e1 := TxnErrorFromError(errors.New("hello"))
	e2 := TxnErrorFromResponseCode(400)
	e3 := TxnErrorFromPanic(errors.New("oh no panic"))
	e4 := TxnErrorFromPanic(123)
	e5 := TxnErrorFromError(errors.New("too many errors, dropped in harvest"))
	e6 := TxnErrorFromError(errors.New("too many errors, dropped in transaction"))
	e1.When = when
	e2.When = when
	e3.When = when
	e4.When = when
	e5.When = when
	e6.When = when
	ers.Add(&e1)
	ers.Add(&e2)
	ers.Add(&e3)
	ers.Add(&e4)
	ers.Add(&e5)
	ers.Add(&e6)

	he := newHarvestErrors(4)
	MergeTxnErrors(he, ers, "txnName", "requestURI", nil)
	js, err := he.Data("agentRunID", time.Now())
	if nil != err {
		t.Error(err)
	}
	expect := CompactJSONString(`
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
