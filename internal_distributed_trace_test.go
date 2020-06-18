// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/crossagent"
)

type PayloadTest struct {
	V *[2]int                `json:"v,omitempty"`
	D map[string]interface{} `json:"d,omitempty"`
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
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: singleCount},
	}
)

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
	app.ExpectMetrics(t, distributedTracingSuccessMetrics)
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
	app.ExpectMetrics(t, distributedTracingSuccessMetrics)
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
	app.ExpectMetrics(t, distributedTracingSuccessMetrics)
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
	app.ExpectMetrics(t, distributedTracingSuccessMetrics)
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

	if err != errTrustedAccountKey {
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
	if err != errTrustedAccountKey {
		t.Error("Expected ErrTrustedAccountKey from mismatched trustkeys", err)
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
	if err != errTrustedAccountKey {
		t.Error("Expected ErrTrustedAccountKey from mismatched trustkeys", err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}
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
)

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

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/Ignored/Null", Scope: "", Forced: true, Data: singleCount},
	}, backgroundUnknownCaller...))
}

func TestNoticeErrorPayload(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)

	txn := app.StartTransaction("hello", nil, nil)
	txn.NoticeError(errors.New("oh no"))

	err := txn.End()
	if nil != err {
		t.Error(err)
	}

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

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/ParseException", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
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

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/ParseException", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
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

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/ParseException", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
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

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/AcceptPayload/ParseException", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
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
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},

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
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
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

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
}

func isZeroValue(x interface{}) bool {
	// https://stackoverflow.com/questions/13901819/quick-way-to-detect-empty-values-via-reflection-in-go
	return nil == x || x == reflect.Zero(reflect.TypeOf(x)).Interface()
}

func testPayloadFieldsPresent(t *testing.T, p DistributedTracePayload, keys ...string) {
	out := struct {
		Version []int                  `json:"v"`
		Data    map[string]interface{} `json:"d"`
	}{}
	if err := json.Unmarshal([]byte(p.Text()), &out); nil != err {
		t.Fatal("unable to unmarshal payload Text", err)
	}
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
	txn := app.StartTransaction("hello", nil, nil)

	p := txn.CreateDistributedTracePayload()

	testPayloadFieldsPresent(t, p, "ty", "ac", "ap", "tr", "ti")

	err := txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
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

	if nil != payloadData.D["tk"] {
		t.Log("Did not expect trust key (tk) to be there")
		t.Log(p.Text())
		t.Fail()
	}

	err := txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
	}, backgroundUnknownCaller...))
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

	testPayloadFieldsPresent(t, p, "tk")

	err := txn.End()
	if nil != err {
		t.Error(err)
	}

	app.ExpectMetrics(t, append([]internal.WantMetric{
		{Name: "Supportability/DistributedTrace/CreatePayload/Success", Scope: "", Forced: true, Data: nil},
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
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, p)
	if nil != err {
		t.Error(err)
	}

	payload := txn.CreateDistributedTracePayload()

	testPayloadFieldsPresent(t, payload,
		"ty", "ac", "ap", "tr", "ti", "pr", "sa")

	err = txn.End()
	if nil != err {
		t.Error(err)
	}
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
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, p)
	if nil != err {
		t.Error(err)
	}

	payload := txn.CreateDistributedTracePayload()
	testPayloadFieldsPresent(t, payload,
		"ty", "ac", "ap", "id", "tr", "ti", "pr", "sa")

	err = txn.End()
	if nil != err {
		t.Error(err)
	}
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
	switch transport {
	case TransportHTTP.name:
		return TransportHTTP
	case TransportHTTPS.name:
		return TransportHTTPS
	case TransportKafka.name:
		return TransportKafka
	case TransportJMS.name:
		return TransportJMS
	case TransportIronMQ.name:
		return TransportIronMQ
	case TransportAMQP.name:
		return TransportAMQP
	case TransportQueue.name:
		return TransportQueue
	case TransportOther.name:
		return TransportOther
	default:
		return TransportUnknown
	}
}

