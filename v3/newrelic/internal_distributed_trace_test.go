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
	"time"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/crossagent"
)

func distributedTracingReplyFields(reply *internal.ConnectReply) {
	reply.AccountID = "123"
	reply.AppID = "456"
	reply.PrimaryAppID = "456"
	reply.TrustedAccounts = map[int]struct{}{
		123: {},
	}
	reply.TrustedAccountKey = "123"

	reply.SetSampleEverything()
	reply.TraceIDGenerator = internal.NewTraceIDGenerator(1)
	reply.DistributedTraceTimestampGenerator = func() time.Time {
		return time.Unix(1577830891, 900000000)
	}
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

func distributedTracingReplyFieldsSpansDisabled(reply *internal.ConnectReply) {
	reply.AccountID = "123"
	reply.AppID = "456"
	reply.PrimaryAppID = "456"
	reply.TrustedAccounts = map[int]struct{}{
		123: {},
	}
	reply.TrustedAccountKey = "123"

	reply.SetSampleEverything()
	reply.TraceIDGenerator = internal.NewTraceIDGenerator(1)
	reply.CollectSpanEvents = false
	reply.DistributedTraceTimestampGenerator = func() time.Time {
		return time.Unix(1577830891, 900000000)
	}
}

func getDTHeaders(app *Application) http.Header {
	hdrs := http.Header{}
	app.StartTransaction("hello").thread.CreateDistributedTracePayload(hdrs)
	return hdrs
}

func headersFromString(s string) http.Header {
	return map[string][]string{DistributedTraceNewRelicHeader: {s}}
}

func makeHeaders(t *testing.T) http.Header {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello")
	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)
	return hdrs
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

func enableW3COnly(cfg *Config) {
	cfg.DistributedTracer.Enabled = true
	cfg.DistributedTracer.ExcludeNewRelicHeader = true
}

func enableW3COnlySampledAlwaysOn(cfg *Config) {
	cfg.DistributedTracer.Enabled = true
	cfg.DistributedTracer.ExcludeNewRelicHeader = true
	cfg.DistributedTracer.Sampler.RemoteParentSampled = "always_on"
}

func enableW3COnlySampledAlwaysOff(cfg *Config) {
	cfg.DistributedTracer.Enabled = true
	cfg.DistributedTracer.ExcludeNewRelicHeader = true
	cfg.DistributedTracer.Sampler.RemoteParentSampled = "always_off"
}

func enableW3COnlyNotSampledAlwaysOn(cfg *Config) {
	cfg.DistributedTracer.Enabled = true
	cfg.DistributedTracer.ExcludeNewRelicHeader = true
	cfg.DistributedTracer.Sampler.RemoteParentNotSampled = "always_on"
}
func enableW3COnlyNotSampledAlwaysOff(cfg *Config) {
	cfg.DistributedTracer.Enabled = true
	cfg.DistributedTracer.ExcludeNewRelicHeader = true
	cfg.DistributedTracer.Sampler.RemoteParentNotSampled = "always_off"
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

var (
	distributedTracingSuccessMetrics = []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: singleCount},
	}
)

func TestPayloadConnection(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	hdrs := getDTHeaders(app.Application)
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	app.expectNoLoggedErrors(t)
	txn.End()
	app.expectNoLoggedErrors(t)
	app.ExpectMetrics(t, distributedTracingSuccessMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                     "OtherTransaction/Go/hello",
			"parent.type":              "App",
			"parent.account":           "123",
			"parent.app":               "456",
			"parent.transportType":     "HTTP",
			"parent.transportDuration": internal.MatchAnything,
			"parentId":                 "52fdfc072182654f",
			"traceId":                  "52fdfc072182654f163f5f0f9a621d72",
			"parentSpanId":             "9566c74d10d1e2c6",
			"guid":                     internal.MatchAnything,
			"sampled":                  true,
			"priority":                 1.437714, // priority must be >1 when sampled is true
		},
	}})
}

func TestAcceptMultiple(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	hdrs := getDTHeaders(app.Application)
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	app.expectNoLoggedErrors(t)
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": errAlreadyAccepted.Error(),
	})
	txn.End()
	app.expectNoLoggedErrors(t)
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/Ignored/Multiple", Scope: "", Forced: true, Data: singleCount},
	}, distributedTracingSuccessMetrics...))
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                     "OtherTransaction/Go/hello",
			"parent.type":              "App",
			"parent.account":           "123",
			"parent.app":               "456",
			"parent.transportType":     "HTTP",
			"parent.transportDuration": internal.MatchAnything,
			"parentId":                 "52fdfc072182654f",
			"traceId":                  "52fdfc072182654f163f5f0f9a621d72",
			"parentSpanId":             "9566c74d10d1e2c6",
			"guid":                     internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"priority":                 internal.MatchAnything,
		},
	}})
}

func TestInsertDistributedTraceHeadersNotConnected(t *testing.T) {
	// Test that DT headers do not get created if the connect reply does not
	// contain the necessary fields.
	app := testApp(nil, enableBetterCAT, t)
	txn := app.StartTransaction("hello")
	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)
	if len(hdrs) != 0 {
		t.Error(hdrs)
	}
	app.expectNoLoggedErrors(t)
}

func TestAcceptDistributedTraceHeadersNil(t *testing.T) {
	// Test that AcceptDistributedTraceHeaders does not have issues
	// accepting nil headers.
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, nil)
	app.expectNoLoggedErrors(t)
	txn.End()
	app.expectNoLoggedErrors(t)
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/Ignored/Null", Scope: "", Forced: true, Data: nil},
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

func TestAcceptDistributedTraceHeadersBetterCatDisabled(t *testing.T) {
	// Test that AcceptDistributedTraceHeaders only accepts DT headers if DT
	// is enabled.
	app := testApp(nil, disableCAT, t)
	hdrs := makeHeaders(t)
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": errInboundPayloadDTDisabled.Error(),
	})
	txn.End()
	app.expectNoLoggedErrors(t)
	app.ExpectMetrics(t, backgroundMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
	}})
}

func TestPayloadTransactionsDisabled(t *testing.T) {
	cfgFn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.SpanEvents.Enabled = true
		cfg.TransactionEvents.Enabled = false
	}
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)
	if len(hdrs) != 0 {
		t.Fatal(hdrs)
	}
	txn.End()
	app.expectNoLoggedErrors(t)
}

func TestPayloadConnectionEmptyString(t *testing.T) {
	app := testApp(nil, enableBetterCAT, t)
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(""))
	app.expectNoLoggedErrors(t)
	txn.End()
	app.expectNoLoggedErrors(t)
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
	txn := app.StartTransaction("hello")
	txn.End()
	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)
	if len(hdrs) != 0 {
		t.Fatal(hdrs)
	}
}

