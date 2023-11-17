// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

var (
	emptyStackTrace = make([]uintptr, 0)
)

func testExpectedJSON(t testing.TB, expect string, actual string) {
	// Type assertion to support early Go versions.
	if h, ok := t.(interface {
		Helper()
	}); ok {
		h.Helper()
	}
	compactExpect := compactJSONString(expect)
	if compactExpect != actual {
		t.Errorf("\nexpect=%s\nactual=%s\n", compactExpect, actual)
	}
}

func TestErrorNoCAT(t *testing.T) {
	he := &tracedError{
		errorData: errorData{
			When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			Stack: emptyStackTrace,
			Msg:   "my_msg",
			Klass: "my_class",
		},
		txnEvent: txnEvent{
			FinalName: "my_txn_name",
			Attrs:     nil,
			TxnID:     "txn-guid-id",
			BetterCAT: betterCAT{
				Enabled: false,
			},
			TotalTime: 2 * time.Second,
		},
	}
	js, err := json.Marshal(he)
	if nil != err {
		t.Error(err)
	}

	expect := `
	[
		1.41713646e+12,
		"my_txn_name",
		"my_msg",
		"my_class",
		{
			"agentAttributes":{},
			"userAttributes":{},
			"intrinsics":{
				"totalTime":2
			},
			"stack_trace":[]
		},
		"txn-guid-id"
	]`
	testExpectedJSON(t, expect, string(js))
}

func TestErrorTraceMarshal(t *testing.T) {
	he := &tracedError{
		errorData: errorData{
			When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			Stack: emptyStackTrace,
			Msg:   "my_msg",
			Klass: "my_class",
		},
		txnEvent: txnEvent{
			FinalName: "my_txn_name",
			Attrs:     nil,
			TxnID:     "txn-guid-id",
			BetterCAT: betterCAT{
				Enabled:  true,
				TxnID:    "txn-id",
				TraceID:  "trace-id",
				Priority: 0.5,
			},
			TotalTime: 2 * time.Second,
		},
	}
	js, err := json.Marshal(he)
	if nil != err {
		t.Error(err)
	}

	expect := `
	[
		1.41713646e+12,
		"my_txn_name",
		"my_msg",
		"my_class",
		{
			"agentAttributes":{},
			"userAttributes":{},
			"intrinsics":{
				"totalTime":2,
				"guid":"txn-id",
				"traceId":"trace-id",
				"priority":0.500000,
				"sampled":false
			},
			"stack_trace":[]
		},
		"txn-guid-id"
	]`
	testExpectedJSON(t, expect, string(js))
}

func TestErrorTraceMarshalOldCAT(t *testing.T) {
	he := &tracedError{
		errorData: errorData{
			When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			Stack: emptyStackTrace,
			Msg:   "my_msg",
			Klass: "my_class",
		},
		txnEvent: txnEvent{
			FinalName: "my_txn_name",
			Attrs:     nil,
			BetterCAT: betterCAT{
				Enabled: false,
			},
			TotalTime: 2 * time.Second,
			TxnID:     "txn-guid-id",
		},
	}
	js, err := json.Marshal(he)
	if nil != err {
		t.Error(err)
	}

	expect := `
	[
		1.41713646e+12,
		"my_txn_name",
		"my_msg",
		"my_class",
		{
			"agentAttributes":{},
			"userAttributes":{},
			"intrinsics":{
				"totalTime":2
			},
			"stack_trace":[]
		},
		"txn-guid-id"
	]`
	testExpectedJSON(t, expect, string(js))
}

func TestErrorTraceAttributes(t *testing.T) {
	aci := config{Config: defaultConfig()}
	aci.ErrorCollector.Attributes.Exclude = append(aci.ErrorCollector.Attributes.Exclude, "zap")
	aci.ErrorCollector.Attributes.Exclude = append(aci.ErrorCollector.Attributes.Exclude, AttributeHostDisplayName)
	cfg := createAttributeConfig(aci, true)
	attr := newAttributes(cfg)
	attr.Agent.Add(AttributeHostDisplayName, "exclude me", nil)
	attr.Agent.Add(AttributeRequestURI, "my_request_uri", nil)
	addUserAttribute(attr, "zap", 123, destAll)
	addUserAttribute(attr, "zip", 456, destAll)

	he := &tracedError{
		errorData: errorData{
			When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			Stack: nil,
			Msg:   "my_msg",
			Klass: "my_class",
		},
		txnEvent: txnEvent{
			FinalName: "my_txn_name",
			Attrs:     attr,
			TxnID:     "txn-id",

			BetterCAT: betterCAT{
				Enabled:  true,
				Priority: 0.5,
				TraceID:  "trace-id",
				TxnID:    "txn-id",
			},
			TotalTime: 2 * time.Second,
		},
	}
	js, err := json.Marshal(he)
	if nil != err {
		t.Error(err)
	}
	expect := `
	[
		1.41713646e+12,
		"my_txn_name",
		"my_msg",
		"my_class",
		{
			"agentAttributes":{"request.uri":"my_request_uri"},
			"userAttributes":{"zip":456},
			"intrinsics":{
				"totalTime":2,
				"guid":"txn-id",
				"traceId":"trace-id",
				"priority":0.500000,
				"sampled":false
			}
		},
		"txn-id"
	]`
	testExpectedJSON(t, expect, string(js))
}

