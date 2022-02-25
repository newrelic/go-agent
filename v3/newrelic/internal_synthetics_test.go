// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/cat"
)

// This collection of top-level tests affirms, for all possible combinations of
// Old CAT, BetterCAT, and Synthetics, that when an inbound request contains a
// synthetics header, the subsequent outbound request propagates that synthetics
// header.  Synthetics uses an obfuscated JSON header, so this test requires a
// really particular set of values, e.g. rrrrrrr-rrrr-1234-rrrr-rrrrrrrrrrrr.

var (
	trustedAccounts = func() map[int]struct{} {
		ta := make(map[int]struct{})
		ta[1] = struct{}{}   // Trust account 1, from syntheticsConnectReplyFn.
		ta[444] = struct{}{} // Trust account 444, from syntheticsHeader.
		return ta
	}()

	syntheticsConnectReplyFn = func(reply *internal.ConnectReply) {
		reply.EncodingKey = "1234567890123456789012345678901234567890"
		reply.CrossProcessID = "1#1"
		reply.TrustedAccounts = trustedAccounts
	}
)

func inboundSyntheticsRequestBuilder(oldCatEnabled bool, betterCatEnabled bool) *http.Request {
	cfgFn := func(cfg *Config) {
		cfg.CrossApplicationTracer.Enabled = oldCatEnabled
		cfg.DistributedTracer.Enabled = betterCatEnabled
	}
	app := testApp(syntheticsConnectReplyFn, cfgFn, nil)
	txn := app.StartTransaction("requester")
	req, err := http.NewRequest("GET", "newrelic.com", nil)
	if nil != err {
		panic(err)
	}

	req.Header.Add(
		"X-NewRelic-Synthetics",
		"agMfAAECGxpLQkNAQUZHG0VKS0IcAwEHARtFSktCHEBBRkdERUpLQkNAQRYZFF1SU1pbWFkZX1xdUhQBAwEHGV9cXVIUWltYWV5fXF1SU1pbEB8WWFtaVVRdXB9eWVhbGgkLAwUfXllYWxpVVF1cX15ZWFtaVVQSbA==")

	StartExternalSegment(txn, req)

	if betterCatEnabled || !oldCatEnabled {
		if cat.NewRelicIDName == req.Header.Get(cat.NewRelicIDName) {
			panic("Header contains old cat header NewRelicIDName: " + req.Header.Get(cat.NewRelicIDName))
		}
		if cat.NewRelicTxnName == req.Header.Get(cat.NewRelicTxnName) {
			panic("Header contains old cat header NewRelicTxnName: " + req.Header.Get(cat.NewRelicTxnName))
		}
	}

	if oldCatEnabled {
		if "" == req.Header.Get(cat.NewRelicIDName) {
			panic("Missing old cat header NewRelicIDName: " + req.Header.Get(cat.NewRelicIDName))
		}
		if "" == req.Header.Get(cat.NewRelicTxnName) {
			panic("Missing old cat header NewRelicTxnName: " + req.Header.Get(cat.NewRelicTxnName))
		}
	}

	if "" == req.Header.Get(cat.NewRelicSyntheticsName) {
		panic("missing synthetics header NewRelicSyntheticsName: " + req.Header.Get(cat.NewRelicSyntheticsName))
	}

	return req
}

func TestSyntheticsOldCAT(t *testing.T) {
	cfgFn := func(cfg *Config) {
		cfg.CrossApplicationTracer.Enabled = true
		cfg.DistributedTracer.Enabled = false
	}
	app := testApp(syntheticsConnectReplyFn, cfgFn, t)
	clientTxn := app.StartTransaction("helloOldCAT")
	clientTxn.SetWebRequestHTTP(inboundSyntheticsRequestBuilder(true, false))

	req, err := http.NewRequest("GET", "newrelic.com", nil)

	if nil != err {
		panic(err)
	}

	StartExternalSegment(clientTxn, req)
	clientTxn.End()

	if "" == req.Header.Get(cat.NewRelicSyntheticsName) {
		panic("Outbound request missing synthetics header NewRelicSyntheticsName: " + req.Header.Get(cat.NewRelicSyntheticsName))
	}

	expectedIntrinsics := map[string]interface{}{
		"name":                        "WebTransaction/Go/helloOldCAT",
		"client_cross_process_id":     "1#1",
		"nr.syntheticsResourceId":     "rrrrrrr-rrrr-1234-rrrr-rrrrrrrrrrrr",
		"nr.syntheticsJobId":          "jjjjjjj-jjjj-1234-jjjj-jjjjjjjjjjjj",
		"nr.syntheticsMonitorId":      "mmmmmmm-mmmm-1234-mmmm-mmmmmmmmmmmm",
		"nr.apdexPerfZone":            internal.MatchAnything,
		"nr.tripId":                   internal.MatchAnything,
		"nr.pathHash":                 internal.MatchAnything,
		"nr.referringPathHash":        internal.MatchAnything,
		"nr.referringTransactionGuid": internal.MatchAnything,
		"nr.guid":                     internal.MatchAnything,
	}

	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: expectedIntrinsics,
	}})
}

