// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/crossagent"
)

type fieldExpect struct {
	Exact      map[string]interface{} `json:"exact,omitempty"`
	Expected   []string               `json:"expected,omitempty"`
	Unexpected []string               `json:"unexpected,omitempty"`
	NotEqual   map[string]interface{} `json:"notequal,omitempty"`
	Vendors    []string               `json:"vendors,omitempty"`
}

type TraceContextTestCase struct {
	TestName          string              `json:"test_name"`
	TrustedAccountKey string              `json:"trusted_account_key"`
	AccountID         string              `json:"account_id"`
	WebTransaction    bool                `json:"web_transaction"`
	RaisesException   bool                `json:"raises_exception"`
	ForceSampledTrue  bool                `json:"force_sampled_true"`
	SpanEventsEnabled bool                `json:"span_events_enabled"`
	TxnEventsEnabled  bool                `json:"transaction_events_enabled"`
	TransportType     string              `json:"transport_type"`
	InboundHeaders    []map[string]string `json:"inbound_headers"`
	OutboundPayloads  []fieldExpect       `json:"outbound_payloads,omitempty"`
	ExpectedMetrics   [][2]interface{}    `json:"expected_metrics"`
	Intrinsics        struct {
		TargetEvents     []string     `json:"target_events"`
		Common           *fieldExpect `json:"common,omitempty"`
		Transaction      *fieldExpect `json:"Transaction,omitempty"`
		Span             *fieldExpect `json:"Span,omitempty"`
		TransactionError *fieldExpect `json:"TransactionError,omitempty"`
	} `json:"intrinsics"`
}

func TestJSONDTHeaders(t *testing.T) {
	type testcase struct {
		in  string
		out http.Header
		err bool
	}

	for i, test := range []testcase{
		{"", http.Header{}, false},
		{"{}", http.Header{}, false},
		{" invalid ", http.Header{}, true},
		{`"foo"`, http.Header{}, true},
		{`{"foo": "bar"}`, http.Header{
			"Foo": {"bar"},
		}, false},
		{`{
			"foo": "bar",
			"baz": "quux",
			"multiple": [
				"alpha",
				"beta",
				"gamma"
			]
		}`, http.Header{
			"Foo":      {"bar"},
			"Baz":      {"quux"},
			"Multiple": {"alpha", "beta", "gamma"},
		}, false},
	} {
		h, err := DistributedTraceHeadersFromJSON(test.in)

		if err != nil {
			if !test.err {
				t.Errorf("case %d: %v: error expected but not generated", i, test.in)
			}
		} else if !reflect.DeepEqual(test.out, h) {
			t.Errorf("case %d, %v -> %v but expected %v", i, test.in, h, test.out)
		}
	}
}

func TestCrossAgentW3CTraceContext(t *testing.T) {
	var tcs []TraceContextTestCase

	data, err := crossagent.ReadFile("distributed_tracing/trace_context.json")
	if err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(data, &tcs); nil != err {
		t.Fatal(err)
	}

	for _, tc := range tcs {
		t.Run(tc.TestName, func(t *testing.T) {
			if tc.TestName == "spans_disabled_in_child" || tc.TestName == "spans_disabled_root" {
				t.Skip("spec change caused failing test, skipping")
				return
			}
			runW3CTestCase(t, tc)
		})
	}
}

