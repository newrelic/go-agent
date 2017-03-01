package internal

import (
	"encoding/json"
	"testing"
	"time"
)

func testErrorEventJSON(t *testing.T, e *ErrorEvent, expect string) {
	js, err := json.Marshal(e)
	if nil != err {
		t.Error(err)
		return
	}
	expect = CompactJSONString(expect)
	if string(js) != expect {
		t.Error(string(js), expect)
	}
}

var (
	sampleErrorData = ErrorData{
		Klass: "*errors.errorString",
		Msg:   "hello",
		When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
	}
)

func TestErrorEventMarshal(t *testing.T) {
	testErrorEventJSON(t, &ErrorEvent{
		ErrorData: sampleErrorData,
		TxnEvent: TxnEvent{
			FinalName: "myName",
			Duration:  3 * time.Second,
			Attrs:     nil,
			Priority: Priority{
				priority: 12345,
			},
			ID: "txn-guid-id",
		},
	}, `[
		{
			"type":"TransactionError",
			"error.class":"*errors.errorString",
			"error.message":"hello",
			"timestamp":1.41713646e+09,
			"transactionName":"myName",
			"nr.transactionGuid":"txn-guid-id",
			"nr.priority":12345,
			"nr.depth":1,
			"nr.tripId":"txn-guid-id",
			"duration":3
		},
		{},
		{}
	]`)

	// Many error event intrinsics are shared with txn events using sharedEventIntrinsics:  See
	// the txn event tests.
}

func TestErrorEventAttributes(t *testing.T) {
	aci := sampleAttributeConfigInput
	aci.ErrorCollector.Exclude = append(aci.ErrorCollector.Exclude, "zap")
	aci.ErrorCollector.Exclude = append(aci.ErrorCollector.Exclude, hostDisplayName)
	cfg := CreateAttributeConfig(aci)
	attr := NewAttributes(cfg)
	attr.Agent.HostDisplayName = "exclude me"
	attr.Agent.RequestMethod = "GET"
	AddUserAttribute(attr, "zap", 123, DestAll)
	AddUserAttribute(attr, "zip", 456, DestAll)

	testErrorEventJSON(t, &ErrorEvent{
		ErrorData: sampleErrorData,
		TxnEvent: TxnEvent{
			FinalName: "myName",
			Duration:  3 * time.Second,
			Attrs:     attr,
		},
	}, `[
		{
			"type":"TransactionError",
			"error.class":"*errors.errorString",
			"error.message":"hello",
			"timestamp":1.41713646e+09,
			"transactionName":"myName",
			"nr.transactionGuid":"",
			"nr.priority":0,
			"nr.depth":1,
			"nr.tripId":"",
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