func TestAcceptPayloadFinished(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	hdrs := getDTHeaders(app.Application)
	txn := app.StartTransaction("hello")
	txn.End()
	app.expectNoLoggedErrors(t)
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": errAlreadyEnded.Error(),
	})
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
	hdrs1 := getDTHeaders(app.Application)
	txn := app.StartTransaction("hello")
	hdrs2 := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs2)
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs1)
	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": errOutboundPayloadCreated.Error(),
	})
	txn.End()
	app.expectNoLoggedErrors(t)
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: singleCount},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: singleCount},
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
	txn := app.StartTransaction("hello")
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
	txn.AcceptDistributedTraceHeaders(emptyTransport, headersFromString(p))
	app.expectNoLoggedErrors(t)
	txn.End()
	app.expectNoLoggedErrors(t)
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
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
	txn := app.StartTransaction("hello")
	p := `{
			"v":[100,0],
			"d":{
				"ty":"App",
				"ap":"456",
				"ac":"123",
				"ti":1488325987402
			}
		}`
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(p))
	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": "unsupported major version number 100",
	})
	txn.End()
	app.expectNoLoggedErrors(t)
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/Ignored/MajorVersion", Scope: "", Forced: true, Data: singleCount},
	}, backgroundUnknownCallerWithTransport...))
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
	txn := app.StartTransaction("hello")
	p := `{
			"v":[0,1],
			"d":[]
		}`
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(p))
	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": "unable to unmarshal payload data: json: cannot unmarshal array into Go value of type newrelic.payload",
	})
	txn.End()
	app.expectNoLoggedErrors(t)
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/ParseException", Scope: "", Forced: true, Data: singleCount},
	}, backgroundUnknownCallerWithTransport...))
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
	hdrs := http.Header{}
	traceParent := "00-52fdfc072182654f163f5f0f9a621d72-9566c74d10037c4d-01"
	traceState := "123@nr=0-0-123-456-9566c74d10037c4d-52fdfc072182654f-1-0.390345-TIME"
	futureTime := time.Now().Add(1 * time.Hour)
	timeStr := fmt.Sprintf("%d", timeToUnixMilliseconds(futureTime))
	traceState = strings.Replace(traceState, "TIME", timeStr, 1)
	hdrs.Set(DistributedTraceW3CTraceParentHeader, traceParent)
	hdrs.Set(DistributedTraceW3CTraceStateHeader, traceState)

	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	app.expectNoLoggedErrors(t)
	txn.End()
	app.expectNoLoggedErrors(t)
	app.ExpectMetrics(t, distributedTracingSuccessMetrics)
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                     "OtherTransaction/Go/hello",
			"parent.type":              "App",
			"parent.account":           "123",
			"parent.app":               "456",
			"parent.transportType":     "HTTP",
			"parent.transportDuration": 0,
			"parentId":                 "52fdfc072182654f",
			"traceId":                  "52fdfc072182654f163f5f0f9a621d72",
			"parentSpanId":             "9566c74d10037c4d",
			"guid":                     internal.MatchAnything,
			"sampled":                  internal.MatchAnything,
			"priority":                 internal.MatchAnything,
		},
	}})
}

func TestPayloadUntrustedAccount(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	p := `{
			"v":[0,1],
			"d":{
				"ty":"App",
				"ap":"456",
				"ac":"321",
				"id":"id",
				"tr":"traceID",
				"ti":1488325987402
			}
		}`

	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(p))
	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": errTrustedAccountKey.Error(),
	})
	txn.End()
	app.expectNoLoggedErrors(t)
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/Ignored/UntrustedAccount", Scope: "", Forced: true, Data: singleCount},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: singleCount},
	}, backgroundUnknownCallerWithTransport...))
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
	txn := app.StartTransaction("hello")
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
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(p))
	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": "payload is missing Version/v",
	})
	txn.End()
	app.expectNoLoggedErrors(t)
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
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(p))
	app.expectNoLoggedErrors(t)
	txn.End()
	app.expectNoLoggedErrors(t)
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
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(p))
	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": errTrustedAccountKey.Error(),
	})
	txn.End()
	app.expectNoLoggedErrors(t)
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
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(p))
	app.expectNoLoggedErrors(t)
	txn.End()
	app.expectNoLoggedErrors(t)
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
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(p))
	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": errTrustedAccountKey.Error(),
	})
	txn.End()
	app.expectNoLoggedErrors(t)
}

var (
	backgroundUnknownCaller = []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
	}
	backgroundUnknownCallerWithTransport = []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/HTTP/allOther", Scope: "", Forced: false, Data: nil},
	}
)

func TestNilPayload(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, nil)
	app.expectNoLoggedErrors(t)
	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/Ignored/Null", Scope: "", Forced: true, Data: singleCount},
	}, backgroundUnknownCaller...))
}

func TestNoticeErrorPayload(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	txn := app.StartTransaction("hello")
	txn.NoticeError(errors.New("oh no"))

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Errors/all", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Errors/OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "ErrorsByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
	}, backgroundUnknownCaller...))
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

	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(p))

	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": "payload is missing both guid/id and TransactionId/tx",
	})
	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/ParseException", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCallerWithTransport...))
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

	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(p))
	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": "payload is missing Version/v",
	})
	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/ParseException", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCallerWithTransport...))
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

	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(p))

	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": "payload is missing Account/ac",
	})
	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/ParseException", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCallerWithTransport...))
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

	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(p))

	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": "unable to unmarshal payload: unexpected end of JSON input",
	})
	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/ParseException", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCallerWithTransport...))
}

func TestErrorsByCaller(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	txn := app.StartTransaction("hello")
	hdrs := getDTHeaders(app.Application)
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	app.expectNoLoggedErrors(t)

	txn.NoticeError(errors.New("oh no"))

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},

		{Name: "TransportDuration/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: nil},
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
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	// empty/shim payload objects return empty strings
	if len(hdrs) != 0 {
		t.Log("Non empty result of InsertDistributedTraceHeaders() method:", hdrs)
		t.Fail()
	}
	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
	})

}

func TestCreateDistributedTraceBetterCatDisabled(t *testing.T) {

	// when distributed tracing is disabled, CreateDistributedTracePayload
	// should return a value that indicates an empty payload. Examples of
	// this depend on language but may be nil/null/None or an empty payload
	// object.

	app := testApp(distributedTracingReplyFields, enableOldCATDisableBetterCat, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	if len(hdrs) != 0 {
		t.Log("Non empty result of InsertDistributedTraceHeaders() method:", hdrs)
		t.Fail()
	}

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
	})

}

func TestCreateDistributedTraceBetterCatEnabled(t *testing.T) {

	// When distributed tracing is enabled and the application is connected,
	// CreateDistributedTracePayload should return a valid payload object

	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	if len(hdrs) == 0 {
		t.Log("Empty result of InsertDistributedTraceHeaders() method:", hdrs)
		t.Fail()
	}

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
}

func isZeroValue(x interface{}) bool {
	// https://stackoverflow.com/questions/13901819/quick-way-to-detect-empty-values-via-reflection-in-go
	return nil == x || x == reflect.Zero(reflect.TypeOf(x)).Interface()
}

func payloadFieldsFromHeaders(t *testing.T, hdrs http.Header) (out struct {
	Version []int                  `json:"v"`
	Data    map[string]interface{} `json:"d"`
}) {
	encoded := hdrs.Get(DistributedTraceNewRelicHeader)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal("unable to bas64 decode tracing header", err)
	}
	if err := json.Unmarshal(decoded, &out); nil != err {
		t.Fatal("unable to unmarshal payload NRText", err)
	}
	return
}