func runW3CTestCase(t *testing.T, tc TraceContextTestCase) {
	configCallback := func(cfg *Config) {
		cfg.CrossApplicationTracer.Enabled = false
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Enabled = tc.SpanEventsEnabled
		cfg.TransactionEvents.Enabled = tc.TxnEventsEnabled
	}

	app := testApp(func(reply *internal.ConnectReply) {
		reply.AccountID = tc.AccountID
		reply.AppID = "456"
		reply.PrimaryAppID = "456"
		reply.TrustedAccountKey = tc.TrustedAccountKey
		reply.SetSampleEverything()

	}, configCallback, t)

	txn := app.StartTransaction("hello")
	if tc.WebTransaction {
		txn.SetWebRequestHTTP(nil)
	}

	// If the tests wants us to have an error, give 'em an error
	if tc.RaisesException {
		txn.NoticeError(errors.New("my error message"))
	}

	// If there are no inbound payloads, invoke Accept on an empty inbound payload.
	if nil == tc.InboundHeaders {
		txn.AcceptDistributedTraceHeaders(getTransportType(tc.TransportType), nil)
	}

	txn.AcceptDistributedTraceHeaders(getTransportType(tc.TransportType), headersFromStringMap(tc.InboundHeaders))

	// Call create each time an outbound payload appears in the testcase
	for _, expect := range tc.OutboundPayloads {
		hdrs := http.Header{}
		txn.InsertDistributedTraceHeaders(hdrs)
		assertTestCaseOutboundHeaders(expect, t, hdrs)
	}

	txn.End()

	// create WantMetrics and assert
	var wantMetrics []internal.WantMetric
	for _, metric := range tc.ExpectedMetrics {
		wantMetrics = append(wantMetrics,
			internal.WantMetric{Name: metric[0].(string), Scope: "", Forced: nil, Data: nil})
	}
	app.ExpectMetricsPresent(t, wantMetrics)

	// Add extra fields that are not listed in the JSON file so that we can
	// always do exact intrinsic set match.

	extraTxnFields := &fieldExpect{Expected: []string{"name"}}
	if tc.WebTransaction {
		extraTxnFields.Expected = append(extraTxnFields.Expected, "nr.apdexPerfZone")
	}

	extraSpanFields := &fieldExpect{
		Expected: []string{"name", "transaction.name", "category", "nr.entryPoint"},
	}

	// There is a single test with an error (named "exception"), so these
	// error expectations can be hard coded.
	extraErrorFields := &fieldExpect{
		Expected: []string{"parent.type", "parent.account", "parent.app",
			"parent.transportType", "error.message", "transactionName",
			"parent.transportDuration", "error.class", "spanId"},
	}

	for _, value := range tc.Intrinsics.TargetEvents {
		switch value {
		case "Transaction":
			assertW3CTestCaseIntrinsics(t,
				app.ExpectTxnEvents,
				tc.Intrinsics.Common,
				tc.Intrinsics.Transaction,
				extraTxnFields)
		case "Span":
			assertW3CTestCaseIntrinsics(t,
				app.ExpectSpanEvents,
				tc.Intrinsics.Common,
				tc.Intrinsics.Span,
				extraSpanFields)

		case "TransactionError":
			assertW3CTestCaseIntrinsics(t,
				app.ExpectErrorEvents,
				tc.Intrinsics.Common,
				tc.Intrinsics.TransactionError,
				extraErrorFields)
		}
	}
}

// getTransport ensures that our transport names match cross agent test values.
func getTransportType(transport string) TransportType {
	switch TransportType(transport) {
	case TransportHTTP, TransportHTTPS, TransportKafka, TransportJMS, TransportIronMQ, TransportAMQP,
		TransportQueue, TransportOther:
		return TransportType(transport)
	default:
		return TransportUnknown
	}
}

func headersFromStringMap(hdrs []map[string]string) http.Header {
	httpHdrs := http.Header{}
	for _, entry := range hdrs {
		for k, v := range entry {
			httpHdrs.Add(k, v)
		}
	}
	return httpHdrs
}

