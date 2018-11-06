package newrelic

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/crossagent"
)

type PayloadTest struct {
	V [2]int          `json:"v"`
	D PayloadTestData `json:"d"`
}

type PayloadTestData struct {
	TY *string  `json:"ty"`
	AC *string  `json:"ac"`
	AP *string  `json:"ap"`
	ID *string  `json:"id"`
	TR *string  `json:"tr"`
	TK *string  `json:"tk"`
	PR *float32 `json:"pr"`
	SA *bool    `json:"sa"`
	TI *uint    `json:"ti"`
	TX *string  `json:"tx"`
}

func distributedTracingReplyFields(reply *internal.ConnectReply) {
	reply.AccountID = "123"
	reply.AppID = "456"
	reply.PrimaryAppID = "456"
	reply.TrustedAccounts = map[int]struct{}{
		123: {},
	}
	reply.TrustedAccountKey = "123"

	reply.AdaptiveSampler = internal.SampleEverything{}
}

func distributedTracingReplyFieldsNeedTrustKey(reply *internal.ConnectReply) {
	reply.AccountID = "123"
	reply.AppID = "456"
	reply.PrimaryAppID = "456"
	reply.TrustedAccounts = map[int]struct{}{
		123: {},
	}
	reply.TrustedAccountKey = "789"
}

func makePayload(app Application, u *url.URL) DistributedTracePayload {
	txn := app.StartTransaction("hello", nil, nil)
	return txn.CreateDistributedTracePayload()
}

func makePayloadFromTestcaseInbound(t *testing.T, tci distributedTraceTestcasePayloadTest) []byte {
	js, err := json.Marshal(tci)
	if nil != err {
		t.Error(err)
	}
	return js
}

func enableOldCATDisableBetterCat(cfg *Config) {
	cfg.CrossApplicationTracer.Enabled = true
	cfg.DistributedTracer.Enabled = false
}

func disableCAT(cfg *Config) {
	cfg.CrossApplicationTracer.Enabled = false
	cfg.DistributedTracer.Enabled = false
}

func enableBetterCAT(cfg *Config) {
	cfg.CrossApplicationTracer.Enabled = false
	cfg.DistributedTracer.Enabled = true
}

func disableSpanEvents(cfg *Config) {
	cfg.CrossApplicationTracer.Enabled = false
	cfg.DistributedTracer.Enabled = true
	cfg.SpanEvents.Enabled = false
}

func disableDistributedTracerEnableSpanEvents(cfg *Config) {
	cfg.CrossApplicationTracer.Enabled = true
	cfg.DistributedTracer.Enabled = false
	cfg.SpanEvents.Enabled = true
}

func TestPayloadConnection(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	payload := makePayload(app, nil)
	ip, ok := payload.(internal.Payload)
	if !ok {
		t.Fatal(payload)
	}
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	if nil != err {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: singleCount},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                     "OtherTransaction/Go/hello",
			"parent.type":              "App",
			"parent.account":           "123",
			"parent.app":               "456",
			"parent.transportType":     "HTTP",
			"parent.transportDuration": internal.MatchAnything,
			"parentId":                 ip.TransactionID,
			"traceId":                  ip.TransactionID,
			"parentSpanId":             ip.ID,
			"guid":                     internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"priority":                 internal.MatchAnything,
		},
	}})
}

func TestAcceptMultiple(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	payload := makePayload(app, nil)
	ip, ok := payload.(internal.Payload)
	if !ok {
		t.Fatal(payload)
	}
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	if nil != err {
		t.Error(err)
	}
	err = txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	if err != errAlreadyAccepted {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: singleCount},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Ignored/Multiple", Scope: "", Forced: true, Data: singleCount},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                     "OtherTransaction/Go/hello",
			"parent.type":              "App",
			"parent.account":           "123",
			"parent.app":               "456",
			"parent.transportType":     "HTTP",
			"parent.transportDuration": internal.MatchAnything,
			"parentId":                 ip.TransactionID,
			"traceId":                  ip.TransactionID,
			"parentSpanId":             ip.ID,
			"guid":                     internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"priority":                 internal.MatchAnything,
		},
	}})
}

func TestPayloadConnectionText(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	payload := makePayload(app, nil)
	ip, ok := payload.(internal.Payload)
	if !ok {
		t.Fatal(payload)
	}
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, payload.Text())
	if nil != err {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: singleCount},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                     "OtherTransaction/Go/hello",
			"parent.type":              "App",
			"parent.account":           "123",
			"parent.app":               "456",
			"parent.transportType":     "HTTP",
			"parent.transportDuration": internal.MatchAnything,
			"parentId":                 ip.TransactionID,
			"traceId":                  ip.TransactionID,
			"parentSpanId":             ip.ID,
			"guid":                     internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"priority":                 internal.MatchAnything,
		},
	}})
}

func validBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

func TestPayloadConnectionHTTPSafe(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	payload := makePayload(app, nil)
	ip, ok := payload.(internal.Payload)
	if !ok {
		t.Fatal(payload)
	}
	txn := app.StartTransaction("hello", nil, nil)
	p := payload.HTTPSafe()
	if !validBase64(p) {
		t.Error(p)
	}
	err := txn.AcceptDistributedTracePayload(TransportHTTP, p)
	if nil != err {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: singleCount},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                     "OtherTransaction/Go/hello",
			"parent.type":              "App",
			"parent.account":           "123",
			"parent.app":               "456",
			"parent.transportType":     "HTTP",
			"parent.transportDuration": internal.MatchAnything,
			"parentId":                 ip.TransactionID,
			"traceId":                  ip.TransactionID,
			"parentSpanId":             ip.ID,
			"guid":                     internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"priority":                 internal.MatchAnything,
		},
	}})
}