func testPayloadFieldsPresent(t *testing.T, hdrs http.Header, keys ...string) {
	out := payloadFieldsFromHeaders(t, hdrs)
	for _, key := range keys {
		val, ok := out.Data[key]
		if !ok {
			t.Fatal("required key missing", key)
		}
		if isZeroValue(val) {
			t.Fatal("value has default value", key, val)
		}
	}
}

func TestCreateDistributedTraceRequiredFields(t *testing.T) {

	// creates a distributed trace payload and then checks
	// to ensure the required fields are in place
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	testPayloadFieldsPresent(t, hdrs, "ty", "ac", "ap", "tr", "ti")

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
}

func TestCreateDistributedTraceTrustKeyAbsent(t *testing.T) {

	// creates a distributed trace payload and then checks
	// to ensure the required fields are in place
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	data := payloadFieldsFromHeaders(t, hdrs)

	if nil != data.Data["tk"] {
		t.Fatal("unexpected trust key (tk)", hdrs)
	}

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
}

func TestCreateDistributedTraceTrustKeyNeeded(t *testing.T) {

	// creates a distributed trace payload and then checks
	// to ensure the required fields are in place
	app := testApp(distributedTracingReplyFieldsNeedTrustKey, enableBetterCAT, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	testPayloadFieldsPresent(t, hdrs, "tk")

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
}

func TestCreateDistributedTraceAfterAcceptSampledTrue(t *testing.T) {

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
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(p))
	app.expectNoLoggedErrors(t)

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	testPayloadFieldsPresent(t, hdrs,
		"ty", "ac", "ap", "tr", "ti", "pr", "sa")

	txn.End()
	app.expectNoLoggedErrors(t)
}

func TestCreateDistributedTraceAfterAcceptSampledNotSet(t *testing.T) {

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
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, headersFromString(p))
	app.expectNoLoggedErrors(t)

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	testPayloadFieldsPresent(t, hdrs,
		"ty", "ac", "ap", "id", "tr", "ti", "pr", "sa")

	txn.End()
	app.expectNoLoggedErrors(t)
}

type fieldExpectations struct {
	Exact      map[string]interface{} `json:"exact,omitempty"`
	Expected   []string               `json:"expected,omitempty"`
	Unexpected []string               `json:"unexpected,omitempty"`
}

type distributedTraceTestcase struct {
	TestName          string            `json:"test_name"`
	Comment           string            `json:"comment,omitempty"`
	TrustedAccountKey string            `json:"trusted_account_key"`
	AccountID         string            `json:"account_id"`
	WebTransaction    bool              `json:"web_transaction"`
	RaisesException   bool              `json:"raises_exception"`
	ForceSampledTrue  bool              `json:"force_sampled_true"`
	SpanEventsEnabled bool              `json:"span_events_enabled"`
	MajorVersion      int               `json:"major_version"`
	MinorVersion      int               `json:"minor_version"`
	TransportType     string            `json:"transport_type"`
	InboundPayloads   []json.RawMessage `json:"inbound_payloads"`

	OutboundPayloads []fieldExpectations `json:"outbound_payloads,omitempty"`

	Intrinsics struct {
		TargetEvents     []string           `json:"target_events"`
		Common           *fieldExpectations `json:"common,omitempty"`
		Transaction      *fieldExpectations `json:"Transaction,omitempty"`
		Span             *fieldExpectations `json:"Span,omitempty"`
		TransactionError *fieldExpectations `json:"TransactionError,omitempty"`
		UnexpectedEvents []string           `json:"unexpected_events,omitempty"`
	} `json:"intrinsics"`

	ExpectedMetrics [][2]interface{} `json:"expected_metrics"`
}

func (fe *fieldExpectations) add(intrinsics map[string]interface{}) {
	if nil != fe {
		for k, v := range fe.Exact {
			intrinsics[k] = v
		}
		for _, v := range fe.Expected {
			intrinsics[v] = internal.MatchAnything
		}
	}
}

func (fe *fieldExpectations) unexpected() []string {
	if nil != fe {
		return fe.Unexpected
	}
	return nil
}

// getTransport ensures that our transport names match cross agent test values.
func getTransport(transport string) TransportType {
	switch TransportType(transport) {
	case TransportHTTP, TransportHTTPS, TransportKafka, TransportJMS, TransportIronMQ, TransportAMQP,
		TransportQueue, TransportOther:
		return TransportType(transport)
	default:
		return TransportUnknown
	}
}

func runDistributedTraceCrossAgentTestcase(tst *testing.T, tc distributedTraceTestcase, extraAsserts func(expectApp, internal.Validator)) {
	t := extendValidator(tst, "test="+tc.TestName)
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
		reply.SetSampleEverything()

	}, configCallback, tst)

	txn := app.StartTransaction("hello")
	if tc.WebTransaction {
		txn.SetWebRequestHTTP(nil)
	}

	// If the tests wants us to have an error, give 'em an error
	if tc.RaisesException {
		txn.NoticeError(errors.New("my error message"))
	}

	// If there are no inbound payloads, invoke Accept on an empty inbound payload.
	if nil == tc.InboundPayloads {
		txn.AcceptDistributedTraceHeaders(getTransport(tc.TransportType), nil)
	}

	for _, value := range tc.InboundPayloads {
		// Note that the error return value is not tested here because
		// some of the tests are intentionally errors.
		txn.AcceptDistributedTraceHeaders(getTransport(tc.TransportType), headersFromString(string(value)))
	}

	//call create each time an outbound payload appears in the testcase
	for _, expect := range tc.OutboundPayloads {
		hdrs := http.Header{}
		txn.InsertDistributedTraceHeaders(hdrs)
		actual := hdrs.Get(DistributedTraceNewRelicHeader)
		assertTestCaseOutboundPayload(expect, t, actual)
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

	extraTxnFields := &fieldExpectations{Expected: []string{"name"}}
	if tc.WebTransaction {
		extraTxnFields.Expected = append(extraTxnFields.Expected, "nr.apdexPerfZone")
	}

	extraSpanFields := &fieldExpectations{
		Expected: []string{"name", "transaction.name", "category", "nr.entryPoint"},
	}

	// There is a single test with an error (named "exception"), so these
	// error expectations can be hard coded. TODO: Move some of these.
	// fields into the cross agent tests.
	extraErrorFields := &fieldExpectations{
		Expected: []string{"parent.type", "parent.account", "parent.app",
			"parent.transportType", "error.message", "transactionName",
			"parent.transportDuration", "error.class", "spanId"},
	}

	for _, value := range tc.Intrinsics.TargetEvents {
		switch value {
		case "Transaction":
			assertTestCaseIntrinsics(t,
				app.ExpectTxnEvents,
				tc.Intrinsics.Common,
				tc.Intrinsics.Transaction,
				extraTxnFields)
		case "Span":
			assertTestCaseIntrinsics(t,
				app.ExpectSpanEvents,
				tc.Intrinsics.Common,
				tc.Intrinsics.Span,
				extraSpanFields)

		case "TransactionError":
			assertTestCaseIntrinsics(t,
				app.ExpectErrorEvents,
				tc.Intrinsics.Common,
				tc.Intrinsics.TransactionError,
				extraErrorFields)
		}
	}

	extraAsserts(app, t)
}

