package internal

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func testTxnEventJSON(t *testing.T, e *TxnEvent, expect string) {
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
	sampleTxnEvent = TxnEvent{
		FinalName: "myName",
		ID:        "txn-id",
		Priority:  Priority{priority: 12345},
		Start:     timeFromUnixMilliseconds(1488393111000),
		Duration:  2 * time.Second,
		Zone:      ApdexNone,
		Attrs:     nil,
	}
)

func TestTxnEventMarshal(t *testing.T) {
	e := sampleTxnEvent
	testTxnEventJSON(t, &e, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1.488393111e+09,
		"nr.guid":"txn-id",
		"nr.priority":12345,
		"nr.depth":1,
		"nr.tripId":"txn-id",
		"duration":2
	},
	{},
	{}]`)
}

func TestTxnEventMarshalWithApdex(t *testing.T) {
	e := sampleTxnEvent
	e.Zone = ApdexFailing
	testTxnEventJSON(t, &e, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1.488393111e+09,
		"nr.apdexPerfZone":"F",
		"nr.guid":"txn-id",
		"nr.priority":12345,
		"nr.depth":1,
		"nr.tripId":"txn-id",
		"duration":2
	},
	{},
	{}]`)
}

func TestTxnEventMarshalWithProxy(t *testing.T) {
	e := sampleTxnEvent
	hdr := make(http.Header)
	hdr.Set("x-newrelic-timestamp-zap", "1488393108")
	e.Proxies = NewProxies(hdr, e.Start)
	testTxnEventJSON(t, &e, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1.488393111e+09,
		"nr.guid":"txn-id",
		"queueDuration":3,
		"caller.transportDuration.Zap":3,
		"nr.priority":12345,
		"nr.depth":1,
		"nr.tripId":"txn-id",
		"duration":2
	},
	{},
	{}]`)
}

func TestTxnEventMarshalWithDatastoreExternal(t *testing.T) {
	e := sampleTxnEvent
	e.DatastoreExternalTotals = DatastoreExternalTotals{
		externalCallCount:  22,
		externalDuration:   1122334 * time.Millisecond,
		datastoreCallCount: 33,
		datastoreDuration:  5566778 * time.Millisecond,
	}
	testTxnEventJSON(t, &e, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1.488393111e+09,
		"nr.guid":"txn-id",
		"nr.priority":12345,
		"nr.depth":1,
		"nr.tripId":"txn-id",
		"duration":2,
		"externalCallCount":22,
		"externalDuration":1122.334,
		"databaseCallCount":33,
		"databaseDuration":5566.778
	},
	{},
	{}]`)
}

func TestTxnEventMarshalWithInboundCaller(t *testing.T) {
	e := sampleTxnEvent
	e.Inbound = &PayloadV1{
		payloadCaller: payloadCaller{
			TransportType: "HTTP",
			Type:          "Browser",
			App:           "caller-app",
			Account:       "caller-account",
		},
		ID:                "caller-id",
		Trip:              "trip-id",
		Order:             22,
		Depth:             3,
		Host:              "caller-host",
		TransportDuration: 2 * time.Second,
	}
	testTxnEventJSON(t, &e, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1.488393111e+09,
		"nr.guid":"txn-id",
		"caller.type":"Browser",
		"caller.app":"caller-app",
		"caller.account":"caller-account",
		"caller.transportType":"HTTP",
		"caller.host":"caller-host",
		"caller.transportDuration":2,
		"nr.order":22,
		"nr.referringTransactionGuid":"caller-id",
		"nr.priority":12345,
		"nr.depth":3,
		"nr.tripId":"trip-id",
		"duration":2
	},
	{},
	{}]`)
}

func TestTxnEventMarshalWithAttributes(t *testing.T) {
	aci := sampleAttributeConfigInput
	aci.TransactionEvents.Exclude = append(aci.TransactionEvents.Exclude, "zap")
	aci.TransactionEvents.Exclude = append(aci.TransactionEvents.Exclude, hostDisplayName)
	cfg := CreateAttributeConfig(aci)
	attr := NewAttributes(cfg)
	attr.Agent.HostDisplayName = "exclude me"
	attr.Agent.RequestMethod = "GET"
	AddUserAttribute(attr, "zap", 123, DestAll)
	AddUserAttribute(attr, "zip", 456, DestAll)
	e := sampleTxnEvent
	e.Attrs = attr
	testTxnEventJSON(t, &e, `[
	{
		"type":"Transaction",
		"name":"myName",
		"timestamp":1.488393111e+09,
		"nr.guid":"txn-id",
		"nr.priority":12345,
		"nr.depth":1,
		"nr.tripId":"txn-id",
		"duration":2
	},
	{
		"zip":456
	},
	{
		"request.method":"GET"
	}]`)
}
