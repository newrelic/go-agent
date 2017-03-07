package newrelic

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal"
)

func distributedTracingReplyFields(reply *internal.ConnectReply) {
	reply.AccountID = "123"
	reply.AppID = "456"
	reply.TrustedAccounts = []int{
		123,
	}
}

func makePayload(app Application, u *url.URL) DistributedTracePayload {
	txn := app.StartTransaction("hello", nil, nil)
	return txn.CreateDistributedTracePayload(u)
}

func TestPayloadConnection(t *testing.T) {
	app := testApp(distributedTracingReplyFields, nil, t)
	payload := makePayload(app, nil)
	ip, ok := payload.(internal.PayloadV1)
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
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                        "OtherTransaction/Go/hello",
			"nr.depth":                    2,
			"caller.type":                 "App",
			"caller.account":              "123",
			"caller.app":                  "456",
			"caller.transportType":        "HTTP",
			"caller.transportDuration":    internal.MatchAnything,
			"nr.order":                    0,
			"nr.referringTransactionGuid": ip.ID,
			"nr.tripId":                   ip.ID,
		},
	}})
}

func TestPayloadConnectionWithHost(t *testing.T) {
	app := testApp(distributedTracingReplyFields, nil, t)
	u, err := url.Parse("http://example.com/zip/zap?secret=shh")
	if nil != err {
		t.Fatal(err)
	}
	payload := makePayload(app, u)
	ip, ok := payload.(internal.PayloadV1)
	if !ok {
		t.Fatal(payload)
	}
	txn := app.StartTransaction("hello", nil, nil)
	err = txn.AcceptDistributedTracePayload(TransportHTTP, payload)
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
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                        "OtherTransaction/Go/hello",
			"nr.depth":                    2,
			"caller.type":                 "App",
			"caller.account":              "123",
			"caller.app":                  "456",
			"caller.transportType":        "HTTP",
			"caller.transportDuration":    internal.MatchAnything,
			"nr.order":                    0,
			"nr.referringTransactionGuid": ip.ID,
			"nr.tripId":                   ip.ID,
			"caller.host":                 "example.com",
		},
	}})
}

func TestPayloadConnectionText(t *testing.T) {
	app := testApp(distributedTracingReplyFields, nil, t)
	payload := makePayload(app, nil)
	ip, ok := payload.(internal.PayloadV1)
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
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                        "OtherTransaction/Go/hello",
			"nr.depth":                    2,
			"caller.type":                 "App",
			"caller.account":              "123",
			"caller.app":                  "456",
			"caller.transportType":        "HTTP",
			"caller.transportDuration":    internal.MatchAnything,
			"nr.order":                    0,
			"nr.referringTransactionGuid": ip.ID,
			"nr.tripId":                   ip.ID,
		},
	}})
}

func validBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

func TestPayloadConnectionHTTPSafe(t *testing.T) {
	app := testApp(distributedTracingReplyFields, nil, t)
	payload := makePayload(app, nil)
	ip, ok := payload.(internal.PayloadV1)
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
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                        "OtherTransaction/Go/hello",
			"nr.depth":                    2,
			"caller.type":                 "App",
			"caller.account":              "123",
			"caller.app":                  "456",
			"caller.transportType":        "HTTP",
			"caller.transportDuration":    internal.MatchAnything,
			"nr.order":                    0,
			"nr.referringTransactionGuid": ip.ID,
			"nr.tripId":                   ip.ID,
		},
	}})
}

func TestPayloadConnectionNotConnected(t *testing.T) {
	app := testApp(nil, nil, t)
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
	app.ExpectMetrics(t, backgroundMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
	}})
}

func TestPayloadConnectionEmptyString(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, "")
	if nil != err {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, backgroundMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
	}})
}

func TestCreatePayloadFinished(t *testing.T) {
	app := testApp(distributedTracingReplyFields, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	txn.End()
	payload := txn.CreateDistributedTracePayload(nil)
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
	app := testApp(distributedTracingReplyFields, nil, t)
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
	app.ExpectMetrics(t, backgroundMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
	}})
}

func TestPayloadTypeUnknown(t *testing.T) {
	app := testApp(distributedTracingReplyFields, nil, t)
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
	app.ExpectMetrics(t, backgroundMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
	}})
}

func TestPayloadAcceptAfterCreate(t *testing.T) {
	app := testApp(distributedTracingReplyFields, nil, t)
	payload := makePayload(app, nil)
	txn := app.StartTransaction("hello", nil, nil)
	txn.CreateDistributedTracePayload(nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	if errOutboundPayloadCreated != err {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, backgroundMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
	}})
}