func TestPayloadConnectionNotConnected(t *testing.T) {
	app := testApp(nil, enableBetterCAT, t)
	payload := makePayload(app, nil)
	txn := app.StartTransaction("hello", nil, nil)
	if nil == payload {
		t.Fatal(payload)
	}
	if "" != payload.Text() {
		t.Error(payload.Text())
	}
	if "" != payload.HTTPSafe() {
		t.Error(payload.HTTPSafe())
	}
	err := txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	if nil != err {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, backgroundMetricsUnknownCaller)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":     "OtherTransaction/Go/hello",
			"guid":     internal.MatchAnything,
			"traceId":  internal.MatchAnything,
			"priority": internal.MatchAnything,
			"sampled":  internal.MatchAnything,
		},
	}})
}

func TestPayloadConnectionBetterCatDisabled(t *testing.T) {
	app := testApp(nil, disableCAT, t)
	payload := makePayload(app, nil)
	txn := app.StartTransaction("hello", nil, nil)
	if nil == payload {
		t.Fatal(payload)
	}
	if "" != payload.Text() {
		t.Error(payload.Text())
	}
	if "" != payload.HTTPSafe() {
		t.Error(payload.HTTPSafe())
	}
	err := txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	if err == nil {
		t.Error("missing expected error")
	}
	if errInboundPayloadDTDisabled != err {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
}

func TestPayloadTransactionsDisabled(t *testing.T) {
	cfgFn := func(cfg *Config) {
		cfg.CrossApplicationTracer.Enabled = false
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Enabled = true
		cfg.TransactionEvents.Enabled = false
	}
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello", nil, nil)

	payload := txn.CreateDistributedTracePayload()
	if nil == payload {
		t.Fatal(payload)
	}
	if "" != payload.Text() {
		t.Error(payload.Text())
	}
	if "" != payload.HTTPSafe() {
		t.Error(payload.HTTPSafe())
	}
	err := txn.End()
	if nil != err {
		t.Error(err)
	}
}

func TestPayloadConnectionEmptyString(t *testing.T) {
	app := testApp(nil, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, "")
	if nil != err {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, backgroundMetricsUnknownCaller)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":     "OtherTransaction/Go/hello",
			"guid":     internal.MatchAnything,
			"traceId":  internal.MatchAnything,
			"priority": internal.MatchAnything,
			"sampled":  internal.MatchAnything,
		},
	}})
}

func TestCreatePayloadFinished(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	txn.End()
	payload := txn.CreateDistributedTracePayload()
	if nil == payload {
		t.Fatal(payload)
	}
	if "" != payload.Text() {
		t.Error(payload.Text())
	}
	if "" != payload.HTTPSafe() {
		t.Error(payload.HTTPSafe())
	}
}

func TestAcceptPayloadFinished(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	payload := makePayload(app, nil)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.End()
	if nil != err {
		t.Error(err)
	}
	err = txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	if err != errAlreadyEnded {
		t.Fatal(err)
	}
	app.ExpectMetrics(t, backgroundMetricsUnknownCaller)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":     "OtherTransaction/Go/hello",
			"guid":     internal.MatchAnything,
			"traceId":  internal.MatchAnything,
			"priority": internal.MatchAnything,
			"sampled":  internal.MatchAnything,
		},
	}})
}

func TestPayloadTypeUnknown(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	invalidPayload := 22
	err := txn.AcceptDistributedTracePayload(TransportHTTP, invalidPayload)
	if nil != err {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, backgroundMetricsUnknownCaller)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":     "OtherTransaction/Go/hello",
			"guid":     internal.MatchAnything,
			"traceId":  internal.MatchAnything,
			"priority": internal.MatchAnything,
			"sampled":  internal.MatchAnything,
		},
	}})
}

func TestPayloadAcceptAfterCreate(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	payload := makePayload(app, nil)
	txn := app.StartTransaction("hello", nil, nil)
	txn.CreateDistributedTracePayload()
	err := txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	if errOutboundPayloadCreated != err {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: singleCount},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Ignored/CreateBeforeAccept", Scope: "", Forced: true, Data: singleCount},
	}, backgroundMetricsUnknownCaller...))
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":     "OtherTransaction/Go/hello",
			"guid":     internal.MatchAnything,
			"traceId":  internal.MatchAnything,
			"priority": internal.MatchAnything,
			"sampled":  internal.MatchAnything,
		},
	}})
}

func TestPayloadFromApplicationEmptyTransportType(t *testing.T) {
	// A user has two options when it comes to TransportType.  They can either use one of the
	// defined vars, like TransportHTTP, or create their own empty variable. The name field inside of
	// the TransportType struct is not exported outside of the package so users cannot modify its value.
	// When they make the attempt, Go reports:
	//
	// implicit assignment of unexported field 'name' in newrelic.TransportType literal.
	//
	// This test makes sure an empty TransportType resolves to "Unknown"
	var emptyTransport TransportType

	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(emptyTransport,
		`{
                              "v":[0,1],
                              "d":{
                              "ty":"App",
                              "ap":"456",
                              "ac":"123",
                              "id":"id",
                              "tr":"traceID",
                              "ti":1488325987402
                              }
		}`)
	if nil != err {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/456/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: singleCount},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                     "OtherTransaction/Go/hello",
			"parent.type":              "App",
			"parent.account":           "123",
			"parent.app":               "456",
			"parent.transportType":     "Unknown",
			"parent.transportDuration": internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"priority":                 internal.MatchAnything,
			"traceId":                  "traceID",
			"parentSpanId":             "id",
			"guid":                     internal.MatchAnything,
		},
	}})
}

func TestPayloadFutureVersion(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP,
		`{
			"v":[100,0],
			"d":{
				"ty":"App",
				"ap":"456",
				"ac":"123",
				"ti":1488325987402
			}
		}`)
	if nil == err {
		t.Error("missing expected error here")
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/Ignored/MajorVersion", Scope: "", Forced: true, Data: singleCount},
	}, backgroundMetricsUnknownCaller...))
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":     "OtherTransaction/Go/hello",
			"sampled":  internal.MatchAnything,
			"priority": internal.MatchAnything,
			"traceId":  internal.MatchAnything,
			"guid":     internal.MatchAnything,
		},
	}})
}