func assertTestCaseOutboundPayload(expect fieldExpectations, t internal.Validator, encoded string) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if nil != err {
		t.Error("unable to decode payload header", err)
		return
	}
	type outboundTestcase struct {
		Version [2]uint                `json:"v"`
		Data    map[string]interface{} `json:"d"`
	}
	var actualPayload outboundTestcase
	err = json.Unmarshal([]byte(decoded), &actualPayload)
	if nil != err {
		t.Error(err)
	}
	// Affirm that the exact values are in the payload.
	for k, v := range expect.Exact {
		if k != "v" {
			field := strings.Split(k, ".")[1]
			if v != actualPayload.Data[field] {
				t.Error(fmt.Sprintf("exact outbound payload field mismatch key=%s wanted=%v got=%v",
					k, v, actualPayload.Data[field]))
			}
		}
	}
	// Affirm that the expected values are in the actual payload.
	for _, e := range expect.Expected {
		field := strings.Split(e, ".")[1]
		if nil == actualPayload.Data[field] {
			t.Error(fmt.Sprintf("expected outbound payload field missing key=%s", e))
		}
	}
	// Affirm that the unexpected values are not in the actual payload.
	for _, u := range expect.Unexpected {
		field := strings.Split(u, ".")[1]
		if nil != actualPayload.Data[field] {
			t.Error(fmt.Sprintf("unexpected outbound payload field present key=%s", u))
		}
	}
}

func assertTestCaseIntrinsics(t internal.Validator,
	expect func(internal.Validator, []internal.WantEvent),
	fields ...*fieldExpectations) {

	intrinsics := map[string]interface{}{}
	for _, f := range fields {
		f.add(intrinsics)
	}
	expect(t, []internal.WantEvent{{Intrinsics: intrinsics}})
}

func TestDistributedTraceCrossAgent(t *testing.T) {
	var tcs []distributedTraceTestcase
	data, err := crossagent.ReadFile(`distributed_tracing/distributed_tracing.json`)
	if nil != err {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &tcs); nil != err {
		t.Fatal(err)
	}
	// Test that we are correctly parsing all of the testcase fields by
	// comparing an opaque object from original JSON to an object from JSON
	// created by our testcases.
	backToJSON, err := json.Marshal(tcs)
	if nil != err {
		t.Fatal(err)
	}
	var fromFile []map[string]interface{}
	var fromMarshalled []map[string]interface{}
	if err := json.Unmarshal(data, &fromFile); nil != err {
		t.Fatal(err)
	}
	if err := json.Unmarshal(backToJSON, &fromMarshalled); nil != err {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromFile, fromMarshalled) {
		t.Error(internal.CompactJSONString(string(data)), "\n",
			internal.CompactJSONString(string(backToJSON)))
	}

	// Iterate over all cross-agent tests
	for _, tc := range tcs {
		extraAsserts := func(app expectApp, t internal.Validator) {}
		if "spans_disabled_in_child" == tc.TestName {
			// if span events are disabled but distributed tracing is enabled, then
			// we expect there are zero span events
			extraAsserts = func(app expectApp, t internal.Validator) {
				app.ExpectSpanEvents(t, nil)
			}
		}
		runDistributedTraceCrossAgentTestcase(t, tc, extraAsserts)
	}
}

func TestDistributedTraceDisabledSpanEventsEnabled(t *testing.T) {
	app := testApp(distributedTracingReplyFields, disableDistributedTracerEnableSpanEvents, t)
	hdrs := makeHeaders(t)
	txn := app.StartTransaction("hello")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": errInboundPayloadDTDisabled.Error(),
	})
	txn.End()
	app.expectNoLoggedErrors(t)

	// ensure no span events created
	app.ExpectSpanEvents(t, nil)
}

func TestCreatePayloadAppNotConnected(t *testing.T) {
	// Test that an app which isn't connected does not create distributed
	// trace payloads.
	app := testApp(nil, enableBetterCAT, t)
	txn := app.StartTransaction("hello")
	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)
	if len(hdrs) != 0 {
		t.Error(hdrs)
	}
}

func TestCreatePayloadReplyMissingTrustKey(t *testing.T) {
	// Test that an app whose reply is missing the trust key does not create
	// distributed trace payloads.
	app := testApp(func(reply *internal.ConnectReply) {
		distributedTracingReplyFields(reply)
		reply.TrustedAccountKey = ""
	}, enableBetterCAT, t)
	txn := app.StartTransaction("hello")
	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)
	if len(hdrs) != 0 {
		t.Error(hdrs)
	}
}

func TestAcceptPayloadAppNotConnected(t *testing.T) {
	// Test that an app which isn't connected does not accept distributed
	// trace payloads.
	app := testApp(nil, enableBetterCAT, t)
	txn := testApp(distributedTracingReplyFields, enableBetterCAT, t).
		StartTransaction("name")
	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)
	if len(hdrs) == 0 {
		t.Fatal(hdrs)
	}
	txn2 := app.StartTransaction("hello")
	txn2.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	app.expectNoLoggedErrors(t)
	txn2.End()
	app.ExpectMetrics(t, backgroundUnknownCaller)
}

func TestAcceptPayloadReplyMissingTrustKey(t *testing.T) {
	// Test that an app whose reply is missing a trust key does not accept
	// distributed trace payloads.
	app := testApp(func(reply *internal.ConnectReply) {
		distributedTracingReplyFields(reply)
		reply.TrustedAccountKey = ""
	}, enableBetterCAT, t)
	txn := testApp(distributedTracingReplyFields, enableBetterCAT, t).
		StartTransaction("name")
	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)
	if len(hdrs) == 0 {
		t.Fatal(hdrs)
	}
	txn2 := app.StartTransaction("hello")
	txn2.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	app.expectNoLoggedErrors(t)
	txn2.End()
	app.ExpectMetrics(t, backgroundUnknownCaller)
}

func verifyHeaders(t *testing.T, actual http.Header, expected http.Header) {
	if !reflect.DeepEqual(actual, expected) {
		t.Error("Headers do not match - expected/actual: ", expected, actual)
	}
}

func TestW3CTraceHeaders(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableW3COnly, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456-9566c74d10d1e2c6-52fdfc072182654f-1-1.437714-1577830891900"},
	}
	verifyHeaders(t, hdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))

}

func TestW3CTraceHeadersSamplingAlwaysOn(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableW3COnlySampledAlwaysOn, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456-9566c74d10d1e2c6-52fdfc072182654f-1-2-1577830891900"},
	}
	verifyHeaders(t, hdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":     "OtherTransaction/Go/hello",
				"guid":     internal.MatchAnything,
				"priority": 2.0,
				"sampled":  internal.MatchAnything,
				"traceId":  internal.MatchAnything,
			},
		},
	})

}
func TestW3CTraceHeadersSamplingAlwaysOff(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableW3COnlySampledAlwaysOff, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456-9566c74d10d1e2c6-52fdfc072182654f-1-0-1577830891900"},
	}
	verifyHeaders(t, hdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":     "OtherTransaction/Go/hello",
				"guid":     internal.MatchAnything,
				"priority": 0.0,
				"sampled":  internal.MatchAnything,
				"traceId":  internal.MatchAnything,
			},
		},
	})
}

