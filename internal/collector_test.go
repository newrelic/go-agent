// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/newrelic/go-agent/internal/crossagent"
	"github.com/newrelic/go-agent/internal/logger"
)

func TestResponseCodeError(t *testing.T) {
	testcases := []struct {
		code            int
		success         bool
		disconnect      bool
		restart         bool
		saveHarvestData bool
	}{
		// success
		{code: 200, success: true, disconnect: false, restart: false, saveHarvestData: false},
		{code: 202, success: true, disconnect: false, restart: false, saveHarvestData: false},
		// disconnect
		{code: 410, success: false, disconnect: true, restart: false, saveHarvestData: false},
		// restart
		{code: 401, success: false, disconnect: false, restart: true, saveHarvestData: false},
		{code: 409, success: false, disconnect: false, restart: true, saveHarvestData: false},
		// save data
		{code: 408, success: false, disconnect: false, restart: false, saveHarvestData: true},
		{code: 429, success: false, disconnect: false, restart: false, saveHarvestData: true},
		{code: 500, success: false, disconnect: false, restart: false, saveHarvestData: true},
		{code: 503, success: false, disconnect: false, restart: false, saveHarvestData: true},
		// other errors
		{code: 400, success: false, disconnect: false, restart: false, saveHarvestData: false},
		{code: 403, success: false, disconnect: false, restart: false, saveHarvestData: false},
		{code: 404, success: false, disconnect: false, restart: false, saveHarvestData: false},
		{code: 405, success: false, disconnect: false, restart: false, saveHarvestData: false},
		{code: 407, success: false, disconnect: false, restart: false, saveHarvestData: false},
		{code: 411, success: false, disconnect: false, restart: false, saveHarvestData: false},
		{code: 413, success: false, disconnect: false, restart: false, saveHarvestData: false},
		{code: 414, success: false, disconnect: false, restart: false, saveHarvestData: false},
		{code: 415, success: false, disconnect: false, restart: false, saveHarvestData: false},
		{code: 417, success: false, disconnect: false, restart: false, saveHarvestData: false},
		{code: 431, success: false, disconnect: false, restart: false, saveHarvestData: false},
		// unexpected weird codes
		{code: -1, success: false, disconnect: false, restart: false, saveHarvestData: false},
		{code: 1, success: false, disconnect: false, restart: false, saveHarvestData: false},
		{code: 999999, success: false, disconnect: false, restart: false, saveHarvestData: false},
	}
	for _, tc := range testcases {
		resp := newRPMResponse(tc.code)
		if tc.success != (nil == resp.Err) {
			t.Error("error", tc.code, tc.success, resp.Err)
		}
		if tc.disconnect != resp.IsDisconnect() {
			t.Error("disconnect", tc.code, tc.disconnect, resp.Err)
		}
		if tc.restart != resp.IsRestartException() {
			t.Error("restart", tc.code, tc.restart, resp.Err)
		}
		if tc.saveHarvestData != resp.ShouldSaveHarvestData() {
			t.Error("save harvest data", tc.code, tc.saveHarvestData, resp.Err)
		}
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func TestCollectorRequest(t *testing.T) {
	cmd := RpmCmd{
		Name:              "cmd_name",
		Collector:         "collector.com",
		RunID:             "run_id",
		Data:              nil,
		RequestHeadersMap: map[string]string{"zip": "zap"},
		MaxPayloadSize:    maxPayloadSizeInBytes,
	}
	testField := func(name, v1, v2 string) {
		if v1 != v2 {
			t.Error(name, v1, v2)
		}
	}
	cs := RpmControls{
		License: "the_license",
		Client: &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				testField("method", r.Method, "POST")
				testField("url", r.URL.String(), "https://collector.com/agent_listener/invoke_raw_method?license_key=the_license&marshal_format=json&method=cmd_name&protocol_version=17&run_id=run_id")
				testField("Accept-Encoding", r.Header.Get("Accept-Encoding"), "identity, deflate")
				testField("Content-Type", r.Header.Get("Content-Type"), "application/octet-stream")
				testField("User-Agent", r.Header.Get("User-Agent"), "NewRelic-Go-Agent/agent_version")
				testField("Content-Encoding", r.Header.Get("Content-Encoding"), "gzip")
				testField("zip", r.Header.Get("zip"), "zap")
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader("body")),
				}, nil
			}),
		},
		Logger:       logger.ShimLogger{IsDebugEnabled: true},
		AgentVersion: "agent_version",
		GzipWriterPool: &sync.Pool{
			New: func() interface{} {
				return gzip.NewWriter(io.Discard)
			},
		},
	}
	resp := CollectorRequest(cmd, cs)
	if nil != resp.Err {
		t.Error(resp.Err)
	}
}