func TestPayloadParsingError(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP,
		`{
			"v":[0,1],
			"d":[]
		}`)
	if nil == err {
		t.Error("missing expected parsing error")
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/ParseException", Scope: "", Forced: true, Data: singleCount},
	}, backgroundMetricsUnknownCaller...))
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":     "OtherTransaction/Go/hello",
			"sampled":  internal.MatchAnything,
			"priority": internal.MatchAnything,
			"traceId":  internal.MatchAnything,
			"guid":     internal.MatchAnything,
		},
	}})
}

func TestPayloadFromFuture(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	payload := makePayload(app, nil)
	ip, ok := payload.(internal.Payload)
	if !ok {
		t.Fatal(payload)
	}
	ip.Timestamp.Set(time.Now().Add(1 * time.Hour))
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, ip)
	if nil != err {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: singleCount},
		{Name: "TransportDuration/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: singleCount},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: singleCount},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                     "OtherTransaction/Go/hello",
			"parent.type":              "App",
			"parent.account":           "123",
			"parent.app":               "456",
			"parent.transportType":     "HTTP",
			"parent.transportDuration": 0,
			"parentId":                 ip.TransactionID,
			"traceId":                  ip.TransactionID,
			"parentSpanId":             ip.ID,
			"guid":                     internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"priority":                 internal.MatchAnything,
		},
	}})
}

func TestPayloadUntrustedAccount(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	payload := makePayload(app, nil)
	ip, ok := payload.(internal.Payload)
	if !ok {
		t.Fatal(payload)
	}
	ip.Account = "12345"
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, ip)

	if _, ok := err.(internal.ErrTrustedAccountKey); !ok {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/Ignored/UntrustedAccount", Scope: "", Forced: true, Data: singleCount},
	}, backgroundMetricsUnknownCaller...))
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":     "OtherTransaction/Go/hello",
			"guid":     internal.MatchAnything,
			"traceId":  internal.MatchAnything,
			"priority": internal.MatchAnything,
			"sampled":  internal.MatchAnything,
		},
	}})
}

func TestPayloadMissingVersion(t *testing.T) {
	// ensures that a complete distributed trace payload without a version fails
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP,
		`{
			"d":{
				"ty":"App",
				"ap":"456",
				"ac":"123",
				"id":"id",
				"tr":"traceID",
				"ti":1488325987402
			}
		}`)
	if nil == err {
		t.Log("Expected error from missing Version (v)")
		t.Fail()
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
}

func TestTrustedAccountKeyPayloadHasKeyAndMatches(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	// fixture has a "tk" of 123, which matches the trusted_account_key
	// from distributedTracingReplyFields.
	p := `{
		"v":[0,1],
		"d":{
			"ty":"App",
			"ap":"456",
			"ac":"321",
			"id":"id",
			"tr":"traceID",
			"ti":1488325987402,
			"tk":"123"
		}
	}`
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, p)
	if nil != err {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
}

func TestTrustedAccountKeyPayloadHasKeyAndDoesNotMatch(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	// fixture has a "tk" of 1234, which does not match the
	// trusted_account_key from distributedTracingReplyFields.
	p := `{
		"v":[0,1],
		"d":{
			"ty":"App",
			"ap":"456",
			"ac":"321",
			"id":"id",
			"tr":"traceID",
			"ti":1488325987402,
			"tk":"1234"
		}
	}`
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, p)
	if _, ok := err.(internal.ErrTrustedAccountKey); !ok {
		t.Log("Expected ErrTrustedAccountKey from mismatched trustkeys")
		t.Fail()
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
}

func TestTrustedAccountKeyPayloadMissingKeyAndAccountIdMatches(t *testing.T) {

	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	// fixture has no trust key but its account id of 123 matches
	// trusted_account_key from distributedTracingReplyFields.
	p := `{
		"v":[0,1],
		"d":{
			"ty":"App",
			"ap":"456",
			"ac":"123",
			"id":"id",
			"tr":"traceID",
			"ti":1488325987402
		}
	}`
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, p)
	if nil != err {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}

}

func TestTrustedAccountKeyPayloadMissingKeyAndAccountIdDoesNotMatch(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	// fixture has no trust key and its account id of 1234 does not match the
	// trusted_account_key from distributedTracingReplyFields.
	p := `{
		"v":[0,1],
		"d":{
			"ty":"App",
			"ap":"456",
			"ac":"1234",
			"id":"id",
			"tr":"traceID",
			"ti":1488325987402
		}
	}`
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, p)
	if _, ok := err.(internal.ErrTrustedAccountKey); !ok {
		t.Log("Expected ErrTrustedAccountKey from mismatched trustkeys")
		t.Fail()
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
}

func TestNilPayload(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, nil)

	if nil != err {
		t.Error(err)
	}

	err = txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Ignored/Null", Scope: "", Forced: true, Data: singleCount},
	})
}

func TestNoticeErrorPayload(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	txn := app.StartTransaction("hello", nil, nil)
	txn.NoticeError(errors.New("oh no"))

	err := txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
	})
}

func TestMissingIDsForSupportabilityMetric(t *testing.T) {
	p := `{
		"v":[0,1],
		"d":{
			"ty":"App",
			"ap":"456",
			"ac":"123",
			"tr":"traceID",
			"ti":1488325987402
		}
	}`

	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, p)

	if nil == err {
		t.Log("Expected error from missing guid and transactionId")
		t.Fail()
	}

	err = txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/ParseException", Scope: "", Forced: true, Data: nil},
	})
}

func TestMissingVersionForSupportabilityMetric(t *testing.T) {
	p := `{
		"d":{
			"ty":"App",
			"ap":"456",
			"ac":"123",
			"id":"id",
			"tr":"traceID",
			"ti":1488325987402
		}
	}`

	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, p)

	if nil == err {
		t.Log("Expected error from missing version")
		t.Fail()
	}

	err = txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/ParseException", Scope: "", Forced: true, Data: nil},
	})
}

