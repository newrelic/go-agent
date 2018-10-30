package internal

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
	compactExpect := CompactJSONString(expect)
	if compactExpect != actual {
		t.Errorf("\nexpect=%s\nactual=%s\n", compactExpect, actual)
	}
}

func TestErrorTraceMarshal(t *testing.T) {
	he := &tracedError{
		ErrorData: ErrorData{
			When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			Stack: emptyStackTrace,
			Msg:   "my_msg",
			Klass: "my_class",
		},
		TxnEvent: TxnEvent{
			FinalName: "my_txn_name",
			CleanURL:  "my_request_uri",
			Attrs:     nil,
			BetterCAT: BetterCAT{
				Enabled:  true,
				ID:       "txn-id",
				Priority: 0.5,
			},
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
				"guid":"txn-id",
				"traceId":"txn-id",
				"priority":0.500000,
				"sampled":false
			},
			"stack_trace":[],
			"request_uri":"my_request_uri"
		}
	]`
	testExpectedJSON(t, expect, string(js))
}

func TestErrorTraceMarshalOldCAT(t *testing.T) {
	he := &tracedError{
		ErrorData: ErrorData{
			When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			Stack: emptyStackTrace,
			Msg:   "my_msg",
			Klass: "my_class",
		},
		TxnEvent: TxnEvent{
			FinalName: "my_txn_name",
			CleanURL:  "my_request_uri",
			Attrs:     nil,
			BetterCAT: BetterCAT{
				Enabled: false,
			},
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
			"intrinsics":{},
			"stack_trace":[],
			"request_uri":"my_request_uri"
		}
	]`
	testExpectedJSON(t, expect, string(js))
}

func TestErrorTraceAttributes(t *testing.T) {
	aci := sampleAttributeConfigInput
	aci.ErrorCollector.Exclude = append(aci.ErrorCollector.Exclude, "zap")
	aci.ErrorCollector.Exclude = append(aci.ErrorCollector.Exclude, AttributeHostDisplayName.name())
	cfg := CreateAttributeConfig(aci, true)
	attr := NewAttributes(cfg)
	attr.Agent.Add(AttributeHostDisplayName, "exclude me", nil)
	attr.Agent.Add(attributeRequestMethod, "GET", nil)
	AddUserAttribute(attr, "zap", 123, DestAll)
	AddUserAttribute(attr, "zip", 456, DestAll)

	he := &tracedError{
		ErrorData: ErrorData{
			When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			Stack: nil,
			Msg:   "my_msg",
			Klass: "my_class",
		},
		TxnEvent: TxnEvent{
			FinalName: "my_txn_name",
			CleanURL:  "my_request_uri",
			Attrs:     attr,
			BetterCAT: BetterCAT{
				Enabled:  true,
				ID:       "txn-id",
				Priority: 0.5,
			},
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
			"agentAttributes":{"request.method":"GET"},
			"userAttributes":{"zip":456},
			"intrinsics":{
				"guid":"txn-id",
				"traceId":"txn-id",
				"priority":0.500000,
				"sampled":false
			},
			"request_uri":"my_request_uri"
		}
	]`
	testExpectedJSON(t, expect, string(js))
}

func TestErrorTraceAttributesOldCAT(t *testing.T) {
	aci := sampleAttributeConfigInput
	aci.ErrorCollector.Exclude = append(aci.ErrorCollector.Exclude, "zap")
	aci.ErrorCollector.Exclude = append(aci.ErrorCollector.Exclude, AttributeHostDisplayName.name())
	cfg := CreateAttributeConfig(aci, true)
	attr := NewAttributes(cfg)
	attr.Agent.Add(AttributeHostDisplayName, "exclude me", nil)
	attr.Agent.Add(attributeRequestMethod, "GET", nil)
	AddUserAttribute(attr, "zap", 123, DestAll)
	AddUserAttribute(attr, "zip", 456, DestAll)

	he := &tracedError{
		ErrorData: ErrorData{
			When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			Stack: nil,
			Msg:   "my_msg",
			Klass: "my_class",
		},
		TxnEvent: TxnEvent{
			FinalName: "my_txn_name",
			CleanURL:  "my_request_uri",
			Attrs:     attr,
			BetterCAT: BetterCAT{
				Enabled: false,
			},
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
			"agentAttributes":{"request.method":"GET"},
			"userAttributes":{"zip":456},
			"intrinsics":{},
			"request_uri":"my_request_uri"
		}
	]`
	testExpectedJSON(t, expect, string(js))
}

func TestErrorsLifecycle(t *testing.T) {
	ers := NewTxnErrors(5)

	when := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	ers.Add(TxnErrorFromResponseCode(when, 400))
	ers.Add(TxnErrorFromPanic(when, errors.New("oh no panic")))
	ers.Add(TxnErrorFromPanic(when, 123))
	ers.Add(TxnErrorFromPanic(when, 123))

	he := newHarvestErrors(3)
	MergeTxnErrors(&he, ers, TxnEvent{
		FinalName: "txnName",
		CleanURL:  "requestURI",
		Attrs:     nil,
		BetterCAT: BetterCAT{
			Enabled:  true,
			ID:       "txn-id",
			Priority: 0.5,
		},
	})
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
         "Bad Request",
         "400",
         {
            "agentAttributes":{},
            "userAttributes":{},
            "intrinsics":{
               "guid":"txn-id",
               "traceId":"txn-id",
               "priority":0.500000,
               "sampled":false
            },
            "request_uri":"requestURI"
         }
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
               "guid":"txn-id",
               "traceId":"txn-id",
               "priority":0.500000,
               "sampled":false
            },
            "request_uri":"requestURI"
         }
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
               "guid":"txn-id",
               "traceId":"txn-id",
               "priority":0.500000,
               "sampled":false
            },
            "request_uri":"requestURI"
         }
      ]
   ]
]`)
	if string(js) != expect {
		t.Error(string(js), expect)
	}
}

func BenchmarkErrorsJSON(b *testing.B) {
	when := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	max := 20
	ers := NewTxnErrors(max)

	for i := 0; i < max; i++ {
		ers.Add(ErrorData{
			When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
			Msg:   "error message",
			Klass: "error class",
		})
	}

	cfg := CreateAttributeConfig(sampleAttributeConfigInput, true)
	attr := NewAttributes(cfg)
	attr.Agent.Add(attributeRequestMethod, "GET", nil)
	AddUserAttribute(attr, "zip", 456, DestAll)

	he := newHarvestErrors(max)
	MergeTxnErrors(&he, ers, TxnEvent{
		FinalName: "WebTransaction/Go/hello",
		CleanURL:  "/url",
		Attrs:     attr,
	})

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		js, err := he.Data("agentRundID", when)
		if nil != err || nil == js {
			b.Fatal(err, js)
		}
	}
}