func TestW3CTraceHeadersNoSamplingAlwaysOn(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		distributedTracingReplyFields(reply)
		reply.SetSampleNothing()
	}
	app := testApp(replyfn, enableW3COnlyNotSampledAlwaysOn, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-00"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456-9566c74d10d1e2c6-52fdfc072182654f-0-2-1577830891900"},
	}
	verifyHeaders(t, hdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":     "OtherTransaction/Go/hello",
				"guid":     internal.MatchAnything,
				"priority": 2.0,
				"sampled":  internal.MatchAnything,
				"traceId":  internal.MatchAnything,
			},
		},
	})

}

func TestW3CTraceHeadersNoSamplingAlwaysOff(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		distributedTracingReplyFields(reply)
		reply.SetSampleNothing()
	}
	app := testApp(replyfn, enableW3COnlyNotSampledAlwaysOff, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-00"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456-9566c74d10d1e2c6-52fdfc072182654f-0-0-1577830891900"},
	}
	verifyHeaders(t, hdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":     "OtherTransaction/Go/hello",
				"guid":     internal.MatchAnything,
				"priority": 0.0,
				"sampled":  internal.MatchAnything,
				"traceId":  internal.MatchAnything,
			},
		},
	})

}

var acceptAndSendDT = []internal.WantMetric{
	{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
	{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
	{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
	{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
	{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: nil},
	{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	{Name: "DurationByCaller/App/1349956/41346604/HTTP/all", Scope: "", Forced: false, Data: nil},
	{Name: "DurationByCaller/App/1349956/41346604/HTTP/allOther", Scope: "", Forced: false, Data: nil},
	{Name: "TransportDuration/App/1349956/41346604/HTTP/all", Scope: "", Forced: false, Data: nil},
	{Name: "TransportDuration/App/1349956/41346604/HTTP/allOther", Scope: "", Forced: false, Data: nil},
}

func TestW3CTraceHeadersNoMatchingNREntry(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableW3COnly, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	hdrs.Set(DistributedTraceW3CTraceParentHeader,
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	hdrs.Set(DistributedTraceW3CTraceStateHeader,
		"99999@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	outgoingHdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(outgoingHdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-4bf92f3577b34da6a3ce929d0e0e4736-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456-9566c74d10d1e2c6-52fdfc072182654f-1-1.437714-1577830891900,99999@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277"},
	}
	verifyHeaders(t, outgoingHdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/TraceState/NoNrEntry", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCallerWithTransport...))
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":             "OtherTransaction/Go/hello",
				"transaction.name": "OtherTransaction/Go/hello",
				"sampled":          true,
				"priority":         internal.MatchAnything,
				"category":         "generic",
				"parentId":         "00f067aa0ba902b7",
				"nr.entryPoint":    true,
				"guid":             "9566c74d10d1e2c6",
				"transactionId":    "52fdfc072182654f",
				"traceId":          "4bf92f3577b34da6a3ce929d0e0e4736",
				"tracingVendors":   "99999@nr",
			},
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"parent.transportType": "HTTP",
			},
		},
	})

}

func TestW3CTraceHeadersRoundTrip(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableW3COnly, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	hdrs.Set(DistributedTraceW3CTraceParentHeader,
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	hdrs.Set(DistributedTraceW3CTraceStateHeader,
		"123@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	outgoingHdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(outgoingHdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-4bf92f3577b34da6a3ce929d0e0e4736-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456-9566c74d10d1e2c6-52fdfc072182654f-1-0.24689-1577830891900"},
	}
	verifyHeaders(t, outgoingHdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, acceptAndSendDT)

}

func TestW3CTraceHeadersDuplicateTraceState(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableW3COnly, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	hdrs.Set(DistributedTraceW3CTraceParentHeader,
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	hdrs.Set(DistributedTraceW3CTraceStateHeader,
		"123@nr=0-0-1349956-41346604-27ddd2d8890283b4-b28be285632bbc0a-1-0.246890-1569367663277,congo=congosSecondPosition,rojo=rojosFirstPosition,123@nr=0-0-1349956-41346604-aaaaaaaaaaaaaaaa-b28be285632bbc0a-1-0.246890-1569367663277")
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	outgoingHdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(outgoingHdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-4bf92f3577b34da6a3ce929d0e0e4736-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456-9566c74d10d1e2c6-52fdfc072182654f-1-0.24689-1577830891900,congo=congosSecondPosition,rojo=rojosFirstPosition"},
	}
	verifyHeaders(t, outgoingHdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, acceptAndSendDT)

}

func TestW3CTraceHeadersSpansDisabledSampledTrue(t *testing.T) {
	app := testApp(distributedTracingReplyFieldsSpansDisabled, enableW3COnly, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456--52fdfc072182654f-1-1.437714-1577830891900"},
	}
	verifyHeaders(t, hdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))

}

func TestW3CTraceHeadersSpansDisabledSampledFalse(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		distributedTracingReplyFieldsSpansDisabled(reply)
		reply.SetSampleNothing()
	}
	app := testApp(replyfn, enableW3COnly, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-00"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456--52fdfc072182654f-0-0.437714-1577830891900"},
	}
	verifyHeaders(t, hdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))

}

func TestW3CTraceHeadersSpansDisabledWithTraceState(t *testing.T) {
	app := testApp(distributedTracingReplyFieldsSpansDisabled, enableW3COnly, t)
	txn := app.StartTransaction("hello")

	originalTraceParent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	originalTraceState := "rojo=00f067aa0ba902b7"
	incomingHdrs := http.Header{}
	incomingHdrs.Set(DistributedTraceW3CTraceParentHeader, originalTraceParent)
	incomingHdrs.Set(DistributedTraceW3CTraceStateHeader, originalTraceState)
	txn.AcceptDistributedTraceHeaders(TransportHTTP, incomingHdrs)

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-4bf92f3577b34da6a3ce929d0e0e4736-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456--52fdfc072182654f-1-1.437714-1577830891900," + originalTraceState},
	}
	verifyHeaders(t, hdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/TraceState/NoNrEntry", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCallerWithTransport...))
}