func TestMissingFieldForSupportabilityMetric(t *testing.T) {
	p := `{
		"v":[0,1],
		"d":{
			"ty":"App",
			"ap":"456",
			"id":"id",
			"tr":"traceID",
			"ti":1488325987402
		}
	}`

	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, p)

	if nil == err {
		t.Log("Expected error from missing ac field")
		t.Fail()
	}

	err = txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/ParseException", Scope: "", Forced: true, Data: nil},
	})
}

func TestParseExceptionSupportabilityMetric(t *testing.T) {
	p := `{
		"v":[0,1],
		"d":{
			"ty":"App",
			"ap":"456",
			"id":"id",
			"tr":"traceID",
			"ti":1488325987402
		}
	`

	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, p)

	if nil == err {
		t.Log("Expected error from invalid json")
		t.Fail()
	}

	err = txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/ParseException", Scope: "", Forced: true, Data: nil},
	})
}

func TestErrorsByCaller(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	txn := app.StartTransaction("hello", nil, nil)
	payload := makePayload(app, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, payload)

	if nil != err {
		t.Error(err)
	}

	txn.NoticeError(errors.New("oh no"))

	err = txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},

		{Name: "TransportDuration/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},

		{Name: "ErrorsByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "ErrorsByCaller/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
	})
}

func TestCreateDistributedTraceCatDisabled(t *testing.T) {

	// when distributed tracing is disabled, CreateDistributedTracePayload
	// should return a value that indicates an empty payload. Examples of
	// this depend on language but may be nil/null/None or an empty payload
	// object.

	app := testApp(distributedTracingReplyFields, disableCAT, t)
	txn := app.StartTransaction("hello", nil, nil)

	p := txn.CreateDistributedTracePayload()

	// empty/shim payload objects return empty strings
	if "" != p.Text() {
		t.Log("Non empty string response for .Text() method")
		t.Fail()
	}

	if "" != p.HTTPSafe() {
		t.Log("Non empty string response for .HTTPSafe() method")
		t.Fail()
	}

	err := txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
	})

}

func TestCreateDistributedTraceBetterCatDisabled(t *testing.T) {

	// when distributed tracing is disabled, CreateDistributedTracePayload
	// should return a value that indicates an empty payload. Examples of
	// this depend on language but may be nil/null/None or an empty payload
	// object.

	app := testApp(distributedTracingReplyFields, enableOldCATDisableBetterCat, t)
	txn := app.StartTransaction("hello", nil, nil)

	p := txn.CreateDistributedTracePayload()

	// empty/shim payload objects return empty strings
	if "" != p.Text() {
		t.Log("Non empty string response for .Text() method")
		t.Fail()
	}

	if "" != p.HTTPSafe() {
		t.Log("Non empty string response for .HTTPSafe() method")
		t.Fail()
	}

	err := txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
	})

}

func TestCreateDistributedTraceBetterCatEnabled(t *testing.T) {

	// When distributed tracing is enabled and the application is connected,
	// CreateDistributedTracePayload should return a valid payload object

	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)

	p := txn.CreateDistributedTracePayload()

	// empty/shim payload objects return empty strings
	if "" == p.Text() {
		t.Log("Empty string response for .Text() method")
		t.Fail()
	}

	if "" == p.HTTPSafe() {
		t.Log("Empty string response for .HTTPSafe() method")
		t.Fail()
	}

	err := txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
	})
}

func testHelperIsFieldSetWithValue(t *testing.T, p DistributedTracePayload, field interface{}, key string) {

	fieldExists := true
	fieldHasNonDefaultValue := true

	switch v := field.(type) {
	case *string:
		if nil == v {
			fieldExists = false
		} else if "" == *v {
			fieldHasNonDefaultValue = false
		}

	case *uint:
		if nil == v {
			fieldExists = false
		} else if 0 == *v {
			fieldHasNonDefaultValue = false
		}

	case *float32:
		if nil == v {
			fieldExists = false
		} else if 0 == *v {
			fieldHasNonDefaultValue = false
		}

	case *bool:
		if nil == v {
			fieldExists = false
		} else if false == *v {
			fieldHasNonDefaultValue = false
		}
	default:
		t.Log("Unhandled type passed to testHelperIsFieldSetWithValue")
		t.Fail()
	}

	if !fieldExists {
		t.Logf("Field not set: %s", key)
		t.Log(p.Text())
		t.Fail()
	}

	if !fieldHasNonDefaultValue {
		t.Logf("Field has default value: %s", key)
		t.Log(p.Text())
		t.Fail()
	}

}

func TestCreateDistributedTraceRequiredFields(t *testing.T) {

	// creates a distributed trace payload and then checks
	// to ensure the required fields are in place
	var payloadData PayloadTest
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)

	p := txn.CreateDistributedTracePayload()

	if err := json.Unmarshal([]byte(p.Text()), &payloadData); nil != err {
		t.Log("Could not marshall payload into test struct")
		t.Error(err)
	}

	testHelperIsFieldSetWithValue(t, p, payloadData.D.TY, "ty")
	testHelperIsFieldSetWithValue(t, p, payloadData.D.AC, "ac")
	testHelperIsFieldSetWithValue(t, p, payloadData.D.AP, "ap")
	testHelperIsFieldSetWithValue(t, p, payloadData.D.TR, "tr")
	testHelperIsFieldSetWithValue(t, p, payloadData.D.TI, "ti")

	err := txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
	})
}

func TestCreateDistributedTraceTrustKeyAbsent(t *testing.T) {

	// creates a distributed trace payload and then checks
	// to ensure the required fields are in place
	var payloadData PayloadTest
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)

	p := txn.CreateDistributedTracePayload()

	if err := json.Unmarshal([]byte(p.Text()), &payloadData); nil != err {
		t.Log("Could not marshall payload into test struct")
		t.Error(err)
	}

	if nil != payloadData.D.TK {
		t.Log("Did not expect trust key (tk) to be there")
		t.Log(p.Text())
		t.Fail()
	}

	err := txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
	})
}