func TestCollectorBadRequest(t *testing.T) {
	cmd := RpmCmd{
		Name:              "cmd_name",
		Collector:         "collector.com",
		RunID:             "run_id",
		Data:              nil,
		RequestHeadersMap: map[string]string{"zip": "zap"},
	}
	cs := RpmControls{
		License: "the_license",
		Client: &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(strings.NewReader("body")),
				}, nil
			}),
		},
		Logger:       logger.ShimLogger{IsDebugEnabled: true},
		AgentVersion: "agent_version",
		GzipWriterPool: &sync.Pool{
			New: func() interface{} {
				return gzip.NewWriter(io.Discard)
			},
		},
	}
	u := ":" // bad url
	resp := collectorRequestInternal(u, cmd, cs)
	if nil == resp.Err {
		t.Error("missing expected error")
	}

}

func TestUrl(t *testing.T) {
	cmd := RpmCmd{
		Name:      "foo_method",
		Collector: "example.com",
	}
	cs := RpmControls{
		License:      "123abc",
		Client:       nil,
		Logger:       nil,
		AgentVersion: "1",
		GzipWriterPool: &sync.Pool{
			New: func() interface{} {
				return gzip.NewWriter(io.Discard)
			},
		},
	}

	out := rpmURL(cmd, cs)
	u, err := url.Parse(out)
	if err != nil {
		t.Fatalf("url.Parse(%q) = %q", out, err)
	}

	got := u.Query().Get("license_key")
	if got != cs.License {
		t.Errorf("got=%q cmd.License=%q", got, cs.License)
	}
	if u.Scheme != "https" {
		t.Error(u.Scheme)
	}
}

const (
	unknownRequiredPolicyBody = `{"return_value":{"redirect_host":"special_collector","security_policies":{"unknown_policy":{"enabled":true,"required":true}}}}`
	redirectBody              = `{"return_value":{"redirect_host":"special_collector"}}`
	connectBody               = `{"return_value":{"agent_run_id":"my_agent_run_id"}}`
	malformedBody             = `{"return_value":}}`
)

func makeResponse(code int, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Body:       ioutil.NopCloser(strings.NewReader(body)),
	}
}

type endpointResult struct {
	response *http.Response
	err      error
}

type connectMock struct {
	redirect endpointResult
	connect  endpointResult
	// testConfig will be used if this is nil
	config ConnectJSONCreator
}

func (m connectMock) RoundTrip(r *http.Request) (*http.Response, error) {
	cmd := r.URL.Query().Get("method")
	switch cmd {
	case cmdPreconnect:
		return m.redirect.response, m.redirect.err
	case cmdConnect:
		return m.connect.response, m.connect.err
	default:
		return nil, fmt.Errorf("unknown cmd: %s", cmd)
	}
}

func (m connectMock) CancelRequest(req *http.Request) {}

type testConfig struct{}

func (tc testConfig) CreateConnectJSON(*SecurityPolicies) ([]byte, error) {
	return []byte(`"connect-json"`), nil
}

type errorConfig struct{}

func (c errorConfig) CreateConnectJSON(*SecurityPolicies) ([]byte, error) {
	return nil, errors.New("error creating config JSON")
}

func testConnectHelper(cm connectMock) (*ConnectReply, RPMResponse) {
	config := cm.config
	if nil == config {
		config = testConfig{}
	}
	cs := RpmControls{
		License:      "12345",
		Client:       &http.Client{Transport: cm},
		Logger:       logger.ShimLogger{IsDebugEnabled: true},
		AgentVersion: "1",
		GzipWriterPool: &sync.Pool{
			New: func() interface{} {
				return gzip.NewWriter(io.Discard)
			},
		},
	}

	return ConnectAttempt(config, "", false, cs)
}

func TestConnectAttemptSuccess(t *testing.T) {
	run, resp := testConnectHelper(connectMock{
		redirect: endpointResult{response: makeResponse(200, redirectBody)},
		connect:  endpointResult{response: makeResponse(200, connectBody)},
	})
	if nil == run || nil != resp.Err {
		t.Fatal(run, resp.Err)
	}
	if run.Collector != "special_collector" {
		t.Error(run.Collector)
	}
	if run.RunID != "my_agent_run_id" {
		t.Error(run)
	}
}

func TestConnectClientError(t *testing.T) {
	run, resp := testConnectHelper(connectMock{
		redirect: endpointResult{response: makeResponse(200, redirectBody)},
		connect:  endpointResult{err: errors.New("client error")},
	})
	if nil != run {
		t.Fatal(run)
	}
	if resp.Err == nil {
		t.Fatal("missing expected error")
	}
}