func TestW3CTraceHeadersTxnEventsDisabled(t *testing.T) {
	cfgfn := func(cfg *Config) {
		enableW3COnly(cfg)
		cfg.TransactionEvents.Enabled = false
	}
	app := testApp(distributedTracingReplyFields, cfgfn, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456-9566c74d10d1e2c6--1-1.437714-1577830891900"},
	}
	verifyHeaders(t, hdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
}

func TestW3CTraceHeadersTxnAndSpanEventsDisabledSampledTrue(t *testing.T) {
	cfgfn := func(cfg *Config) {
		enableW3COnly(cfg)
		cfg.TransactionEvents.Enabled = false
	}
	app := testApp(distributedTracingReplyFieldsSpansDisabled, cfgfn, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456---1-1.437714-1577830891900"},
	}
	verifyHeaders(t, hdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
}

func TestW3CTraceHeadersTxnAndSpanEventsDisabledSampledFalse(t *testing.T) {
	cfgfn := func(cfg *Config) {
		enableW3COnly(cfg)
		cfg.TransactionEvents.Enabled = false
	}
	app := testApp(distributedTracingReplyFieldsSpansDisabled, cfgfn, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456---1-1.437714-1577830891900"},
	}
	verifyHeaders(t, hdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
}

func TestW3CTraceHeadersNoTraceState(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableW3COnly, t)
	txn := app.StartTransaction("hello")

	originalTraceParent := "00-12345678901234567890123456789012-1234567890123456-01"
	incomingHdrs := http.Header{}
	incomingHdrs.Set(DistributedTraceW3CTraceParentHeader, originalTraceParent)
	txn.AcceptDistributedTraceHeaders(TransportHTTP, incomingHdrs)

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-12345678901234567890123456789012-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456-9566c74d10d1e2c6-52fdfc072182654f-1-1.437714-1577830891900"},
	}
	verifyHeaders(t, hdrs, expected)

	txn.End()
	app.expectNoLoggedErrors(t)

}

// Based on test_traceparent_trace_id_all_zero in
// https://github.com/w3c/trace-context/blob/3d02cfc15778ef850df9bc4e9d2740a4a2627fd5/test/test.py
func TestW3CTraceHeadersInvalidTraceID(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableW3COnly, t)
	txn := app.StartTransaction("hello")

	originalTraceParent := "00-00000000000000000000000000000000-1234567890123456-01"
	incomingHdrs := http.Header{}
	incomingHdrs.Set(DistributedTraceW3CTraceParentHeader, originalTraceParent)
	txn.AcceptDistributedTraceHeaders(TransportHTTP, incomingHdrs)

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456-9566c74d10d1e2c6-52fdfc072182654f-1-1.437714-1577830891900"},
	}
	verifyHeaders(t, hdrs, expected)

	txn.End()
	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": "invalid TraceParent trace ID",
	})
}

// Based on test_traceparent_parent_id_all_zero in
// https://github.com/w3c/trace-context/blob/3d02cfc15778ef850df9bc4e9d2740a4a2627fd5/test/test.py
func TestW3CTraceHeadersInvalidParentID(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableW3COnly, t)
	txn := app.StartTransaction("hello")

	originalTraceParent := "00-12345678901234567890123456789012-0000000000000000-01"
	incomingHdrs := http.Header{}
	incomingHdrs.Set(DistributedTraceW3CTraceParentHeader, originalTraceParent)
	txn.AcceptDistributedTraceHeaders(TransportHTTP, incomingHdrs)

	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)

	expected := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-01"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456-9566c74d10d1e2c6-52fdfc072182654f-1-1.437714-1577830891900"},
	}
	verifyHeaders(t, hdrs, expected)

	txn.End()
	app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
		"reason": "invalid TraceParent parent ID",
	})
}

// Based on test_traceparent_version_0x00, test_traceparent_version_0xcc, test_traceparent_version_0xff in
// https://github.com/w3c/trace-context/blob/3d02cfc15778ef850df9bc4e9d2740a4a2627fd5/test/test.py
func TestW3CTraceHeadersFutureVersion(t *testing.T) {
	cases := map[string]string{
		"00-12345678901234567890123456789012-1234567890123456-01-what-the-future-will-be-like": "invalid TraceParent flags for this version",
		"cc-12345678901234567890123456789012-1234567890123456-01":                              "",
		"cc-12345678901234567890123456789012-1234567890123456-01-what-the-future-will-be-like": "",
		"cc-12345678901234567890123456789012-1234567890123456-01.what-the-future-will-be-like": "invalid number of TraceParent entries",
		"ff-12345678901234567890123456789012-1234567890123456-01":                              "invalid TraceParent flags for this version",
	}
	for testCase, failureMessage := range cases {
		app := testApp(distributedTracingReplyFields, enableW3COnly, t)
		txn := app.StartTransaction("hello")

		originalTraceParent := testCase
		incomingHdrs := http.Header{}
		incomingHdrs.Set(DistributedTraceW3CTraceParentHeader, originalTraceParent)
		txn.AcceptDistributedTraceHeaders(TransportHTTP, incomingHdrs)

		outgoingHdrs := http.Header{}
		txn.InsertDistributedTraceHeaders(outgoingHdrs)

		if len(outgoingHdrs) != 2 {
			t.Log("Not all headers present:", outgoingHdrs)
			t.Fail()
		}
		expected := "00-12345678901234567890123456789012-9566c74d10d1e2c6-01"
		if failureMessage != "" {
			if outgoingHdrs.Get(DistributedTraceW3CTraceParentHeader) == expected {
				t.Errorf("Invalid TraceParent header resulting from %s", testCase)
			}

		} else {
			if outgoingHdrs.Get(DistributedTraceW3CTraceParentHeader) != expected {
				t.Errorf("Invalid TraceParent header resulting from %s", testCase)
			}
		}

		txn.End()
		if failureMessage != "" {
			app.expectSingleLoggedError(t, "unable to accept trace payload", map[string]interface{}{
				"reason": failureMessage,
			})
		} else {
			app.expectNoLoggedErrors(t)
		}
	}
}

func TestW3CTraceParentWithoutTraceContext(t *testing.T) {
	traceparent := "00-050c91b77efca9b0ef38b30c182355ce-560ccffb087d1906-01"

	app := testApp(distributedTracingReplyFields, enableW3COnly, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{}
	hdrs.Set(DistributedTraceW3CTraceParentHeader, traceparent)
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	txn.End()

	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":                 "OtherTransaction/Go/hello",
			"traceId":              "050c91b77efca9b0ef38b30c182355ce",
			"parentSpanId":         "560ccffb087d1906",
			"guid":                 internal.MatchAnything,
			"sampled":              internal.MatchAnything,
			"priority":             internal.MatchAnything,
			"parent.transportType": "HTTP",
		},
	}})
}