func TestCreateDistributedTraceTrustKeyNeeded(t *testing.T) {

	// creates a distributed trace payload and then checks
	// to ensure the required fields are in place
	var payloadData PayloadTest
	app := testApp(distributedTracingReplyFieldsNeedTrustKey, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)

	p := txn.CreateDistributedTracePayload()

	if err := json.Unmarshal([]byte(p.Text()), &payloadData); nil != err {
		t.Log("Could not marshall payload into test struct")
		t.Error(err)
	}

	testHelperIsFieldSetWithValue(t, p, payloadData.D.TK, "tk")

	err := txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
	})
}

func TestCreateDistributedTraceAfterAcceptSampledTrue(t *testing.T) {
	var payloadData PayloadTest

	// simulates 1. reading distributed trace payload from non-header external storage
	// (for queues, other customer integrations); 2. Accpeting that Payload; 3. Creating
	// a new payload

	// tests that the required fields, plus priority and sampled are set
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	// fixture has a "tk" of 123, which matches the trusted_account_key
	// from distributedTracingReplyFields.
	p := `{
	"v":[0,1],
	"d":{
		"ty":"App",
		"ap":"456",
		"ac":"321",
		"id":"id",
		"tr":"traceID",
		"ti":1488325987402,
		"tk":"123",
		"sa":true
	}
}`
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, p)
	if nil != err {
		t.Error(err)
	}

	payload := txn.CreateDistributedTracePayload()

	if err := json.Unmarshal([]byte(payload.Text()), &payloadData); nil != err {
		t.Log("Could not marshall payload into test struct")
		t.Error(err)
	}

	testHelperIsFieldSetWithValue(t, payload, payloadData.D.TY, "ty")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.TY, "ty")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.AC, "ac")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.AP, "ap")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.TR, "tr")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.TI, "ti")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.PR, "pr")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.SA, "sa")

	err = txn.End()
	if nil != err {
		t.Error(err)
	}
}

func TestCreateDistributedTraceAfterAcceptSampledNotSet(t *testing.T) {
	var payloadData PayloadTest

	// simulates 1. reading distributed trace payload from non-header external storage
	// (for queues, other customer integrations); 2. Accpeting that Payload; 3. Creating
	// a new payload

	// tests that the required fields, plus priority and sampled are set.  When "sa"
	// is not set, the payload should pickup on sampled value of the transaction
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	// fixture has a "tk" of 123, which matches the trusted_account_key
	// from distributedTracingReplyFields.
	p := `{
	"v":[0,1],
	"d":{
		"ty":"App",
		"ap":"456",
		"ac":"321",
		"id":"id",
		"tr":"traceID",
		"ti":1488325987402,
		"tk":"123",
		"pr":0.54343
	}
}`
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, p)
	if nil != err {
		t.Error(err)
	}

	payload := txn.CreateDistributedTracePayload()

	if err := json.Unmarshal([]byte(`{"v":[0,1],"d":{"ty":"App","ap":"456","ac":"123","tx":"id","id":"8ac36ab049908fc","tr":"traceID","pr":0.54343,"sa":true,"ti":1532644494523}}`), &payloadData); nil != err {
		t.Log("Could not marshall payload into test struct")
		t.Error(err)
	}

	testHelperIsFieldSetWithValue(t, payload, payloadData.D.TY, "ty")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.TY, "ty")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.AC, "ac")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.AP, "ap")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.ID, "id")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.TR, "tr")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.TI, "ti")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.PR, "pr")
	testHelperIsFieldSetWithValue(t, payload, payloadData.D.SA, "sa")

	err = txn.End()
	if nil != err {
		t.Error(err)
	}
}

type distributedTraceTestcasePayloadTest PayloadTest

type distributedTraceOutboundTestcase struct {
	Exact      map[string]interface{} `json:"exact"`
	Expected   []string               `json:"expected"`
	Unexpected []string               `json:"unexpected"`
}

type distributedTraceTestcase struct {
	TestName          string                                `json:"test_name"`
	TrustedAccountKey string                                `json:"trusted_account_key"`
	AccountID         string                                `json:"account_id"`
	WebTransaction    bool                                  `json:"web_transaction"`
	RaisesException   bool                                  `json:"raises_exception"`
	ForceSampledTrue  bool                                  `json:"force_sampled_true"`
	SpanEventsEnabled bool                                  `json:"span_events_enabled"`
	MajorVersion      int                                   `json:"major_version"`
	MinorVersion      int                                   `json:"minor_version"`
	TransportType     string                                `json:"transport_type"`
	InboundPayloads   []distributedTraceTestcasePayloadTest `json:"inbound_payloads"`

	OutboundPayloads []distributedTraceOutboundTestcase `json:"outbound_payloads"`

	Intrinsics struct {
		TargetEvents []string `json:"target_events"`
		Common       struct {
			Exact      map[string]interface{} `json:"exact"`
			Expected   []string               `json:"expected"`
			Unexpected []string               `json:"unexpected"`
		} `json:"common"`

		Transaction struct {
			Exact      map[string]interface{} `json:"exact"`
			Expected   []string               `json:"expected"`
			Unexpected []string               `json:"unexpected"`
		} `json:"Transaction"`

		Span struct {
			Exact      map[string]interface{} `json:"exact"`
			Expected   []string               `json:"expected"`
			Unexpected []string               `json:"unexpected"`
		}

		TransactionError struct {
			Exact      map[string]interface{} `json:"exact"`
			Expected   []string               `json:"expected"`
			Unexpected []string               `json:"unexpected"`
		}
	} `json:"intrinsics"`

	ExpectedMetrics [][2]interface{} `json:"expected_metrics"`
}