func TestErrorTraceAttributesOldCAT(t *testing.T) {
	aci := config{Config: defaultConfig()}
	aci.ErrorCollector.Attributes.Exclude = append(aci.ErrorCollector.Attributes.Exclude, "zap")
	aci.ErrorCollector.Attributes.Exclude = append(aci.ErrorCollector.Attributes.Exclude, AttributeHostDisplayName)
	cfg := createAttributeConfig(aci, true)
	attr := newAttributes(cfg)
	attr.Agent.Add(AttributeHostDisplayName, "exclude me", nil)
	attr.Agent.Add(AttributeRequestURI, "my_request_uri", nil)
	addUserAttribute(attr, "zap", 123, destAll)
	addUserAttribute(attr, "zip", 456, destAll)

	he := &tracedError{
		errorData: errorData{
			When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			Stack: nil,
			Msg:   "my_msg",
			Klass: "my_class",
		},
		txnEvent: txnEvent{
			TxnID:     "txn-guid-id",
			FinalName: "my_txn_name",
			Attrs:     attr,
			BetterCAT: betterCAT{
				Enabled: false,
			},
			TotalTime: 2 * time.Second,
		},
	}
	js, err := json.Marshal(he)
	if nil != err {
		t.Error(err)
	}
	expect := `
	[
		1.41713646e+12,
		"my_txn_name",
		"my_msg",
		"my_class",
		{
			"agentAttributes":{"request.uri":"my_request_uri"},
			"userAttributes":{"zip":456},
			"intrinsics":{
				"totalTime":2
			}
		},
		"txn-guid-id"
	]`
	testExpectedJSON(t, expect, string(js))
}

func TestErrorsLifecycle(t *testing.T) {
	ers := newTxnErrors(5)

	when := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	ers.Add(txnErrorFromResponseCode(when, 15))
	ers.Add(txnErrorFromResponseCode(when, 400))
	ers.Add(txnErrorFromPanic(when, errors.New("oh no panic")))
	ers.Add(txnErrorFromPanic(when, 123))
	ers.Add(txnErrorFromPanic(when, 123))

	he := newHarvestErrors(4)
	mergeTxnErrors(&he, ers, txnEvent{
		FinalName: "txnName",
		Attrs:     nil,
		TxnID:     "txn-id",

		BetterCAT: betterCAT{
			Enabled:  true,
			TxnID:    "txn-id",
			TraceID:  "trace-id",
			Priority: 0.5,
		},
		TotalTime: 2 * time.Second,
	}, nil)
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
         "response code 15",
         "15",
         {
            "agentAttributes":{},
            "userAttributes":{},
            "intrinsics":{
               "totalTime":2,
			   "guid":"txn-id",
               "traceId":"trace-id",
               "priority":0.500000,
               "sampled":false
            }
         },
		 "txn-id"
      ],
      [
         1.41713646e+12,
         "txnName",
         "Bad Request",
         "400",
         {
            "agentAttributes":{},
            "userAttributes":{},
            "intrinsics":{
               "totalTime":2,
			   "guid":"txn-id",
               "traceId":"trace-id",
               "priority":0.500000,
               "sampled":false
            }
         },
		 "txn-id"
      ],
      [
         1.41713646e+12,
         "txnName",
         "oh no panic",
         "panic",
         {
            "agentAttributes":{},
            "userAttributes":{},
            "intrinsics":{
               "totalTime":2,
			   "guid":"txn-id",
               "traceId":"trace-id",
               "priority":0.500000,
               "sampled":false
            }
         },
		 "txn-id"
      ],
      [
         1.41713646e+12,
         "txnName",
         "123",
         "panic",
         {
            "agentAttributes":{},
            "userAttributes":{},
            "intrinsics":{
               "totalTime":2,
			   "guid":"txn-id",
               "traceId":"trace-id",
               "priority":0.500000,
               "sampled":false
            }
         },
		 "txn-id"
      ]
   ]
]`)
	if string(js) != expect {
		t.Error("actual: ", string(js), "expect: ", expect)
	}
}

func BenchmarkErrorsJSON(b *testing.B) {
	when := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	max := 20
	ers := newTxnErrors(max)

	for i := 0; i < max; i++ {
		ers.Add(errorData{
			When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			Msg:   "error message",
			Klass: "error class",
		})
	}

	cfg := createAttributeConfig(config{Config: defaultConfig()}, true)
	attr := newAttributes(cfg)
	attr.Agent.Add(AttributeRequestMethod, "GET", nil)
	addUserAttribute(attr, "zip", 456, destAll)

	he := newHarvestErrors(max)
	mergeTxnErrors(&he, ers, txnEvent{
		FinalName: "WebTransaction/Go/hello",
		Attrs:     attr,
	}, nil)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		js, err := he.Data("agentRundID", when)
		if nil != err || nil == js {
			b.Fatal(err, js)
		}
	}
}
