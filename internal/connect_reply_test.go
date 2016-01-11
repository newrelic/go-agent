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
		{"", true, "WebTransaction/Pattern/"},
		{"/", true, "WebTransaction/Pattern/"},
		{"hello", true, "WebTransaction/Pattern/hello"},
		{"/hello", true, "WebTransaction/Pattern/hello"},

		{"", false, "OtherTransaction/Pattern/"},
		{"/", false, "OtherTransaction/Pattern/"},
		{"hello", false, "OtherTransaction/Pattern/hello"},
		{"/hello", false, "OtherTransaction/Pattern/hello"},
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
		"match_expression":"^WebTransaction/Pattern/zap/zip/zep$",
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

func TestCreateFullTxnNameAllRules(t *testing.T) {
	js := `{
		"url_rules":[
			{"match_expression":"zip","each_segment":true,"replacement":"zoop"}
		],
		"transaction_name_rules":[
			{"match_expression":"WebTransaction/Pattern/zap/zoop/zep",
			 "replacement":"WebTransaction/Pattern/zap/zoop/zep/zup/zyp"}
		],
		"transaction_segment_terms":[
			{"prefix": "WebTransaction/Pattern/",
			 "terms": ["zyp", "zoop", "zap"]}
		]
	}`
	reply := ConnectReplyDefaults()
	err := json.Unmarshal([]byte(js), &reply)
	if nil != err {
		t.Fatal(err)
	}
	if out := CreateFullTxnName("/zap/zip/zep", reply, true); out != "WebTransaction/Pattern/zap/zoop/*/zyp" {
		t.Error(out)
	}
}

func TestCalculateApdexThreshold(t *testing.T) {
	reply := ConnectReplyDefaults()
	threshold := calculateApdexThreshold(reply, "WebTransaction/Pattern/hello")
	if threshold != 500*time.Millisecond {
		t.Error("default apdex threshold", threshold)
	}

	reply = ConnectReplyDefaults()
	reply.ApdexThresholdSeconds = 1.3
	reply.KeyTxnApdex = map[string]float64{
		"WebTransaction/Pattern/zip": 2.2,
		"WebTransaction/Pattern/zap": 2.3,
	}
	threshold = calculateApdexThreshold(reply, "WebTransaction/Pattern/hello")
	if threshold != 1300*time.Millisecond {
		t.Error(threshold)
	}
	threshold = calculateApdexThreshold(reply, "WebTransaction/Pattern/zip")
	if threshold != 2200*time.Millisecond {
		t.Error(threshold)
	}
}
