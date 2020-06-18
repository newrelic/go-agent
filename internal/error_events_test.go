// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"encoding/json"
	"testing"
	"time"
)

func testErrorEventJSON(t testing.TB, e *ErrorEvent, expect string) {
	js, err := json.Marshal(e)
	if nil != err {
		t.Error(err)
		return
	}
	expect = CompactJSONString(expect)
	// Type assertion to support early Go versions.
	if h, ok := t.(interface {
		Helper()
	}); ok {
		h.Helper()
	}
	actual := string(js)
	if expect != actual {
		t.Errorf("\nexpect=%s\nactual=%s\n", expect, actual)
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
			BetterCAT: BetterCAT{
				Enabled:  true,
				Priority: 0.5,
				ID:       "txn-guid-id",
			},
		},
	}, `[
		{
			"type":"TransactionError",
			"error.class":"*errors.errorString",
			"error.message":"hello",
			"timestamp":1.41713646e+09,
			"transactionName":"myName",
			"duration":3,
			"guid":"txn-guid-id",
			"traceId":"txn-guid-id",
			"priority":0.500000,
			"sampled":false
		},
		{},
		{}
	]`)

	// Many error event intrinsics are shared with txn events using sharedEventIntrinsics:  See
	// the txn event tests.
}

func TestErrorEventMarshalOldCAT(t *testing.T) {
	testErrorEventJSON(t, &ErrorEvent{
		ErrorData: sampleErrorData,
		TxnEvent: TxnEvent{
			FinalName: "myName",
			Duration:  3 * time.Second,
			Attrs:     nil,
			BetterCAT: BetterCAT{
				Enabled: false,
			},
		},
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

	// Many error event intrinsics are shared with txn events using sharedEventIntrinsics:  See
	// the txn event tests.
}

func TestErrorEventAttributes(t *testing.T) {
	aci := sampleAttributeConfigInput
	aci.ErrorCollector.Exclude = append(aci.ErrorCollector.Exclude, "zap")
	aci.ErrorCollector.Exclude = append(aci.ErrorCollector.Exclude, AttributeHostDisplayName.name())
	cfg := CreateAttributeConfig(aci, true)
	attr := NewAttributes(cfg)
	attr.Agent.Add(AttributeHostDisplayName, "exclude me", nil)
	attr.Agent.Add(attributeRequestMethod, "GET", nil)
	AddUserAttribute(attr, "zap", 123, DestAll)
	AddUserAttribute(attr, "zip", 456, DestAll)

	testErrorEventJSON(t, &ErrorEvent{
		ErrorData: sampleErrorData,
		TxnEvent: TxnEvent{
			FinalName: "myName",
			Duration:  3 * time.Second,
			Attrs:     attr,
			BetterCAT: BetterCAT{
				Enabled:  true,
				Priority: 0.5,
				ID:       "txn-guid-id",
			},
		},
	}, `[
		{
			"type":"TransactionError",
			"error.class":"*errors.errorString",
			"error.message":"hello",
			"timestamp":1.41713646e+09,
			"transactionName":"myName",
			"duration":3,
			"guid":"txn-guid-id",
			"traceId":"txn-guid-id",
 			"priority":0.500000,
 			"sampled":false
		},
		{
			"zip":456
		},
		{
			"request.method":"GET"
		}
	]`)
}

func TestErrorEventAttributesOldCAT(t *testing.T) {
	aci := sampleAttributeConfigInput
	aci.ErrorCollector.Exclude = append(aci.ErrorCollector.Exclude, "zap")
	aci.ErrorCollector.Exclude = append(aci.ErrorCollector.Exclude, AttributeHostDisplayName.name())
	cfg := CreateAttributeConfig(aci, true)
	attr := NewAttributes(cfg)
	attr.Agent.Add(AttributeHostDisplayName, "exclude me", nil)
	attr.Agent.Add(attributeRequestMethod, "GET", nil)
	AddUserAttribute(attr, "zap", 123, DestAll)
	AddUserAttribute(attr, "zip", 456, DestAll)

	testErrorEventJSON(t, &ErrorEvent{
		ErrorData: sampleErrorData,
		TxnEvent: TxnEvent{
			FinalName: "myName",
			Duration:  3 * time.Second,
			Attrs:     attr,
			BetterCAT: BetterCAT{
				Enabled: false,
			},
		},
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

func TestErrorEventMarshalWithInboundCaller(t *testing.T) {
	e := TxnEvent{
		FinalName: "myName",
		Duration:  3 * time.Second,
		Attrs:     nil,
	}

	e.BetterCAT.Enabled = true
	e.BetterCAT.Inbound = &Payload{
		payloadCaller: payloadCaller{
			TransportType: "HTTP",
			Type:          "Browser",
			App:           "caller-app",
			Account:       "caller-account",
		},
		ID:                "caller-id",
		TransactionID:     "caller-parent-id",
		TracedID:          "trip-id",
		TransportDuration: 2 * time.Second,
	}

	testErrorEventJSON(t, &ErrorEvent{
		ErrorData: sampleErrorData,
		TxnEvent:  e,
	}, `[
		{
			"type":"TransactionError",
			"error.class":"*errors.errorString",
			"error.message":"hello",
			"timestamp":1.41713646e+09,
			"transactionName":"myName",
			"duration":3,
			"parent.type": "Browser",
			"parent.app": "caller-app",
			"parent.account": "caller-account",
			"parent.transportType": "HTTP",
			"parent.transportDuration": 2,
			"guid":"",
			"traceId":"trip-id",
			"priority":0.000000,
			"sampled":false
		},
		{},
		{}
	]`)
}