func runDistributedTraceCrossAgentTestcase(tst *testing.T, tc distributedTraceTestcase, extraAsserts func(expectApp, internal.Validator)) {
	t := internal.ExtendValidator(tst, "test="+tc.TestName)
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

	}, configCallback, tst)

	txn := app.StartTransaction("hello", nil, nil)
	if tc.WebTransaction {
		txn.SetWebRequest(nil)
	}

	// If the tests wants us to have an error, give 'em an error
	if tc.RaisesException {
		txn.NoticeError(errors.New("my error message"))
	}

	// If there are no inbound payloads, invoke Accept on an empty inbound payload.
	if nil == tc.InboundPayloads {
		txn.AcceptDistributedTracePayload(getTransport(tc.TransportType), nil)
	}

	for _, value := range tc.InboundPayloads {
		// Note that the error return value is not tested here because
		// some of the tests are intentionally errors.
		txn.AcceptDistributedTracePayload(getTransport(tc.TransportType), string(value))
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

	// Add extra fields that are not listed in the JSON file so that we can
	// always do exact intrinsic set match.

	extraTxnFields := &fieldExpectations{Expected: []string{"name"}}
	if tc.WebTransaction {
		extraTxnFields.Expected = append(extraTxnFields.Expected, "nr.apdexPerfZone")
	}

	extraSpanFields := &fieldExpectations{
		Expected: []string{"name", "category", "nr.entryPoint"},
	}

	// There is a single test with an error (named "exception"), so these
	// error expectations can be hard coded. TODO: Move some of these.
	// fields into the cross agent tests.
	extraErrorFields := &fieldExpectations{
		Expected: []string{"parent.type", "parent.account", "parent.app",
			"parent.transportType", "error.message", "transactionName",
			"parent.transportDuration", "error.class"},
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

func assertTestCaseOutboundPayload(expect fieldExpectations, t internal.Validator, actual string) {
	type outboundTestcase struct {
		Version [2]uint                `json:"v"`
		Data    map[string]interface{} `json:"d"`
	}
	var actualPayload outboundTestcase
	err := json.Unmarshal([]byte(actual), &actualPayload)
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
	payload := makePayload(app, nil)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	if err != errInboundPayloadDTDisabled {
		t.Fatal("we expected an error with DT disabled", err)
	}
	err = txn.End()
	if nil != err {
		t.Error(err)
	}

	// ensure no span events created
	app.ExpectSpanEvents(t, nil)
}

func TestCreatePayloadAppNotConnected(t *testing.T) {
	// Test that an app which isn't connected does not create distributed
	// trace payloads.
	app := testApp(nil, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	payload := txn.CreateDistributedTracePayload()
	if payload.Text() != "" || payload.HTTPSafe() != "" {
		t.Error(payload.Text(), payload.HTTPSafe())
	}
}
func TestCreatePayloadReplyMissingTrustKey(t *testing.T) {
	// Test that an app whose reply is missing the trust key does not create
	// distributed trace payloads.
	app := testApp(func(reply *internal.ConnectReply) {
		distributedTracingReplyFields(reply)
		reply.TrustedAccountKey = ""
	}, enableBetterCAT, t)
	txn := app.StartTransaction("hello", nil, nil)
	payload := txn.CreateDistributedTracePayload()
	if payload.Text() != "" || payload.HTTPSafe() != "" {
		t.Error(payload.Text(), payload.HTTPSafe())
	}
}

func TestAcceptPayloadAppNotConnected(t *testing.T) {
	// Test that an app which isn't connected does not accept distributed
	// trace payloads.
	app := testApp(nil, enableBetterCAT, t)
	payload := testApp(distributedTracingReplyFields, enableBetterCAT, t).
		StartTransaction("name", nil, nil).
		CreateDistributedTracePayload()
	if payload.Text() == "" {
		t.Fatal(payload)
	}
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectMetrics(t, backgroundUnknownCaller)
}

func TestAcceptPayloadReplyMissingTrustKey(t *testing.T) {
	// Test that an app whose reply is missing a trust key does not accept
	// distributed trace payloads.
	app := testApp(func(reply *internal.ConnectReply) {
		distributedTracingReplyFields(reply)
		reply.TrustedAccountKey = ""
	}, enableBetterCAT, t)
	payload := testApp(distributedTracingReplyFields, enableBetterCAT, t).
		StartTransaction("name", nil, nil).
		CreateDistributedTracePayload()
	if payload.Text() == "" {
		t.Fatal(payload)
	}
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectMetrics(t, backgroundUnknownCaller)
}