func TestPayloadFromApplication(t *testing.T) {
	// Some agents may omit certain payload fields if CreateDistributedTracePayload is used
	// as a method on the application.
	app := testApp(distributedTracingReplyFields, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP,
		`{
			"v":[1,0],
			"d":{
				"ty":"App",
				"ap":"456",
				"ac":"123",
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
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                     "OtherTransaction/Go/hello",
			"caller.type":              "App",
			"caller.account":           "123",
			"caller.app":               "456",
			"caller.transportType":     "HTTP",
			"caller.transportDuration": internal.MatchAnything,
		},
	}})
}

func TestPayloadFutureVersion(t *testing.T) {
	app := testApp(distributedTracingReplyFields, nil, t)
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
	app.ExpectMetrics(t, backgroundMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
	}})
}

func TestPayloadInvalidPriority(t *testing.T) {
	app := testApp(distributedTracingReplyFields, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP,
		`{
			"v":[1,0],
			"d":{
				"ty":"App",
				"ap":"456",
				"ac":"123",
				"ti":1488325987402,
				"pr":"!!!"
			}
		}`)
	if nil == err {
		t.Error("missing expected invalid priority error")
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, backgroundMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
	}})
}

func TestPayloadFromFuture(t *testing.T) {
	app := testApp(distributedTracingReplyFields, nil, t)
	payload := makePayload(app, nil)
	ip, ok := payload.(internal.PayloadV1)
	if !ok {
		t.Fatal(payload)
	}
	ip.Time = time.Now().Add(1 * time.Hour)
	ip.TimeMS = internal.TimeToUnixMilliseconds(ip.Time)
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
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                        "OtherTransaction/Go/hello",
			"nr.depth":                    2,
			"caller.type":                 "App",
			"caller.account":              "123",
			"caller.app":                  "456",
			"caller.transportType":        "HTTP",
			"caller.transportDuration":    0,
			"nr.order":                    0,
			"nr.referringTransactionGuid": ip.ID,
			"nr.tripId":                   ip.ID,
		},
	}})
}

func TestPayloadWebWithErrorAndProxy(t *testing.T) {
	app := testApp(distributedTracingReplyFields, nil, t)
	payload := makePayload(app, nil)
	ip, ok := payload.(internal.PayloadV1)
	if !ok {
		t.Fatal(payload)
	}
	req, err := http.NewRequest("GET", helloPath, nil)
	req.Header.Add("X-Newrelic-Timestamp-Myproxy", "1465793282.12345")
	if nil != err {
		t.Fatal(err)
	}
	txn := app.StartTransaction("hello", nil, req)
	err = txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	if nil != err {
		t.Error(err)
	}
	txn.NoticeError(myError{})
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: singleCount},
		{Name: "Errors/WebTransaction/Go/hello", Scope: "", Forced: true, Data: singleCount},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "ErrorsByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: singleCount},
		{Name: "ErrorsByCaller/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: singleCount},
		{Name: "WebFrontend/QueueTime", Scope: "", Forced: true, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allWeb", Scope: "", Forced: false, Data: nil},
		{Name: "IntermediaryTransportDuration/App/123/456/HTTP/Myproxy/all", Scope: "", Forced: false, Data: nil},
		{Name: "IntermediaryTransportDuration/App/123/456/HTTP/Myproxy/allWeb", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                             "WebTransaction/Go/hello",
			"nr.apdexPerfZone":                 "F",
			"nr.depth":                         2,
			"caller.type":                      "App",
			"caller.account":                   "123",
			"caller.app":                       "456",
			"caller.transportType":             "HTTP",
			"caller.transportDuration":         internal.MatchAnything,
			"caller.transportDuration.Myproxy": internal.MatchAnything,
			"queueDuration":                    internal.MatchAnything,
			"nr.order":                         0,
			"nr.referringTransactionGuid":      ip.ID,
			"nr.tripId":                        ip.ID,
		},
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
		Caller:  "go-agent.TestPayloadWebWithErrorAndProxy",
		URL:     "/hello",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":                      "newrelic.myError",
			"error.message":                    "my msg",
			"transactionName":                  "WebTransaction/Go/hello",
			"nr.depth":                         2,
			"caller.type":                      "App",
			"caller.account":                   "123",
			"caller.app":                       "456",
			"caller.transportType":             "HTTP",
			"caller.transportDuration":         internal.MatchAnything,
			"caller.transportDuration.Myproxy": internal.MatchAnything,
			"queueDuration":                    internal.MatchAnything,
			"nr.order":                         0,
			"nr.referringTransactionGuid":      ip.ID,
			"nr.tripId":                        ip.ID,
		},
	}})
}

func TestPayloadHigherDepthAndSequence(t *testing.T) {
	app := testApp(distributedTracingReplyFields, nil, t)
	payload1 := makePayload(app, nil)
	ip1, ok := payload1.(internal.PayloadV1)
	if !ok {
		t.Fatal(payload1)
	}
	txn := app.StartTransaction("hello", nil, nil)
	txn.AcceptDistributedTracePayload(TransportHTTP, payload1)
	txn.CreateDistributedTracePayload(nil) // create an unused payload to increase sequence
	payload2 := txn.CreateDistributedTracePayload(nil)
	ip2, ok := payload2.(internal.PayloadV1)
	if !ok {
		t.Fatal(payload2)
	}
	txn2 := app.StartTransaction("hello", nil, nil)
	err := txn2.AcceptDistributedTracePayload(TransportHTTP, payload2)
	if nil != err {
		t.Error(err)
	}
	err = txn2.End()
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
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                        "OtherTransaction/Go/hello",
			"nr.depth":                    3,
			"caller.type":                 "App",
			"caller.account":              "123",
			"caller.app":                  "456",
			"caller.transportType":        "HTTP",
			"caller.transportDuration":    internal.MatchAnything,
			"nr.order":                    1,
			"nr.referringTransactionGuid": ip2.ID,
			"nr.tripId":                   ip1.ID,
		},
	}})
}

func TestPayloadUntrustedAccount(t *testing.T) {
	app := testApp(distributedTracingReplyFields, nil, t)
	payload := makePayload(app, nil)
	ip, ok := payload.(internal.PayloadV1)
	if !ok {
		t.Fatal(payload)
	}
	ip.Account = "12345"
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, ip)
	if _, ok := err.(internal.ErrUntrustedAccountID); !ok {
		t.Error(err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
	app.ExpectMetrics(t, backgroundMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
	}})
}
