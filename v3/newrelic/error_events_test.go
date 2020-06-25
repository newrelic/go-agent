// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"testing"
	"time"
)

func testErrorEventJSON(t testing.TB, e *errorEvent, expect string) {
	js, err := json.Marshal(e)
	if nil != err {
		t.Error(err)
		return
	}
	expect = compactJSONString(expect)
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
	sampleErrorData = errorData{
		Klass: "*errors.errorString",
		Msg:   "hello",
		When:  time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC),
	}
)

func TestErrorEventMarshal(t *testing.T) {
	testErrorEventJSON(t, &errorEvent{
		errorData: sampleErrorData,
		txnEvent: txnEvent{
			FinalName: "myName",
			Duration:  3 * time.Second,
			Attrs:     nil,
			BetterCAT: betterCAT{
				Enabled:  true,
				Priority: 0.5,
				TxnID:    "txn-guid-id",
				TraceID:  "trace-id",
			},
		},
	}, `[
		{
			"type":"TransactionError",
			"error.class":"*errors.errorString",
			"error.message":"hello",
			"timestamp":1417136460000,
			"transactionName":"myName",
			"duration":3,
			"guid":"txn-guid-id",
			"traceId":"trace-id",
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
	testErrorEventJSON(t, &errorEvent{
		errorData: sampleErrorData,
		txnEvent: txnEvent{
			FinalName: "myName",
			Duration:  3 * time.Second,
			Attrs:     nil,
			BetterCAT: betterCAT{
				Enabled: false,
			},
		},
	}, `[
		{
			"type":"TransactionError",
			"error.class":"*errors.errorString",
			"error.message":"hello",
			"timestamp":1417136460000,
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
	aci := config{Config: defaultConfig()}
	aci.ErrorCollector.Attributes.Exclude = append(aci.ErrorCollector.Attributes.Exclude, "zap")
	aci.ErrorCollector.Attributes.Exclude = append(aci.ErrorCollector.Attributes.Exclude, AttributeHostDisplayName)
	cfg := createAttributeConfig(aci, true)
	attr := newAttributes(cfg)
	attr.Agent.Add(AttributeHostDisplayName, "exclude me", nil)
	attr.Agent.Add(AttributeRequestMethod, "GET", nil)
	addUserAttribute(attr, "zap", 123, destAll)
	addUserAttribute(attr, "zip", 456, destAll)

	testErrorEventJSON(t, &errorEvent{
		errorData: sampleErrorData,
		txnEvent: txnEvent{
			FinalName: "myName",
			Duration:  3 * time.Second,
			Attrs:     attr,
			BetterCAT: betterCAT{
				Enabled:  true,
				Priority: 0.5,
				TxnID:    "txn-guid-id",
				TraceID:  "trace-id",
			},
		},
	}, `[
		{
			"type":"TransactionError",
			"error.class":"*errors.errorString",
			"error.message":"hello",
			"timestamp":1417136460000,
			"transactionName":"myName",
			"duration":3,
			"guid":"txn-guid-id",
			"traceId":"trace-id",
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
	aci := config{Config: defaultConfig()}
	aci.ErrorCollector.Attributes.Exclude = append(aci.ErrorCollector.Attributes.Exclude, "zap")
	aci.ErrorCollector.Attributes.Exclude = append(aci.ErrorCollector.Attributes.Exclude, AttributeHostDisplayName)
	cfg := createAttributeConfig(aci, true)
	attr := newAttributes(cfg)
	attr.Agent.Add(AttributeHostDisplayName, "exclude me", nil)
	attr.Agent.Add(AttributeRequestMethod, "GET", nil)
	addUserAttribute(attr, "zap", 123, destAll)
	addUserAttribute(attr, "zip", 456, destAll)

	testErrorEventJSON(t, &errorEvent{
		errorData: sampleErrorData,
		txnEvent: txnEvent{
			FinalName: "myName",
			Duration:  3 * time.Second,
			Attrs:     attr,
			BetterCAT: betterCAT{
				Enabled: false,
			},
		},
	}, `[
		{
			"type":"TransactionError",
			"error.class":"*errors.errorString",
			"error.message":"hello",
			"timestamp":1417136460000,
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
	e := txnEvent{
		FinalName: "myName",
		Duration:  3 * time.Second,
		Attrs:     nil,
	}

	e.BetterCAT.Enabled = true
	e.BetterCAT.TraceID = "trip-id"
	e.BetterCAT.TransportType = "HTTP"
	e.BetterCAT.Inbound = &payload{
		Type:                 "Browser",
		App:                  "caller-app",
		Account:              "caller-account",
		ID:                   "caller-id",
		TransactionID:        "caller-parent-id",
		TracedID:             "trip-id",
		TransportDuration:    2 * time.Second,
		HasNewRelicTraceInfo: true,
	}

	testErrorEventJSON(t, &errorEvent{
		errorData: sampleErrorData,
		txnEvent:  e,
	}, `[
		{
			"type":"TransactionError",
			"error.class":"*errors.errorString",
			"error.message":"hello",
			"timestamp":1417136460000,
			"transactionName":"myName",
			"duration":3,
			"parent.type": "Browser",
			"parent.app": "caller-app",
			"parent.account": "caller-account",
			"parent.transportDuration": 2,
			"parent.transportType": "HTTP",
			"guid":"",
			"traceId":"trip-id",
			"priority":0.000000,
			"sampled":false
		},
		{},
		{}
	]`)
}