func TestConnectConfigJSONError(t *testing.T) {
	run, resp := testConnectHelper(connectMock{
		redirect: endpointResult{response: makeResponse(200, redirectBody)},
		connect:  endpointResult{response: makeResponse(200, connectBody)},
		config:   errorConfig{},
	})
	if nil != run {
		t.Fatal(run)
	}
	if resp.Err == nil {
		t.Fatal("missing expected error")
	}
}

func TestConnectAttemptDisconnectOnRedirect(t *testing.T) {
	run, resp := testConnectHelper(connectMock{
		redirect: endpointResult{response: makeResponse(410, "")},
		connect:  endpointResult{response: makeResponse(200, connectBody)},
	})
	if nil != run {
		t.Error(run)
	}
	if nil == resp.Err {
		t.Fatal("missing error")
	}
	if !resp.IsDisconnect() {
		t.Fatal("should be disconnect")
	}
}

func TestConnectAttemptDisconnectOnConnect(t *testing.T) {
	run, resp := testConnectHelper(connectMock{
		redirect: endpointResult{response: makeResponse(200, redirectBody)},
		connect:  endpointResult{response: makeResponse(410, "")},
	})
	if nil != run {
		t.Error(run)
	}
	if nil == resp.Err {
		t.Fatal("missing error")
	}
	if !resp.IsDisconnect() {
		t.Fatal("should be disconnect")
	}
}

func TestConnectAttemptBadSecurityPolicies(t *testing.T) {
	run, resp := testConnectHelper(connectMock{
		redirect: endpointResult{response: makeResponse(200, unknownRequiredPolicyBody)},
		connect:  endpointResult{response: makeResponse(200, connectBody)},
	})
	if nil != run {
		t.Error(run)
	}
	if nil == resp.Err {
		t.Fatal("missing error")
	}
	if !resp.IsDisconnect() {
		t.Fatal("should be disconnect")
	}
}

func TestConnectAttemptInvalidJSON(t *testing.T) {
	run, resp := testConnectHelper(connectMock{
		redirect: endpointResult{response: makeResponse(200, redirectBody)},
		connect:  endpointResult{response: makeResponse(200, malformedBody)},
	})
	if nil != run {
		t.Error(run)
	}
	if nil == resp.Err {
		t.Fatal("missing error")
	}
}

func TestConnectAttemptCollectorNotString(t *testing.T) {
	run, resp := testConnectHelper(connectMock{
		redirect: endpointResult{response: makeResponse(200, `{"return_value":123}`)},
		connect:  endpointResult{response: makeResponse(200, connectBody)},
	})
	if nil != run {
		t.Error(run)
	}
	if nil == resp.Err {
		t.Fatal("missing error")
	}
}

func TestConnectAttempt401(t *testing.T) {
	run, resp := testConnectHelper(connectMock{
		redirect: endpointResult{response: makeResponse(200, redirectBody)},
		connect:  endpointResult{response: makeResponse(401, connectBody)},
	})
	if nil != run {
		t.Error(run)
	}
	if nil == resp.Err {
		t.Fatal("missing error")
	}
	if !resp.IsRestartException() {
		t.Fatal("should be restart")
	}
}

func TestConnectAttemptOtherReturnCode(t *testing.T) {
	run, resp := testConnectHelper(connectMock{
		redirect: endpointResult{response: makeResponse(200, redirectBody)},
		connect:  endpointResult{response: makeResponse(413, connectBody)},
	})
	if nil != run {
		t.Error(run)
	}
	if nil == resp.Err {
		t.Fatal("missing error")
	}
}

func TestConnectAttemptMissingRunID(t *testing.T) {
	run, resp := testConnectHelper(connectMock{
		redirect: endpointResult{response: makeResponse(200, redirectBody)},
		connect:  endpointResult{response: makeResponse(200, `{"return_value":{}}`)},
	})
	if nil != run {
		t.Error(run)
	}
	if errMissingAgentRunID != resp.Err {
		t.Fatal("wrong error", resp.Err)
	}
}

func TestCalculatePreconnectHost(t *testing.T) {
	// non-region license
	host := calculatePreconnectHost("0123456789012345678901234567890123456789", "")
	if host != preconnectHostDefault {
		t.Error(host)
	}
	// override present
	override := "other-collector.newrelic.com"
	host = calculatePreconnectHost("0123456789012345678901234567890123456789", override)
	if host != override {
		t.Error(host)
	}
	// four letter region
	host = calculatePreconnectHost("eu01xx6789012345678901234567890123456789", "")
	if host != "collector.eu01.nr-data.net" {
		t.Error(host)
	}
	// five letter region
	host = calculatePreconnectHost("gov01x6789012345678901234567890123456789", "")
	if host != "collector.gov01.nr-data.net" {
		t.Error(host)
	}
	// six letter region
	host = calculatePreconnectHost("foo001x6789012345678901234567890123456789", "")
	if host != "collector.foo001.nr-data.net" {
		t.Error(host)
	}
}

