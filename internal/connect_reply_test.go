// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCreateFullTxnNameBasic(t *testing.T) {
	emptyReply := ConnectReplyDefaults()

	tcs := []struct {
		input      string
		background bool
		expect     string
	}{
		{"", true, "WebTransaction/Go/"},
		{"/", true, "WebTransaction/Go/"},
		{"hello", true, "WebTransaction/Go/hello"},
		{"/hello", true, "WebTransaction/Go/hello"},

		{"", false, "OtherTransaction/Go/"},
		{"/", false, "OtherTransaction/Go/"},
		{"hello", false, "OtherTransaction/Go/hello"},
		{"/hello", false, "OtherTransaction/Go/hello"},
	}

	for _, tc := range tcs {
		if out := CreateFullTxnName(tc.input, emptyReply, tc.background); out != tc.expect {
			t.Error(tc.input, tc.background, out, tc.expect)
		}
	}
}

func TestCreateFullTxnNameURLRulesIgnore(t *testing.T) {
	js := `[{
		"match_expression":".*zip.*$",
		"ignore":true
	}]`
	reply := ConnectReplyDefaults()
	err := json.Unmarshal([]byte(js), &reply.URLRules)
	if nil != err {
		t.Fatal(err)
	}
	if out := CreateFullTxnName("/zap/zip/zep", reply, true); out != "" {
		t.Error(out)
	}
}

func TestCreateFullTxnNameTxnRulesIgnore(t *testing.T) {
	js := `[{
		"match_expression":"^WebTransaction/Go/zap/zip/zep$",
		"ignore":true
	}]`
	reply := ConnectReplyDefaults()
	err := json.Unmarshal([]byte(js), &reply.TxnNameRules)
	if nil != err {
		t.Fatal(err)
	}
	if out := CreateFullTxnName("/zap/zip/zep", reply, true); out != "" {
		t.Error(out)
	}
}

func TestCreateFullTxnNameAllRulesWithCache(t *testing.T) {
	js := `{
		"url_rules":[
			{"match_expression":"zip","each_segment":true,"replacement":"zoop"}
		],
		"transaction_name_rules":[
			{"match_expression":"WebTransaction/Go/zap/zoop/zep",
			 "replacement":"WebTransaction/Go/zap/zoop/zep/zup/zyp"}
		],
		"transaction_segment_terms":[
			{"prefix": "WebTransaction/Go/",
			 "terms": ["zyp", "zoop", "zap"]}
		]
	}`
	reply := ConnectReplyDefaults()
	reply.rulesCache = newRulesCache(3)
	err := json.Unmarshal([]byte(js), &reply)
	if nil != err {
		t.Fatal(err)
	}
	want := "WebTransaction/Go/zap/zoop/*/zyp"
	if out := CreateFullTxnName("/zap/zip/zep", reply, true); out != want {
		t.Error("wanted:", want, "got:", out)
	}
	// Check that the cache was populated as expected.
	if out := reply.rulesCache.find("/zap/zip/zep", true); out != want {
		t.Error("wanted:", want, "got:", out)
	}
	// Check that the next CreateFullTxnName returns the same output.
	if out := CreateFullTxnName("/zap/zip/zep", reply, true); out != want {
		t.Error("wanted:", want, "got:", out)
	}
}

func TestCalculateApdexThreshold(t *testing.T) {
	reply := ConnectReplyDefaults()
	threshold := CalculateApdexThreshold(reply, "WebTransaction/Go/hello")
	if threshold != 500*time.Millisecond {
		t.Error("default apdex threshold", threshold)
	}

	reply = ConnectReplyDefaults()
	reply.ApdexThresholdSeconds = 1.3
	reply.KeyTxnApdex = map[string]float64{
		"WebTransaction/Go/zip": 2.2,
		"WebTransaction/Go/zap": 2.3,
	}
	threshold = CalculateApdexThreshold(reply, "WebTransaction/Go/hello")
	if threshold != 1300*time.Millisecond {
		t.Error(threshold)
	}
	threshold = CalculateApdexThreshold(reply, "WebTransaction/Go/zip")
	if threshold != 2200*time.Millisecond {
		t.Error(threshold)
	}
}

func TestIsTrusted(t *testing.T) {
	for _, test := range []struct {
		id       int
		trusted  string
		expected bool
	}{
		{1, `[]`, false},
		{1, `[2, 3]`, false},
		{1, `[1]`, true},
		{1, `[1, 2, 3]`, true},
	} {
		trustedAccounts := make(trustedAccountSet)
		if err := json.Unmarshal([]byte(test.trusted), &trustedAccounts); err != nil {
			t.Fatal(err)
		}

		if actual := trustedAccounts.IsTrusted(test.id); test.expected != actual {
			t.Errorf("failed asserting whether %d is trusted by %v: expected %v; got %v", test.id, test.trusted, test.expected, actual)
		}
	}
}

func BenchmarkDefaultRules(b *testing.B) {
	js := `{"url_rules":[
		{
			"match_expression":".*\\.(ace|arj|ini|txt|udl|plist|css|gif|ico|jpe?g|js|png|swf|woff|caf|aiff|m4v|mpe?g|mp3|mp4|mov)$",
			"replacement":"/*.\\1",
			"ignore":false,
			"eval_order":1000,
			"terminate_chain":true,
			"replace_all":false,
			"each_segment":false
		},
		{
			"match_expression":"^[0-9][0-9a-f_,.-]*$",
			"replacement":"*",
			"ignore":false,
			"eval_order":1001,
			"terminate_chain":false,
			"replace_all":false,
			"each_segment":true
		},
		{
			"match_expression":"^(.*)/[0-9][0-9a-f_,-]*\\.([0-9a-z][0-9a-z]*)$",
			"replacement":"\\1/.*\\2",
			"ignore":false,
			"eval_order":1002,
			"terminate_chain":false,
			"replace_all":false,
			"each_segment":false
		}
	]}`
	reply := ConnectReplyDefaults()
	reply.rulesCache = newRulesCache(1)
	err := json.Unmarshal([]byte(js), &reply)
	if nil != err {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if out := CreateFullTxnName("/myEndpoint", reply, true); out != "WebTransaction/Go/myEndpoint" {
			b.Error(out)
		}
	}
}

func TestNegativeHarvestLimits(t *testing.T) {
	// Test that negative harvest event limits will cause a connect error.
	// Harvest event limits are never expected to be negative:  This is just
	// extra defensiveness.
	_, err := ConstructConnectReply([]byte(`{"return_value":{
			"event_harvest_config": {
				"harvest_limits": {
					"error_event_data": -1
				}
			}
		}}`), PreconnectReply{})
	if err == nil {
		t.Fatal("expected error missing")
	}
}

type dfltMaxTxnEvents struct{}

func (dfltMaxTxnEvents) MaxTxnEvents() int {
	return MaxTxnEvents
}

func TestDefaultEventHarvestConfigJSON(t *testing.T) {
	js, err := json.Marshal(DefaultEventHarvestConfig(dfltMaxTxnEvents{}))
	if err != nil {
		t.Error(err)
	}
	if string(js) != `{"report_period_ms":60000,"harvest_limits":{"analytic_event_data":10000,"custom_event_data":10000,"error_event_data":100}}` {
		t.Error(string(js))
	}
}