func TestDistributedTraceCrossAgentJsonParse(t *testing.T) {
	// test cases are complicated enough that we want/need this
	// test to ensure we're parsing the test case JSON correctly,
	// and don't have any of go's slient failures if we've
	// typo'd a json field name or something
	var tc distributedTraceTestcase

	// this is not a legitimate fixture, we're only use it to
	// test that our parsing code parses everything as expected
	input := []byte(`{
				"test_name": "fixture_test",
				"trusted_account_key": "33",
				"account_id": "33",
				"web_transaction": true,
				"raises_exception": true,
				"force_sampled_true": true,
				"span_events_enabled": true,
				"major_version": 1,
				"minor_version": 1,
				"transport_type": "HTTP",
				"inbound_payloads": [
						{
								"v": [2, 3],
								"d": {
										"ac": "33",
										"ap": "2827902",
										"id": "7d3efb1b173fecfa",
										"tx": "e8b91a159289ff74",
										"pr": 1.234567,
										"sa": true,
										"ti": 1518469636035,
										"tr": "d6b4ba0c3a712ca",
										"ty": "App"
								}
						}
				],
				"intrinsics": {
						"target_events": ["Transaction", "Span"],
						"common":{
								"exact": {
										"traceId": "d6b4ba0c3a712ca",
										"priority": 1.234567,
										"sampled": true
								},
								"expected": ["guid"],
								"unexpected": ["grandparentId", "cross_process_id"]
						},
						"Transaction": {
								"exact": {
										"parent.type": "App",
										"parent.app": "2827902",
										"parent.account": "33",
										"parent.transportType": "HTTP",
										"parentId": "e8b91a159289ff74",
										"parentSpanId": "parentSpanId"
								},
								"expected": ["parent.transportDuration"]
						},
						"Span": {
								"exact": {
										"parentId": "7d3efb1b173fecfa"
								},
								"expected": ["transactionId"],
								"unexpected": ["parent.app", "parent.account"]
						}
				},
				"expected_metrics": [
						["DurationByCaller/App/33/2827902/HTTP/all", 1],
						["DurationByCaller/App/33/2827902/HTTP/allWeb", 7]
				]}`)

	err := json.Unmarshal(input, &tc)

	if nil != err {
		t.Fatal(err)
	}

	if "fixture_test" != tc.TestName {
		t.Log("Unexpected value for tc.TestName")
		t.Fail()
	}

	if "33" != tc.TrustedAccountKey {
		t.Log("Unexpected value for tc.TrustedAccountKey")
		t.Fail()
	}

	if "33" != tc.AccountID {
		t.Log("Unexpected value for tc.AccountID")
		t.Fail()
	}

	if true != tc.WebTransaction {
		t.Log("Unexpected value for tc.WebTransaction")
		t.Fail()
	}

	if true != tc.RaisesException {
		t.Log("Unexpected value for tc.RaisesException")
		t.Fail()
	}

	if true != tc.ForceSampledTrue {
		t.Log("Unexpected value for tc.ForceSampledTrue")
		t.Fail()
	}

	if true != tc.SpanEventsEnabled {
		t.Log("Unexpected value for tc.SpanEventsEnabled")
		t.Fail()
	}

	if 1 != tc.MajorVersion {
		t.Log("Unexpected value for tc.MajorVersion")
		t.Fail()
	}

	if 1 != tc.MinorVersion {
		t.Log("Unexpected value for tc.MinorVersion")
		t.Fail()
	}

	if "HTTP" != tc.TransportType {
		t.Log("Unexpected value for tc.TransportType")
		t.Fail()
	}

	if 1 != len(tc.InboundPayloads) {
		t.Log("Unexpected value for len(tc.InboundPayloads)")
		t.Fail()
	}

	if 2 != tc.InboundPayloads[0].V[0] {
		t.Log("Unexpected value for tc.InboundPayloads[0].V[0]")
		t.Fail()
	}

	if 3 != tc.InboundPayloads[0].V[1] {
		t.Log("Unexpected value for tc.InboundPayloads[0].V[1]")
		t.Fail()
	}

	if "33" != *tc.InboundPayloads[0].D.AC {
		t.Log("Unexpected value for *tc.InboundPayloads[0].D.AC")
		t.Fail()
	}

	if "2827902" != *tc.InboundPayloads[0].D.AP {
		t.Log("Unexpected value for *tc.InboundPayloads[0].D.AP")
		t.Fail()
	}

	if "7d3efb1b173fecfa" != *tc.InboundPayloads[0].D.ID {
		t.Log("Unexpected value for *tc.InboundPayloads[0].D.ID")
		t.Fail()
	}

	if "e8b91a159289ff74" != *tc.InboundPayloads[0].D.TX {
		t.Log("Unexpected value for *tc.InboundPayloads[0].D.TX")
		t.Fail()
	}

	if 1.234567 != *tc.InboundPayloads[0].D.PR {
		t.Log("Unexpected value for *tc.InboundPayloads[0].D.PR")
		t.Fail()
	}

	if true != *tc.InboundPayloads[0].D.SA {
		t.Log("Unexpected value for *tc.InboundPayloads[0].D.SA")
		t.Fail()
	}

	if 1518469636035 != *tc.InboundPayloads[0].D.TI {
		t.Log("Unexpected value for *tc.InboundPayloads[0].D.TI")
		t.Fail()
	}

	if "d6b4ba0c3a712ca" != *tc.InboundPayloads[0].D.TR {
		t.Log("Unexpected value for *tc.InboundPayloads[0].D.TR")
		t.Fail()
	}

	if "App" != *tc.InboundPayloads[0].D.TY {
		t.Log("Unexpected value for *tc.InboundPayloads[0].D.TY")
		t.Fail()
	}

	if "Transaction" != tc.Intrinsics.TargetEvents[0] {
		t.Log("Unexpected value for tc.Intrinsics.TargetEvents[0]")
		t.Fail()
	}

	if "Span" != tc.Intrinsics.TargetEvents[1] {
		t.Log("Unexpected value for tc.Intrinsics.TargetEvents[1]")
		t.Fail()
	}

	if len(tc.Intrinsics.Common.Exact) < 1 {
		t.Log("No common exact intrinsics found.")
		t.Fail()
	}

	if "guid" != tc.Intrinsics.Common.Expected[0] {
		t.Log("Unexpected value for tc.Intrinsics.Common.Expected[0]")
		t.Fail()
	}

	if "grandparentId" != tc.Intrinsics.Common.Unexpected[0] {
		t.Log("Unexpected value for tc.Intrinsics.Common.Unexpected[0]")
		t.Fail()
	}

	if "cross_process_id" != tc.Intrinsics.Common.Unexpected[1] {
		t.Log("Unexpected value for tc.Intrinsics.Common.Unexpected[1]")
		t.Fail()
	}

	if len(tc.Intrinsics.Transaction.Exact) < 1 {
		t.Log("No transaction exact intrinsics found.")
		t.Fail()
	}

	if "parent.transportDuration" != tc.Intrinsics.Transaction.Expected[0] {
		t.Log("Unexpected value for tc.Intrinsics.Transaction.Expected[0]")
		t.Fail()
	}

	if len(tc.Intrinsics.Span.Exact) < 1 {
		t.Log("No span exact intrinsics found.")
		t.Fail()
	}

	if "transactionId" != tc.Intrinsics.Span.Expected[0] {
		t.Log("Unexpected value for tc.Intrinsics.Span.Expected[0]")
		t.Fail()
	}

	if "parent.app" != tc.Intrinsics.Span.Unexpected[0] {
		t.Log("Unexpected value for tc.Intrinsics.Span.Unexpected[0]")
		t.Fail()
	}

	if "parent.account" != tc.Intrinsics.Span.Unexpected[1] {
		t.Log("Unexpected value for tc.Intrinsics.Span.Unexpected[1]")
		t.Fail()
	}

	if "DurationByCaller/App/33/2827902/HTTP/all" != tc.ExpectedMetrics[0][0].(string) {
		t.Log("Unexpected value for tc.ExpectedMetrics[0][0].(string)")
		t.Fail()
	}

	if 1 != tc.ExpectedMetrics[0][1].(float64) {
		t.Log("Unexpected value for tc.ExpectedMetrics[0][1].(float64)")
		t.Fail()
	}

	if "DurationByCaller/App/33/2827902/HTTP/allWeb" != tc.ExpectedMetrics[1][0].(string) {
		t.Log("Unexpected value for tc.ExpectedMetrics[1][0].(string)")
		t.Fail()
	}

	if 7 != tc.ExpectedMetrics[1][1].(float64) {
		t.Log("Unexpected value for tc.ExpectedMetrics[1][1].(float64)")
		t.Fail()
	}
}