func TestDistributedTraceInteroperabilityErrorFallbacks(t *testing.T) {
	// Test what happens in varying cases when both w3c and newrelic headers
	// are found

	// parent.type  = "App"
	// parentSpanId = "5f474d64b9cc9b2a"
	// traceId      = "3221bf09aa0bcf0d3221bf09aa0bcf0d"
	newrelicHdr := `{
		   "v": [0,1],
		   "d": {
		     "ty": "App",
		     "ac": "123",
		     "ap": "51424",
		     "id": "5f474d64b9cc9b2a",
		     "tr": "3221bf09aa0bcf0d3221bf09aa0bcf0d",
		     "pr": 0.1234,
		     "sa": true,
		     "ti": 1482959525577,
		     "tx": "27856f70d3d314b7"
		   }
		}`
	// parentSpanId = "560ccffb087d1906"
	// traceId      = "050c91b77efca9b0ef38b30c182355ce"
	traceparentHdr := "00-050c91b77efca9b0ef38b30c182355ce-560ccffb087d1906-01"
	// parent.type  = "Browser"
	tracestateHdr := "123@nr=0-1-123-456-1234567890123456-6543210987654321-0-0.24689-0"

	testcases := []struct {
		name          string
		traceparent   string
		tracestate    string
		newrelic      string
		expIntrinsics map[string]interface{}
	}{
		{
			name:        "w3c present, newrelic absent, failure to parse traceparent",
			traceparent: "garbage",
			tracestate:  tracestateHdr,
			newrelic:    "",
			expIntrinsics: map[string]interface{}{
				"guid":     internal.MatchAnything,
				"priority": internal.MatchAnything,
				"sampled":  internal.MatchAnything,
				"name":     internal.MatchAnything,
				"traceId":  "52fdfc072182654f163f5f0f9a621d72", // randomly generated
			},
		},
		{
			name:        "w3c present, newrelic absent, failure to parse tracestate",
			traceparent: traceparentHdr,
			tracestate:  "123@nr=garbage",
			newrelic:    "",
			expIntrinsics: map[string]interface{}{
				"guid":                 internal.MatchAnything,
				"priority":             internal.MatchAnything,
				"sampled":              internal.MatchAnything,
				"name":                 internal.MatchAnything,
				"parent.transportType": internal.MatchAnything,
				"parentSpanId":         "560ccffb087d1906",                 // from traceparent header
				"traceId":              "050c91b77efca9b0ef38b30c182355ce", // from traceparent header
			},
		},
		{
			name:        "w3c present, newrelic present, failure to parse traceparent",
			traceparent: "garbage",
			tracestate:  tracestateHdr,
			newrelic:    newrelicHdr,
			expIntrinsics: map[string]interface{}{
				"guid":     internal.MatchAnything,
				"priority": internal.MatchAnything,
				"sampled":  internal.MatchAnything,
				"name":     internal.MatchAnything,
				"traceId":  "52fdfc072182654f163f5f0f9a621d72", // randomly generated
			},
		},
		{
			name:        "w3c present, newrelic present, failure to parse tracestate",
			traceparent: traceparentHdr,
			tracestate:  "123@nr=garbage",
			newrelic:    newrelicHdr,
			expIntrinsics: map[string]interface{}{
				"guid":                 internal.MatchAnything,
				"priority":             internal.MatchAnything,
				"sampled":              internal.MatchAnything,
				"name":                 internal.MatchAnything,
				"parent.transportType": internal.MatchAnything,
				"parentSpanId":         "560ccffb087d1906",                 // from traceparent header
				"traceId":              "050c91b77efca9b0ef38b30c182355ce", // from traceparent header
			},
		},
		{
			name:        "w3c present, newrelic present",
			traceparent: traceparentHdr,
			tracestate:  tracestateHdr,
			newrelic:    newrelicHdr,
			expIntrinsics: map[string]interface{}{
				"parent.app":               internal.MatchAnything,
				"parent.transportDuration": internal.MatchAnything,
				"guid":                     internal.MatchAnything,
				"priority":                 internal.MatchAnything,
				"sampled":                  internal.MatchAnything,
				"parent.account":           internal.MatchAnything,
				"parentId":                 internal.MatchAnything,
				"name":                     internal.MatchAnything,
				"parent.transportType":     internal.MatchAnything,
				"parent.type":              "Browser",                          // from tracestate header
				"parentSpanId":             "560ccffb087d1906",                 // from traceparent header
				"traceId":                  "050c91b77efca9b0ef38b30c182355ce", // from traceparent header
			},
		},
		{
			name:        "w3c absent, newrelic present",
			traceparent: "",
			tracestate:  "",
			newrelic:    newrelicHdr,
			expIntrinsics: map[string]interface{}{
				"parent.app":               internal.MatchAnything,
				"parent.transportDuration": internal.MatchAnything,
				"guid":                     internal.MatchAnything,
				"priority":                 internal.MatchAnything,
				"sampled":                  internal.MatchAnything,
				"parent.account":           internal.MatchAnything,
				"parentId":                 internal.MatchAnything,
				"name":                     internal.MatchAnything,
				"parent.transportType":     internal.MatchAnything,
				"parent.type":              "App",                              // from newrelic header
				"parentSpanId":             "5f474d64b9cc9b2a",                 // from newrelic header
				"traceId":                  "3221bf09aa0bcf0d3221bf09aa0bcf0d", // from newrelic header
			},
		},
		{
			name:        "w3c absent, newrelic absent",
			traceparent: "",
			tracestate:  "",
			newrelic:    "",
			expIntrinsics: map[string]interface{}{
				"guid":     internal.MatchAnything,
				"priority": internal.MatchAnything,
				"sampled":  internal.MatchAnything,
				"name":     internal.MatchAnything,
				"traceId":  "52fdfc072182654f163f5f0f9a621d72", // randomly generated
			},
		},
	}

	addHdr := func(hdrs http.Header, key, val string) {
		if val != "" {
			hdrs.Add(key, val)
		}
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
			txn := app.StartTransaction("hello")

			hdrs := http.Header{}
			addHdr(hdrs, DistributedTraceW3CTraceParentHeader, tc.traceparent)
			addHdr(hdrs, DistributedTraceW3CTraceStateHeader, tc.tracestate)
			addHdr(hdrs, DistributedTraceNewRelicHeader, tc.newrelic)

			txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
			txn.End()

			app.ExpectTxnEvents(t, []internal.WantEvent{{
				Intrinsics: tc.expIntrinsics,
			}})
		})
	}
}

func TestW3CTraceStateMultipleHeaders(t *testing.T) {
	traceparent := "00-050c91b77efca9b0ef38b30c182355ce-560ccffb087d1906-01"
	nrstatekey := "123@nr=0-0-123-456-1234567890123456-6543210987654321-1-0.24689-0"
	testcases := []struct {
		firstheader  string
		secondheader string
	}{
		{firstheader: "a=1,b=2", secondheader: nrstatekey},
		{firstheader: "a=1", secondheader: "b=2," + nrstatekey},
		{firstheader: "a=1", secondheader: nrstatekey + ",b=2"},
	}

	for _, tc := range testcases {
		t.Run(tc.firstheader+"_"+tc.secondheader, func(t *testing.T) {
			app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
			txn := app.StartTransaction("hello")

			hdrs := http.Header{}
			hdrs.Add(DistributedTraceW3CTraceParentHeader, traceparent)
			hdrs.Add(DistributedTraceW3CTraceStateHeader, tc.firstheader)
			hdrs.Add(DistributedTraceW3CTraceStateHeader, tc.secondheader)

			txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
			txn.End()
			app.expectNoLoggedErrors(t)

			app.ExpectSpanEvents(t, []internal.WantEvent{{
				Intrinsics: map[string]interface{}{
					"category":         "generic",
					"guid":             "9566c74d10d1e2c6",
					"name":             "OtherTransaction/Go/hello",
					"transaction.name": "OtherTransaction/Go/hello",
					"nr.entryPoint":    true,
					"parentId":         "560ccffb087d1906",
					"priority":         internal.MatchAnything,
					"sampled":          true,
					"traceId":          "050c91b77efca9b0ef38b30c182355ce",
					"tracingVendors":   "a,b", // ensures both headers read
					"transactionId":    "52fdfc072182654f",
					"trustedParentId":  "1234567890123456",
				},
			}})
		})
	}
}