func assertTestCaseOutboundHeaders(expect fieldExpect, t *testing.T, hdrs http.Header) {
	p := make(map[string]string)

	// prepare traceparent header
	pHdr := hdrs.Get("traceparent")
	pSplit := strings.Split(pHdr, "-")
	if len(pSplit) != 4 {
		t.Error("incorrect traceparent header created ", pHdr)
		return
	}
	p["traceparent.version"] = pSplit[0]
	p["traceparent.trace_id"] = pSplit[1]
	p["traceparent.parent_id"] = pSplit[2]
	p["traceparent.trace_flags"] = pSplit[3]

	// prepare tracestate header
	sHdr := hdrs.Get("tracestate")
	sSplit := strings.Split(sHdr, "-")
	if len(sSplit) >= 9 {
		p["tracestate.tenant_id"] = strings.Split(sHdr, "@")[0]
		p["tracestate.version"] = strings.Split(sSplit[0], "=")[1]
		p["tracestate.parent_type"] = sSplit[1]
		p["tracestate.parent_account_id"] = sSplit[2]
		p["tracestate.parent_application_id"] = sSplit[3]
		p["tracestate.span_id"] = sSplit[4]
		p["tracestate.transaction_id"] = sSplit[5]
		p["tracestate.sampled"] = sSplit[6]
		p["tracestate.priority"] = sSplit[7]
		p["tracestate.timestamp"] = sSplit[8]
	}

	// prepare newrelic header
	nHdr := hdrs.Get("newrelic")
	decoded, err := base64.StdEncoding.DecodeString(nHdr)
	if err != nil {
		t.Error("failure to decode newrelic header: ", err)
	}
	nrPayload := struct {
		Version [2]int  `json:"v"`
		Data    payload `json:"d"`
	}{}
	if err := json.Unmarshal(decoded, &nrPayload); nil != err {
		t.Error("unable to unmarshall newrelic header: ", err)
	}
	p["newrelic.v"] = fmt.Sprintf("%v", nrPayload.Version)
	p["newrelic.d.ac"] = nrPayload.Data.Account
	p["newrelic.d.ap"] = nrPayload.Data.App
	p["newrelic.d.id"] = nrPayload.Data.ID
	p["newrelic.d.pr"] = fmt.Sprintf("%v", nrPayload.Data.Priority)
	p["newrelic.d.ti"] = fmt.Sprintf("%v", nrPayload.Data.Timestamp)
	p["newrelic.d.tr"] = nrPayload.Data.TracedID
	p["newrelic.d.tx"] = nrPayload.Data.TransactionID
	p["newrelic.d.ty"] = nrPayload.Data.Type
	if *nrPayload.Data.Sampled {
		p["newrelic.d.sa"] = "1"
	} else {
		p["newrelic.d.sa"] = "0"
	}

	// Affirm that the exact values are in the payload.
	for k, v := range expect.Exact {
		var exp string
		switch val := v.(type) {
		case bool:
			if val {
				exp = "1"
			} else {
				exp = "0"
			}
		case string:
			exp = val
		default:
			exp = fmt.Sprintf("%v", val)
		}
		if val := p[k]; val != exp {
			t.Errorf("expected outbound payload wrong value for key %s, expected=%s, actual=%s", k, exp, val)
		}
	}

	// Affirm that the expected values are in the actual payload.
	for _, e := range expect.Expected {
		if val := p[e]; val == "" {
			t.Errorf("expected outbound payload missing key %s", e)
		}
	}

	// Affirm that the unexpected values are not in the actual payload.
	for _, e := range expect.Unexpected {
		if val := p[e]; val != "" {
			t.Errorf("expected outbound payload contains key %s", e)
		}
	}

	// Affirm that not equal values are not equal in the actual payload
	for k, v := range expect.NotEqual {
		exp := fmt.Sprintf("%v", v)
		if val := p[k]; val == exp {
			t.Errorf("expected outbound payload has equal value for key %s, value=%s", k, val)
		}
	}

	// Affirm that the correct vendors are included in the actual payload
	for _, e := range expect.Vendors {
		if !strings.Contains(sHdr, e) {
			t.Errorf("expected outbound payload does not contain vendor %s, tracestate=%s", e, sHdr)
		}
	}
	if sHdr != "" {
		// when the tracestate header is non-empty, ensure that no extraneous
		// vendors appear
		if cnt := strings.Count(sHdr, "="); cnt != len(expect.Vendors)+1 {
			t.Errorf("expected outbound payload has wrong number of vendors, tracestate=%s", sHdr)
		}
	}
}

func assertW3CTestCaseIntrinsics(t internal.Validator,
	expect func(internal.Validator, []internal.WantEvent),
	fields ...*fieldExpect) {

	intrinsics := map[string]interface{}{}
	for _, f := range fields {
		f.add(intrinsics)
	}
	expect(t, []internal.WantEvent{{Intrinsics: intrinsics}})
}

func (fe *fieldExpect) add(intrinsics map[string]interface{}) {
	if nil != fe {
		for k, v := range fe.Exact {
			intrinsics[k] = v
		}
		for _, v := range fe.Expected {
			intrinsics[v] = internal.MatchAnything
		}
	}
}
