package internal

import (
	"net/url"
	"testing"
)

var (
	actualData = "my_data"
	call       = Cmd{
		Name:      "error_data",
		UseTLS:    true,
		Collector: "the-collector.com",
		License:   "12345",
		RunID:     "db97531",
		Data:      []byte(actualData),
	}
)

func TestLicenseInvalid(t *testing.T) {
	r := compactJSONString(`{
		"exception":{
			"message":"Invalid license key, please contact support@newrelic.com",
			"error_type":"NewRelic::Agent::LicenseException"
		}
	}`)
	reply, err := parseResponse([]byte(r))
	if reply != nil {
		t.Fatal(string(reply))
	}
	if !IsLicenseException(err) {
		t.Fatal(err)
	}
}

func TestRedirectSuccess(t *testing.T) {
	r := `{"return_value":"staging-collector-101.newrelic.com"}`
	reply, err := parseResponse([]byte(r))
	if nil != err {
		t.Fatal(err)
	}
	if string(reply) != `"staging-collector-101.newrelic.com"` {
		t.Fatal(string(reply))
	}
}

func TestEmptyHash(t *testing.T) {
	reply, err := parseResponse([]byte(`{}`))
	if nil != err {
		t.Fatal(err)
	}
	if nil != reply {
		t.Fatal(string(reply))
	}
}

func TestReturnValueNull(t *testing.T) {
	reply, err := parseResponse([]byte(`{"return_value":null}`))
	if nil != err {
		t.Fatal(err)
	}
	if "null" != string(reply) {
		t.Fatal(string(reply))
	}
}

func TestReplyNull(t *testing.T) {
	reply, err := parseResponse(nil)

	if nil == err || err.Error() != `unexpected end of JSON input` {
		t.Fatal(err)
	}
	if nil != reply {
		t.Fatal(string(reply))
	}
}

func TestConnectSuccess(t *testing.T) {
	inner := `{
	"agent_run_id":"599551769342729",
	"product_level":40,
	"js_agent_file":"",
	"cross_process_id":"12345#12345",
	"collect_errors":true,
	"url_rules":[
		{
			"each_segment":false,
			"match_expression":".*\\.(txt|udl|plist|css)$",
			"eval_order":1000,
			"replace_all":false,
			"ignore":false,
			"terminate_chain":true,
			"replacement":"\/*.\\1"
		},
		{
			"each_segment":true,
			"match_expression":"^[0-9][0-9a-f_,.-]*$",
			"eval_order":1001,
			"replace_all":false,
			"ignore":false,
			"terminate_chain":false,
			"replacement":"*"
		}
	],
	"messages":[
		{
			"message":"Reporting to staging",
			"level":"INFO"
		}
	],
	"data_report_period":60,
	"collect_traces":true,
	"sampling_rate":0,
	"js_agent_loader":"",
	"encoding_key":"the-encoding-key",
	"apdex_t":0.5,
	"collect_analytics_events":true,
	"trusted_account_ids":[49402]
}`
	outer := `{"return_value":` + inner + `}`
	reply, err := parseResponse([]byte(outer))

	if nil != err {
		t.Fatal(err)
	}
	if string(reply) != inner {
		t.Fatal(string(reply))
	}
}

func TestClientError(t *testing.T) {
	r := `{"exception":{"message":"something","error_type":"my_error"}}`
	reply, err := parseResponse([]byte(r))
	if nil == err || err.Error() != "my_error: something" {
		t.Fatal(err)
	}
	if nil != reply {
		t.Fatal(string(reply))
	}
}

func TestForceRestartException(t *testing.T) {
	// NOTE: This string was generated manually, not taken from the actual
	// collector.
	r := compactJSONString(`{
		"exception":{
			"message":"something",
			"error_type":"NewRelic::Agent::ForceRestartException"
		}
	}`)
	reply, err := parseResponse([]byte(r))
	if reply != nil {
		t.Fatal(string(reply))
	}
	if !IsRestartException(err) {
		t.Fatal(err)
	}
}

func TestForceDisconnectException(t *testing.T) {
	// NOTE: This string was generated manually, not taken from the actual
	// collector.
	r := compactJSONString(`{
		"exception":{
			"message":"something",
			"error_type":"NewRelic::Agent::ForceDisconnectException"
		}
	}`)
	reply, err := parseResponse([]byte(r))
	if reply != nil {
		t.Fatal(string(reply))
	}
	if !IsDisconnect(err) {
		t.Fatal(err)
	}
}

func TestRuntimeError(t *testing.T) {
	// NOTE: This string was generated manually, not taken from the actual
	// collector.
	r := `{"exception":{"message":"something","error_type":"RuntimeError"}}`
	reply, err := parseResponse([]byte(r))
	if reply != nil {
		t.Fatal(string(reply))
	}
	if !IsRuntime(err) {
		t.Fatal(err)
	}
}

func TestUnknownError(t *testing.T) {
	r := `{"exception":{"message":"something","error_type":"unknown_type"}}`
	reply, err := parseResponse([]byte(r))
	if reply != nil {
		t.Fatal(string(reply))
	}
	if nil == err || err.Error() != "unknown_type: something" {
		t.Fatal(err)
	}
}

func TestUrl(t *testing.T) {
	cmd := Cmd{
		Name:      "foo_method",
		Collector: "example.com",
		License:   "123abc",
	}

	out := cmd.url()
	u, err := url.Parse(out)
	if err != nil {
		t.Fatalf("url.Parse(%q) = %q", out, err)
	}

	got := u.Query().Get("license_key")
	if got != cmd.License {
		t.Errorf("got=%q cmd.License=%q", got, cmd.License)
	}
}
