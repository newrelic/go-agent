package newrelic

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal"
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
		123: struct{}{},
	}
	reply.TrustedAccountKey = "123"

	reply.AdaptiveSampler = internal.SampleEverything{}
}

func distributedTracingReplyFieldsNeedTrustKey(reply *internal.ConnectReply) {
	reply.AccountID = "123"
	reply.AppID = "456"
	reply.PrimaryAppID = "456"
	reply.TrustedAccounts = map[int]struct{}{
		123: struct{}{},
	}
	reply.TrustedAccountKey = "789"
}

func makePayload(app Application, u *url.URL) DistributedTracePayload {
	txn := app.StartTransaction("hello", nil, nil)
	return txn.CreateDistributedTracePayload()
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