func runDistributedTraceCrossAgentTestcase(t *testing.T, tc distributedTraceTestcase, extraAsserts func(expectApp, *testing.T, distributedTraceTestcase)) {
	t.Logf("Starting Test: %s", tc.TestName)
	configCallback := enableBetterCAT
	if false == tc.SpanEventsEnabled {
		configCallback = disableSpanEvents
	}

	app := testApp(func(reply *internal.ConnectReply) {
		reply.AccountID = tc.AccountID
		reply.AppID = "456"
		reply.PrimaryAppID = "456"
		reply.TrustedAccountKey = tc.TrustedAccountKey

		// if cross agent tests ever include logic for sampling
		// we'll need to revisit this testing sampler
		reply.AdaptiveSampler = internal.SampleEverything{}

	}, configCallback, t)

	// start a web or background transaction, depending on test
	var txn Transaction

	if true == tc.WebTransaction {
		w := &sampleResponseWriter{
			header: make(http.Header),
		}

		r := func() *http.Request {
			r, err := http.NewRequest("GET", helloPath+helloQueryParams, nil)
			if nil != err {
				panic(err)
			}
			return r
		}()

		txn = app.StartTransaction("hello", w, r)
		t.Log("Starting Web Transaction")
	} else {
		txn = app.StartTransaction("hello", nil, nil)
		t.Log("Starting Background Transaction")
	}

	// If the tests wants us to have an error, give 'em an error
	if tc.RaisesException {
		txn.NoticeError(errors.New("my error message"))
	}

	// If there are no inbound payloads, invoke Accept on an empty inbound payload.
	if nil == tc.InboundPayloads {
		txn.AcceptDistributedTracePayload(TransportType{name: getTransport(tc.TransportType)}, nil)
	}

	for _, value := range tc.InboundPayloads {
		payload := makePayloadFromTestcaseInbound(t, value)
		txn.AcceptDistributedTracePayload(TransportType{name: getTransport(tc.TransportType)}, string(payload))
	}

	//call create each time an outbound payload appears in the testcase
	for _, expect := range tc.OutboundPayloads {
		actual := txn.CreateDistributedTracePayload().Text()
		assertTestCaseOutboundPayload(expect, t, actual)
	}

	err := txn.End()
	if nil != err {
		t.Error(err)
	}

	// create WantMetrics and assert
	wantMetrics := []internal.WantMetric{}
	for _, metric := range tc.ExpectedMetrics {
		wantMetrics = append(wantMetrics,
			internal.WantMetric{Name: metric[0].(string), Scope: "", Forced: nil, Data: nil})
	}
	app.ExpectMetricsPresent(t, wantMetrics)

	//Run through target events and assert for each
	for _, value := range tc.Intrinsics.TargetEvents {
		switch value {
		case "Transaction":
			assertTestCaseTransaction(app, t, tc)
		case "Span":
			assertTestCaseSpan(app, t, tc)
		case "TransactionError":
			assertTestCaseTransactionError(app, t, tc)
		}
	}
	t.Logf("Ending Test: %s", tc.TestName)

	extraAsserts(app, t, tc)
}