func TestSyntheticsBetterCAT(t *testing.T) {
	cfgFn := func(cfg *Config) {
		cfg.CrossApplicationTracer.Enabled = false
		cfg.DistributedTracer.Enabled = true
	}
	app := testApp(syntheticsConnectReplyFn, cfgFn, t)
	clientTxn := app.StartTransaction("helloBetterCAT")
	clientTxn.SetWebRequestHTTP(inboundSyntheticsRequestBuilder(false, true))

	req, err := http.NewRequest("GET", "newrelic.com", nil)

	if nil != err {
		panic(err)
	}

	StartExternalSegment(clientTxn, req)
	clientTxn.End()

	if "" == req.Header.Get(cat.NewRelicSyntheticsName) {
		panic("Outbound request missing synthetics header NewRelicSyntheticsName: " + req.Header.Get(cat.NewRelicSyntheticsName))
	}

	expectedIntrinsics := map[string]interface{}{
		"name":                    "WebTransaction/Go/helloBetterCAT",
		"nr.syntheticsResourceId": "rrrrrrr-rrrr-1234-rrrr-rrrrrrrrrrrr",
		"nr.syntheticsJobId":      "jjjjjjj-jjjj-1234-jjjj-jjjjjjjjjjjj",
		"nr.syntheticsMonitorId":  "mmmmmmm-mmmm-1234-mmmm-mmmmmmmmmmmm",
		"nr.apdexPerfZone":        internal.MatchAnything,
		"priority":                internal.MatchAnything,
		"sampled":                 internal.MatchAnything,
		"traceId":                 internal.MatchAnything,
		"guid":                    internal.MatchAnything,
	}

	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: expectedIntrinsics,
	}})
}

func TestSyntheticsStandalone(t *testing.T) {
	cfgFn := func(cfg *Config) {
		cfg.AppName = "syntheticsReceiver"
		cfg.DistributedTracer.Enabled = false
		cfg.CrossApplicationTracer.Enabled = false
	}
	app := testApp(syntheticsConnectReplyFn, cfgFn, t)
	clientTxn := app.StartTransaction("helloSynthetics")
	clientTxn.SetWebRequestHTTP(inboundSyntheticsRequestBuilder(false, false))

	req, err := http.NewRequest("GET", "newrelic.com", nil)

	if nil != err {
		panic(err)
	}

	StartExternalSegment(clientTxn, req)
	clientTxn.End()

	if "" == req.Header.Get(cat.NewRelicSyntheticsName) {
		panic("Outbound request missing synthetics header NewRelicSyntheticsName: " + req.Header.Get(cat.NewRelicSyntheticsName))
	}

	expectedIntrinsics := map[string]interface{}{
		"name":                    "WebTransaction/Go/helloSynthetics",
		"nr.syntheticsResourceId": "rrrrrrr-rrrr-1234-rrrr-rrrrrrrrrrrr",
		"nr.syntheticsJobId":      "jjjjjjj-jjjj-1234-jjjj-jjjjjjjjjjjj",
		"nr.syntheticsMonitorId":  "mmmmmmm-mmmm-1234-mmmm-mmmmmmmmmmmm",
		"nr.apdexPerfZone":        internal.MatchAnything,
		"nr.guid":                 internal.MatchAnything,
	}

	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: expectedIntrinsics,
	}})
}
