package newrelic

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	ServerlessMode    bool                `json:"serverlessmode_enabled"`
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
	configCallback := enableDistributedTracing
	if !tc.SpanEventsEnabled {
		configCallback = disableSpanEventsConfig
	}
	if tc.ServerlessMode {
		configCallback = serverlessConfig(tc)
	}

	app := testApp(func(reply *internal.ConnectReply) {
		reply.AccountID = tc.AccountID
		reply.AppID = "456"
		reply.PrimaryAppID = "456"
		reply.TrustedAccountKey = tc.TrustedAccountKey
		reply.AdaptiveSampler = internal.SampleEverything{}

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
		Expected: []string{"name", "category", "nr.entryPoint"},
	}

	// There is a single test with an error (named "exception"), so these
	// error expectations can be hard coded.
	extraErrorFields := &fieldExpect{
		Expected: []string{"parent.type", "parent.account", "parent.app",
			"parent.transportType", "error.message", "transactionName",
			"parent.transportDuration", "error.class"},
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

func enableDistributedTracing(cfg *Config) {
	cfg.CrossApplicationTracer.Enabled = false
	cfg.DistributedTracer.Enabled = true
}

func disableSpanEventsConfig(cfg *Config) {
	cfg.CrossApplicationTracer.Enabled = false
	cfg.DistributedTracer.Enabled = true
	cfg.SpanEvents.Enabled = false
}

func serverlessConfig(tc TraceContextTestCase) ConfigOption {
	return func(cfg *Config) {
		cfg.CrossApplicationTracer.Enabled = false
		cfg.DistributedTracer.Enabled = true
		cfg.ServerlessMode.AccountID = tc.AccountID
		cfg.ServerlessMode.TrustedAccountKey = tc.TrustedAccountKey
		cfg.ServerlessMode.Enabled = true
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
	payload := make(map[string]string)

	pHdr := hdrs.Get("traceparent")
	pSplit := strings.Split(pHdr, "-")
	if len(pSplit) != 4 {
		t.Error("incorrect traceparent header created ", pHdr)
		return
	}
	payload["traceparent.version"] = pSplit[0]
	payload["traceparent.trace_id"] = pSplit[1]
	payload["traceparent.parent_id"] = pSplit[2]
	payload["traceparent.trace_flags"] = pSplit[3]

	sHdr := hdrs.Get("tracestate")
	sSplit := strings.Split(sHdr, "-")
	if len(sSplit) >= 9 {
		payload["tracestate.tenant_id"] = strings.Split(sHdr, "@")[0]
		payload["tracestate.version"] = strings.Split(sSplit[0], "=")[1]
		payload["tracestate.parent_type"] = sSplit[1]
		payload["tracestate.parent_account_id"] = sSplit[2]
		payload["tracestate.parent_application_id"] = sSplit[3]
		payload["tracestate.span_id"] = sSplit[4]
		payload["tracestate.transaction_id"] = sSplit[5]
		payload["tracestate.sampled"] = sSplit[6]
		payload["tracestate.priority"] = sSplit[7]
		payload["tracestate.timestamp"] = sSplit[8]
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
		if val := payload[k]; val != exp {
			t.Errorf("expected outbound payload wrong value for key %s, expected=%s, actual=%s", k, exp, val)
		}
	}

	// Affirm that the expected values are in the actual payload.
	for _, e := range expect.Expected {
		if val := payload[e]; val == "" {
			t.Errorf("expected outbound payload missing key %s", e)
		}
	}

	// Affirm that the unexpected values are not in the actual payload.
	for _, e := range expect.Unexpected {
		if val := payload[e]; val != "" {
			t.Errorf("expected outbound payload contains key %s", e)
		}
	}

	// Affirm that not equal values are not equal in the actual payload
	for k, v := range expect.NotEqual {
		exp := fmt.Sprintf("%v", v)
		if val := payload[k]; val == exp {
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