func TestPreconnectHostCrossAgent(t *testing.T) {
	var testcases []struct {
		Name               string `json:"name"`
		ConfigFileKey      string `json:"config_file_key"`
		EnvKey             string `json:"env_key"`
		ConfigOverrideHost string `json:"config_override_host"`
		EnvOverrideHost    string `json:"env_override_host"`
		ExpectHostname     string `json:"hostname"`
	}
	err := crossagent.ReadJSON("collector_hostname.json", &testcases)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range testcases {
		// mimic file/environment precedence of other agents
		configKey := tc.ConfigFileKey
		if "" != tc.EnvKey {
			configKey = tc.EnvKey
		}
		overrideHost := tc.ConfigOverrideHost
		if "" != tc.EnvOverrideHost {
			overrideHost = tc.EnvOverrideHost
		}

		host := calculatePreconnectHost(configKey, overrideHost)
		if host != tc.ExpectHostname {
			t.Errorf(`test="%s" got="%s" expected="%s"`, tc.Name, host, tc.ExpectHostname)
		}
	}
}

func TestCollectorRequestRespectsMaxPayloadSize(t *testing.T) {
	// Test that CollectorRequest returns an error when MaxPayloadSize is
	// exceeded
	cmd := RpmCmd{
		Name:           "cmd_name",
		Collector:      "collector.com",
		RunID:          "run_id",
		Data:           []byte("abcdefghijklmnopqrstuvwxyz"),
		MaxPayloadSize: 3,
	}
	cs := RpmControls{
		Client: &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				t.Error("no response should have gone out!")
				return nil, nil
			}),
		},
		Logger: logger.ShimLogger{IsDebugEnabled: true},
		GzipWriterPool: &sync.Pool{
			New: func() interface{} {
				return gzip.NewWriter(io.Discard)
			},
		},
	}
	resp := CollectorRequest(cmd, cs)
	if nil == resp.Err {
		t.Error("response should have contained error")
	}
	if resp.ShouldSaveHarvestData() {
		t.Error("harvest data should be discarded when max_payload_size_in_bytes is exceeded")
	}
}

func TestConnectReplyMaxPayloadSize(t *testing.T) {
	testcases := []struct {
		replyBody              string
		expectedMaxPayloadSize int
	}{
		{
			replyBody:              `{"return_value":{"agent_run_id":"my_agent_run_id"}}`,
			expectedMaxPayloadSize: 1000 * 1000,
		},
		{
			replyBody:              `{"return_value":{"agent_run_id":"my_agent_run_id","max_payload_size_in_bytes":123}}`,
			expectedMaxPayloadSize: 123,
		},
	}

	controls := func(replyBody string) RpmControls {
		return RpmControls{
			Client: &http.Client{
				Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Body:       ioutil.NopCloser(strings.NewReader(replyBody)),
					}, nil
				}),
			},
			Logger: logger.ShimLogger{IsDebugEnabled: true},
			GzipWriterPool: &sync.Pool{
				New: func() interface{} {
					return gzip.NewWriter(io.Discard)
				},
			},
		}
	}

	for _, test := range testcases {
		reply, resp := ConnectAttempt(testConfig{}, "", false, controls(test.replyBody))
		if nil != resp.Err {
			t.Error("resp returned unexpected error:", resp.Err)
		}
		if test.expectedMaxPayloadSize != reply.MaxPayloadSizeInBytes {
			t.Errorf("incorrect MaxPayloadSizeInBytes: expected=%d actual=%d",
				test.expectedMaxPayloadSize, reply.MaxPayloadSizeInBytes)
		}
	}
}

func TestPreconnectRequestMarshall(t *testing.T) {
	tests := map[string]preconnectRequest{
		`[{"security_policies_token":"securityPoliciesToken","high_security":false}]`: {
			SecurityPoliciesToken: "securityPoliciesToken",
			HighSecurity:          false,
		},
		`[{"security_policies_token":"securityPoliciesToken","high_security":true}]`: {
			SecurityPoliciesToken: "securityPoliciesToken",
			HighSecurity:          true,
		},
		`[{"high_security":true}]`: {
			SecurityPoliciesToken: "",
			HighSecurity:          true,
		},
		`[{"high_security":false}]`: {
			SecurityPoliciesToken: "",
			HighSecurity:          false,
		},
	}
	for expected, request := range tests {
		b, e := json.Marshal([]preconnectRequest{request})
		if e != nil {
			t.Fatal("Unable to marshall preconnect request", e)
		}
		result := string(b)
		if result != expected {
			t.Errorf("Invalid preconnect request marshall: expected %s, got %s", expected, result)
		}
	}
}