func assertTestCaseOutboundPayload(expect distributedTraceOutboundTestcase, t *testing.T, actual string) {
	type outboundTestcase struct {
		Version [2]uint                `json:"v"`
		Data    map[string]interface{} `json:"d"`
	}
	var actualPayload outboundTestcase
	var (
		errExpectedBadValue = errors.New("expected field in outbound payload has bad value")
		errExpectedMissing  = errors.New("expected field in outbound payload not found")
		errUnexpectedFound  = errors.New("found unexpected field in outbound payload")
	)
	err := json.Unmarshal([]byte(actual), &actualPayload)
	if nil != err {
		t.Error(err)
	}
	// Affirm that the exact values are in the payload.
	for k, v := range expect.Exact {
		if k != "v" {
			field := strings.Split(k, ".")[1]
			if v != actualPayload.Data[field] {
				t.Error(errExpectedBadValue)
			}
		}
	}
	// Affirm that the expected values are in the actual payload.
	for _, e := range expect.Expected {
		field := strings.Split(e, ".")[1]
		if nil == actualPayload.Data[field] {
			t.Error(errExpectedMissing)
		}
	}
	// Affirm that the unexpected values are not in the actual payload.
	for _, u := range expect.Unexpected {
		field := strings.Split(u, ".")[1]
		if nil != actualPayload.Data[field] {
			t.Error(errUnexpectedFound)
		}
	}
}

func assertTestCaseTransaction(app expectApp, t *testing.T, tc distributedTraceTestcase) {
	t.Log("Starting Transaction Event Assertions")
	wantEvent := internal.WantEvent{Intrinsics: map[string]interface{}{}}
	// we have common attributes, both exact and expected
	for k, v := range tc.Intrinsics.Common.Exact {
		wantEvent.Intrinsics[k] = v
	}
	for _, v := range tc.Intrinsics.Common.Expected {
		wantEvent.Intrinsics[v] = internal.MatchAnything
	}

	// we also have things specific to the transaction exvent
	for k, v := range tc.Intrinsics.Transaction.Exact {
		wantEvent.Intrinsics[k] = v
	}

	for _, v := range tc.Intrinsics.Transaction.Expected {
		wantEvent.Intrinsics[v] = internal.MatchAnything
	}

	wantEvents := []internal.WantEvent{wantEvent}
	app.ExpectTxnEventsPresent(t, wantEvents)

	combinedUnexpected := append(tc.Intrinsics.Common.Unexpected, tc.Intrinsics.Transaction.Unexpected...)
	app.ExpectTxnEventsAbsent(t, combinedUnexpected)

	t.Log("Ending Transaction Event Assertions")
}

func assertTestCaseSpan(app expectApp, t *testing.T, tc distributedTraceTestcase) {
	t.Log("Starting Span Event Assertions")
	wantEvent := internal.WantEvent{Intrinsics: map[string]interface{}{}}
	// we have common attributes, both exact and expected
	for k, v := range tc.Intrinsics.Common.Exact {
		wantEvent.Intrinsics[k] = v
	}
	for _, v := range tc.Intrinsics.Common.Expected {
		wantEvent.Intrinsics[v] = internal.MatchAnything
	}

	// we also have things specific to the transaction exvent
	for k, v := range tc.Intrinsics.Span.Exact {
		wantEvent.Intrinsics[k] = v
	}

	for _, v := range tc.Intrinsics.Span.Expected {
		wantEvent.Intrinsics[v] = internal.MatchAnything
	}

	wantEvents := []internal.WantEvent{wantEvent}
	app.ExpectSpanEventsPresent(t, wantEvents)

	combinedUnexpected := append(tc.Intrinsics.Common.Unexpected, tc.Intrinsics.Span.Unexpected...)
	app.ExpectSpanEventsAbsent(t, combinedUnexpected)
}

func assertTestCaseTransactionError(app expectApp, t *testing.T, tc distributedTraceTestcase) {
	t.Log("Starting Error Event Assertions")
	wantEvent := internal.WantEvent{Intrinsics: map[string]interface{}{}}
	// we have common attributes, both exact and expected
	for k, v := range tc.Intrinsics.Common.Exact {
		wantEvent.Intrinsics[k] = v
	}
	for _, v := range tc.Intrinsics.Common.Expected {
		wantEvent.Intrinsics[v] = internal.MatchAnything
	}

	for k, v := range tc.Intrinsics.TransactionError.Exact {
		wantEvent.Intrinsics[k] = v
	}
	for _, v := range tc.Intrinsics.TransactionError.Expected {
		wantEvent.Intrinsics[v] = internal.MatchAnything
	}

	wantEvents := []internal.WantEvent{wantEvent}
	app.ExpectErrorEventsPresent(t, wantEvents)

	combinedUnexpected := append(tc.Intrinsics.Common.Unexpected, tc.Intrinsics.TransactionError.Unexpected...)
	app.ExpectErrorEventsAbsent(t, combinedUnexpected)

	t.Log("Ending Error Event Assertions")
}

func TestDistributedTraceCrossAgent(t *testing.T) {
	var tcs []distributedTraceTestcase
	//
	input, err := crossagent.ReadFile(`distributed_tracing/distributed_tracing.json`)
	if nil != err {
		t.Fatal(err)
	}

	err = json.Unmarshal(input, &tcs)
	if nil != err {
		t.Fatal(err)
	}

	// Iterate over all cross-agent tests
	for _, tc := range tcs {
		runDistributedTraceCrossAgentTestcase(t, tc, func(app expectApp, t *testing.T, tc distributedTraceTestcase) {})

		// if there are specific test cases where we'd like to go above and
		// beyond the standard cross agent assertions, do so here
		if "spans_disabled_in_child" == tc.TestName {
			// if span events are disabled but distributed tracing is enabled, then
			// we expect there are zero span events
			runDistributedTraceCrossAgentTestcase(t, tc, func(app expectApp, t *testing.T, tc distributedTraceTestcase) {
				app.ExpectSpanEventsCount(t, 0)
			})
		}

	}
}

func TestDistributedTraceDisabledSpanEventsEnabled(t *testing.T) {
	app := testApp(distributedTracingReplyFields, disableDistributedTracerEnableSpanEvents, t)
	payload := makePayload(app, nil)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	if nil == err {
		t.Log("we expected an error with DT disabled")
		t.Fail()
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}

	// ensure no span events created
	app.ExpectSpanEventsCount(t, 0)
}