func TestW3CTraceIDLengths(t *testing.T) {
	// Test that if the agent received an inbound traceId that is less than 32
	// characters, the traceId included in an outbound payload must be
	// left-padded with zeros.  If it is too long, we chop off from the left.
	testcases := []string{
		"3221bf09aa0bcf0d", // too short
		"00000000000000000000000000000000000000000000003221bf09aa0bcf0d", // too long
	}

	for _, teststr := range testcases {
		app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
		txn := app.StartTransaction("hello")

		in := http.Header{}
		in.Add(DistributedTraceNewRelicHeader,
			fmt.Sprintf(`{
		   "v": [0,1],
		   "d": {
		     "ty": "App",
		     "ac": "123",
		     "ap": "51424",
		     "id": "5f474d64b9cc9b2a",
		     "tr": "%s",
		     "pr": 0.1234,
		     "sa": true,
		     "ti": 1482959525577,
		     "tx": "27856f70d3d314b7"
		   }
		}`, teststr))
		txn.AcceptDistributedTraceHeaders(TransportHTTP, in)
		app.expectNoLoggedErrors(t)

		out := http.Header{}
		txn.InsertDistributedTraceHeaders(out)
		traceparent := out.Get(DistributedTraceW3CTraceParentHeader)
		expected := "00-00000000000000003221bf09aa0bcf0d-9566c74d10d1e2c6-01"
		if traceparent != expected {
			t.Errorf("incorrect traceparent header: expect=%s actual=%s", expected, traceparent)
		}
	}
}

func TestW3CTraceNotSampledOutboundHeaders(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		distributedTracingReplyFields(reply)
		reply.SetSampleNothing()
	}
	app := testApp(replyfn, enableW3COnly, t)
	txn := app.StartTransaction("hello")

	hdrs := make(http.Header)
	txn.InsertDistributedTraceHeaders(hdrs)
	if !reflect.DeepEqual(hdrs, http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-00"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=0-0-123-456-9566c74d10d1e2c6-52fdfc072182654f-0-0.437714-1577830891900"},
	}) {
		t.Error(hdrs)
	}

	txn.End()
	app.expectNoLoggedErrors(t)
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/TraceContext/Create/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
}

func TestW3CTraceStateInvalidNrEntry(t *testing.T) {
	// If the tracestate header has fewer entries (separated by '-') than
	// expected, make sure the correct Supportability metrics are created
	replyfn := func(reply *internal.ConnectReply) {
		distributedTracingReplyFields(reply)
		reply.SetSampleNothing()
	}
	app := testApp(replyfn, enableW3COnly, t)
	txn := app.StartTransaction("hello")

	hdrs := http.Header{
		DistributedTraceW3CTraceParentHeader: []string{"00-52fdfc072182654f163f5f0f9a621d72-9566c74d10d1e2c6-00"},
		DistributedTraceW3CTraceStateHeader:  []string{"123@nr=garbage"},
	}
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)

	txn.End()
	app.expectNoLoggedErrors(t)
	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/TraceContext/Accept/Success", Scope: "", Forced: true, Data: nil},
		{Name: "Supportability/TraceContext/TraceState/InvalidNrEntry", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCallerWithTransport...))
}

func TestUpperCaseTraceIDReceived(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		distributedTracingReplyFields(reply)
		reply.SetSampleNothing()
	}
	app := testApp(replyfn, enableBetterCAT, t)
	txn := app.StartTransaction("hello")
	originalTraceID := "85D7FA2DD1B66D6C" // Legacy .NET agents may send uppercase trace IDs
	incoming := payload{
		Type:              callerTypeApp,
		App:               "123",
		Account:           "456",
		TransactionID:     "1a2b3c",
		ID:                "0f9a8d",
		TracedID:          originalTraceID,
		TrustedAccountKey: "123",
	}
	hdrs := http.Header{
		DistributedTraceNewRelicHeader: []string{incoming.NRText()},
	}
	txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
	outgoing := http.Header{}
	txn.InsertDistributedTraceHeaders(outgoing)

	// Verify the NR header uses the original (short, uppercase) Trace ID
	p := payloadFieldsFromHeaders(t, outgoing)
	s, ok := p.Data["tr"].(string)
	if !ok || s != originalTraceID {
		t.Error("Invalid NewRelic header trace ID", p.Data)
	}

	// Verify that the TraceParent header uses padded and lower-cased Trace ID
	ts := outgoing.Get(DistributedTraceW3CTraceParentHeader)
	expected := "0000000000000000" + strings.ToLower(originalTraceID)
	if ts != "00-"+expected+"-9566c74d10d1e2c6-00" {
		t.Error("Invalid TraceParent header", ts)
	}
}

func TestDTHeadersAddedTwice(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("hello")

	// make sure outbound headers are not added twice
	assertHdrs := func(hdrs http.Header) {
		if h := hdrs[DistributedTraceNewRelicHeader]; len(h) != 1 {
			t.Errorf("incorrect number of newrelic headers: %#v", h)
		}
		if h := hdrs[DistributedTraceW3CTraceParentHeader]; len(h) != 1 {
			t.Errorf("incorrect number of traceparent headers: %#v", h)
		}
		if h := hdrs[DistributedTraceW3CTraceStateHeader]; len(h) != 1 {
			t.Errorf("incorrect number of tracestate headers: %#v", h)
		}
	}

	// Using StartExternalSegment
	req, _ := http.NewRequest("GET", "https://www.something.com/path/zip/zap?secret=ssshhh", nil)
	s := StartExternalSegment(txn, req)
	s.End()
	s = StartExternalSegment(txn, req)
	s.End()
	app.expectNoLoggedErrors(t)
	assertHdrs(req.Header)

	// Using InsertDistributedTraceHeaders
	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)
	txn.InsertDistributedTraceHeaders(hdrs)
	assertHdrs(hdrs)
}

func TestW3CHeaderCases(t *testing.T) {
	traceparent := "00-050c91b77efca9b0ef38b30c182355ce-560ccffb087d1906-01"
	tracestate := "123@nr=0-0-123-456-1234567890123456-6543210987654321-0-0.24689-0"

	testcases := []struct {
		parent string
		state  string
	}{
		{parent: "traceparent", state: "Tracestate"},
		{parent: "Traceparent", state: "Tracestate"},
		{parent: "TraceParent", state: "Tracestate"},
		{parent: "Traceparent", state: "tracestate"},
		{parent: "Traceparent", state: "Tracestate"},
		{parent: "Traceparent", state: "TraceState"},
	}

	for _, tc := range testcases {
		t.Run(tc.parent+"-"+tc.state, func(t *testing.T) {
			app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
			txn := app.StartTransaction("hello")

			hdrs := http.Header{}
			hdrs.Set(tc.parent, traceparent)
			hdrs.Set(tc.state, tracestate)

			txn.AcceptDistributedTraceHeaders(TransportHTTP, hdrs)
			txn.End()

			app.ExpectMetricsPresent(t, []internal.WantMetric{
				// presence of this metric indicates that accepting succeeded
				{Name: "DurationByCaller/App/123/456/HTTP/all"},
			})
		})
	}
}
