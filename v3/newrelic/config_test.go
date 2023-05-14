// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/crossagent"
	"github.com/newrelic/go-agent/v3/internal/utilization"
)

type labelsTestCase struct {
	Name        string `json:"name"`
	LabelString string `json:"labelString"`
	Warning     bool   `json:"warning"`
	Expected    []struct {
		LabelType  string `json:"label_type"`
		LabelValue string `json:"label_value"`
	} `json:"expected"`
}

func TestCrossAgentLabels(t *testing.T) {
	var tcs []json.RawMessage

	err := crossagent.ReadJSON("labels.json", &tcs)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tcs {
		runLabelsTestCase(t, tc)
	}
}

func runLabelsTestCase(t *testing.T, js json.RawMessage) {
	var tc labelsTestCase
	if err := json.Unmarshal(js, &tc); nil != err {
		t.Error(err)
		return
	}

	actual := getLabels(tc.LabelString)
	if len(actual) != len(tc.Expected) {
		t.Errorf("%s: incorrect number of elements: actual=%d expect=%d", tc.Name, len(actual), len(tc.Expected))
		return
	}
	for _, exp := range tc.Expected {
		if v, ok := actual[exp.LabelType]; !ok {
			t.Errorf("%s: key %s not in actual: actual=%#v", tc.Name, exp.LabelType, actual)
		} else if v != exp.LabelValue {
			t.Errorf("%s: incorrect value found for key %s: actual=%#v expect=%#v", tc.Name, exp.LabelType, actual, tc.Expected)
		}
	}
}

var (
	fixRegex = regexp.MustCompile(`e\+\d+`)
)

// In Go 1.8 Marshalling of numbers was changed:
// Before: "StackTraceThreshold":5e+08
// After:  "StackTraceThreshold":500000000
func standardizeNumbers(input string) string {
	return fixRegex.ReplaceAllStringFunc(input, func(s string) string {
		n, err := strconv.Atoi(s[2:])
		if nil != err {
			return s
		}
		return strings.Repeat("0", n)
	})
}

func TestCopyConfigReferenceFieldsPresent(t *testing.T) {
	cfg := defaultConfig()
	cfg.AppName = "my appname"
	cfg.License = "0123456789012345678901234567890123456789"
	cfg.Labels["zip"] = "zap"
	cfg.ErrorCollector.IgnoreStatusCodes = append(cfg.ErrorCollector.IgnoreStatusCodes, 405)
	cfg.ErrorCollector.ExpectStatusCodes = append(cfg.ErrorCollector.ExpectStatusCodes, 500)
	cfg.Attributes.Include = append(cfg.Attributes.Include, "1")
	cfg.Attributes.Exclude = append(cfg.Attributes.Exclude, "2")
	cfg.TransactionEvents.Attributes.Include = append(cfg.TransactionEvents.Attributes.Include, "3")
	cfg.TransactionEvents.Attributes.Exclude = append(cfg.TransactionEvents.Attributes.Exclude, "4")
	cfg.ErrorCollector.Attributes.Include = append(cfg.ErrorCollector.Attributes.Include, "5")
	cfg.ErrorCollector.Attributes.Exclude = append(cfg.ErrorCollector.Attributes.Exclude, "6")
	cfg.TransactionTracer.Attributes.Include = append(cfg.TransactionTracer.Attributes.Include, "7")
	cfg.TransactionTracer.Attributes.Exclude = append(cfg.TransactionTracer.Attributes.Exclude, "8")
	cfg.BrowserMonitoring.Attributes.Include = append(cfg.BrowserMonitoring.Attributes.Include, "9")
	cfg.BrowserMonitoring.Attributes.Exclude = append(cfg.BrowserMonitoring.Attributes.Exclude, "10")
	cfg.SpanEvents.Attributes.Include = append(cfg.SpanEvents.Attributes.Include, "11")
	cfg.SpanEvents.Attributes.Exclude = append(cfg.SpanEvents.Attributes.Exclude, "12")
	cfg.TransactionTracer.Segments.Attributes.Include = append(cfg.TransactionTracer.Segments.Attributes.Include, "13")
	cfg.TransactionTracer.Segments.Attributes.Exclude = append(cfg.TransactionTracer.Segments.Attributes.Exclude, "14")
	cfg.Transport = &http.Transport{}
	cfg.Logger = NewLogger(os.Stdout)

	cp := copyConfigReferenceFields(cfg)

	cfg.Labels["zop"] = "zup"
	cfg.ErrorCollector.IgnoreStatusCodes[0] = 201
	cfg.Attributes.Include[0] = "zap"
	cfg.Attributes.Exclude[0] = "zap"
	cfg.TransactionEvents.Attributes.Include[0] = "zap"
	cfg.TransactionEvents.Attributes.Exclude[0] = "zap"
	cfg.ErrorCollector.Attributes.Include[0] = "zap"
	cfg.ErrorCollector.Attributes.Exclude[0] = "zap"
	cfg.TransactionTracer.Attributes.Include[0] = "zap"
	cfg.TransactionTracer.Attributes.Exclude[0] = "zap"
	cfg.BrowserMonitoring.Attributes.Include[0] = "zap"
	cfg.BrowserMonitoring.Attributes.Exclude[0] = "zap"
	cfg.SpanEvents.Attributes.Include[0] = "zap"
	cfg.SpanEvents.Attributes.Exclude[0] = "zap"
	cfg.TransactionTracer.Segments.Attributes.Include[0] = "zap"
	cfg.TransactionTracer.Segments.Attributes.Exclude[0] = "zap"

	expect := internal.CompactJSONString(fmt.Sprintf(`[
	{
		"pid":123,
		"language":"go",
		"agent_version":"0.2.2",
		"host":"my-hostname",
		"settings":{
			"AppName":"my appname",
			"ApplicationLogging": {
				"Enabled": true,
				"Forwarding": {
					"Enabled": true,
					"MaxSamplesStored": %d
				},
				"LocalDecorating":{
					"Enabled": false
				},
				"Metrics": {
					"Enabled": true
				}
			},
			"Attributes":{"Enabled":true,"Exclude":["2"],"Include":["1"]},
			"BrowserMonitoring":{
				"Attributes":{"Enabled":false,"Exclude":["10"],"Include":["9"]},
				"Enabled":true
			},
			"CodeLevelMetrics":{"Enabled":false,"IgnoredPrefix":"","IgnoredPrefixes":null,"PathPrefix":"","PathPrefixes":null,"RedactIgnoredPrefixes":true,"RedactPathPrefixes":true,"Scope":"all"},
			"CrossApplicationTracer":{"Enabled":false},
			"CustomInsightsEvents":{
				"Enabled":true,
				"MaxSamplesStored":%d
			},
			"DatastoreTracer":{
				"DatabaseNameReporting":{"Enabled":true},
				"InstanceReporting":{"Enabled":true},
				"QueryParameters":{"Enabled":true},
				"SlowQuery":{
					"Enabled":true,
					"Threshold":10000000
				}
			},
			"DistributedTracer":{"Enabled":true,"ExcludeNewRelicHeader":false,"ReservoirLimit":%d},
			"Enabled":true,
			"Error":null,
			"ErrorCollector":{
				"Attributes":{"Enabled":true,"Exclude":["6"],"Include":["5"]},
				"CaptureEvents":true,
				"Enabled":true,
				"ExpectStatusCodes":[500],
				"IgnoreStatusCodes":[0,5,404,405],
				"RecordPanics":false
			},
			"Heroku":{
				"DynoNamePrefixesToShorten":["scheduler","run"],
				"UseDynoNames":true
			},
			"HighSecurity":false,
			"Host":"",
			"HostDisplayName":"",
			"InfiniteTracing": {
				"SpanEvents": {"QueueSize":10000},
				"TraceObserver": {
					"Host": "",
					"Port": 443
                }
			},
			"Labels":{"zip":"zap"},
			"Logger":"*logger.logFile",
			"ModuleDependencyMetrics":{"Enabled":true,"IgnoredPrefixes":null,"RedactIgnoredPrefixes":true},
			"RuntimeSampler":{"Enabled":true},
			"SecurityPoliciesToken":"",
			"ServerlessMode":{
				"AccountID":"",
				"ApdexThreshold":500000000,
				"Enabled":false,
				"PrimaryAppID":"",
				"TrustedAccountKey":""
			},
			"SpanEvents":{
				"Attributes":{
					"Enabled":true,"Exclude":["12"],"Include":["11"]
				},
				"Enabled":true
			},
			"TransactionEvents":{
				"Attributes":{"Enabled":true,"Exclude":["4"],"Include":["3"]},
				"Enabled":true,
				"MaxSamplesStored": %d
			},
			"TransactionTracer":{
				"Attributes":{"Enabled":true,"Exclude":["8"],"Include":["7"]},
				"Enabled":true,
				"Segments":{
					"Attributes":{"Enabled":true,"Exclude":["14"],"Include":["13"]},
					"StackTraceThreshold":500000000,
					"Threshold":2000000
				},
				"Threshold":{
					"Duration":500000000,
					"IsApdexFailing":true
				}
			},
			"Transport":"*http.Transport",
			"Utilization":{
				"BillingHostname":"",
				"DetectAWS":true,
				"DetectAzure":true,
				"DetectDocker":true,
				"DetectGCP":true,
				"DetectKubernetes":true,
				"DetectPCF":true,
				"LogicalProcessors":0,
				"TotalRAMMIB":0
			},
			"browser_monitoring.loader":"rum"
		},
		"app_name":["my appname"],
		"high_security":false,
		"labels":[{"label_type":"zip","label_value":"zap"}],
		"environment":[
			["runtime.NumCPU",8],
			["runtime.Compiler","comp"],
			["runtime.GOARCH","arch"],
			["runtime.GOOS","goos"],
			["runtime.Version","vers"],
			["Modules", null]
		],
		"identifier":"my appname",
		"utilization":{
			"metadata_version":5,
			"logical_processors":16,
			"total_ram_mib":1024,
			"hostname":"my-hostname"
		},
		"security_policies":{
			"record_sql":{"enabled":false},
			"attributes_include":{"enabled":false},
			"allow_raw_exception_messages":{"enabled":false},
			"custom_events":{"enabled":false},
			"custom_parameters":{"enabled":false}
		},
		"metadata":{
			"NEW_RELIC_METADATA_ZAP":"zip"
		},
		"event_harvest_config": {
			"report_period_ms": 60000,
			"harvest_limits": {
				"analytic_event_data": 10000,
				"custom_event_data": %d,
				"log_event_data": %d,
				"error_event_data": 100,
				"span_event_data": %d
			}
		}
	}]`, internal.MaxLogEvents, internal.MaxCustomEvents, internal.MaxSpanEvents, internal.MaxTxnEvents, internal.MaxCustomEvents, internal.MaxTxnEvents, internal.MaxSpanEvents))

	securityPoliciesInput := []byte(`{
		"record_sql":                    { "enabled": false, "required": false },
	        "attributes_include":            { "enabled": false, "required": false },
	        "allow_raw_exception_messages":  { "enabled": false, "required": false },
	        "custom_events":                 { "enabled": false, "required": false },
	        "custom_parameters":             { "enabled": false, "required": false },
	        "custom_instrumentation_editor": { "enabled": false, "required": false },
	        "message_parameters":            { "enabled": false, "required": false },
	        "job_arguments":                 { "enabled": false, "required": false }
	}`)
	var sp internal.SecurityPolicies
	err := json.Unmarshal(securityPoliciesInput, &sp)
	if nil != err {
		t.Fatal(err)
	}

	metadata := map[string]string{
		"NEW_RELIC_METADATA_ZAP": "zip",
	}
	js, err := configConnectJSONInternal(cp, 123, &utilization.SampleData, sampleEnvironment, "0.2.2", sp.PointerIfPopulated(), metadata)
	if nil != err {
		t.Fatal(err)
	}
	out := standardizeNumbers(string(js))
	if out != expect {
		t.Error(expect)
		t.Error(out)
	}
}

func TestCopyConfigReferenceFieldsAbsent(t *testing.T) {
	cfg := defaultConfig()
	cfg.AppName = "my appname"
	cfg.License = "0123456789012345678901234567890123456789"
	cfg.Labels = nil
	cfg.ErrorCollector.IgnoreStatusCodes = nil

	cp := copyConfigReferenceFields(cfg)

	expect := internal.CompactJSONString(fmt.Sprintf(`[
	{
		"pid":123,
		"language":"go",
		"agent_version":"0.2.2",
		"host":"my-hostname",
		"settings":{
			"AppName":"my appname",
			"ApplicationLogging": {
				"Enabled": true,
				"Forwarding": {
					"Enabled": true,
					"MaxSamplesStored": %d
				},
				"LocalDecorating":{
					"Enabled": false
				},
				"Metrics": {
					"Enabled": true
				}
			},
			"Attributes":{"Enabled":true,"Exclude":null,"Include":null},
			"BrowserMonitoring":{
				"Attributes":{
					"Enabled":false,
					"Exclude":null,
					"Include":null
				},
				"Enabled":true
			},
			"CodeLevelMetrics":{"Enabled":false,"IgnoredPrefix":"","IgnoredPrefixes":null,"PathPrefix":"","PathPrefixes":null,"RedactIgnoredPrefixes":true,"RedactPathPrefixes":true,"Scope":"all"},
			"CrossApplicationTracer":{"Enabled":false},
			"CustomInsightsEvents":{
				"Enabled":true,
				"MaxSamplesStored":%d
			},
			"DatastoreTracer":{
				"DatabaseNameReporting":{"Enabled":true},
				"InstanceReporting":{"Enabled":true},
				"QueryParameters":{"Enabled":true},
				"SlowQuery":{
					"Enabled":true,
					"Threshold":10000000
				}
			},
			"DistributedTracer":{"Enabled":true,"ExcludeNewRelicHeader":false,"ReservoirLimit":%d},
			"Enabled":true,
			"Error":null,
			"ErrorCollector":{
				"Attributes":{"Enabled":true,"Exclude":null,"Include":null},
				"CaptureEvents":true,
				"Enabled":true,
				"ExpectStatusCodes":null,
				"IgnoreStatusCodes":null,
				"RecordPanics":false
			},
			"Heroku":{
				"DynoNamePrefixesToShorten":["scheduler","run"],
				"UseDynoNames":true
			},
			"HighSecurity":false,
			"Host":"",
			"HostDisplayName":"",
			"InfiniteTracing": {
				"SpanEvents": {"QueueSize":10000},
				"TraceObserver": {
					"Host": "",
					"Port": 443
                }
			},
			"Labels":null,
			"Logger":null,
			"ModuleDependencyMetrics":{"Enabled":true,"IgnoredPrefixes":null,"RedactIgnoredPrefixes":true},
			"RuntimeSampler":{"Enabled":true},
			"SecurityPoliciesToken":"",
			"ServerlessMode":{
				"AccountID":"",
				"ApdexThreshold":500000000,
				"Enabled":false,
				"PrimaryAppID":"",
				"TrustedAccountKey":""
			},
			"SpanEvents":{
				"Attributes":{"Enabled":true,"Exclude":null,"Include":null},
				"Enabled":true
			},
			"TransactionEvents":{
				"Attributes":{"Enabled":true,"Exclude":null,"Include":null},
				"Enabled":true,
				"MaxSamplesStored": %d
			},
			"TransactionTracer":{
				"Attributes":{"Enabled":true,"Exclude":null,"Include":null},
				"Enabled":true,
				"Segments":{
					"Attributes":{"Enabled":true,"Exclude":null,"Include":null},
					"StackTraceThreshold":500000000,
					"Threshold":2000000
				},
				"Threshold":{
					"Duration":500000000,
					"IsApdexFailing":true
				}
			},
			"Transport":null,
			"Utilization":{
				"BillingHostname":"",
				"DetectAWS":true,
				"DetectAzure":true,
				"DetectDocker":true,
				"DetectGCP":true,
				"DetectKubernetes":true,
				"DetectPCF":true,
				"LogicalProcessors":0,
				"TotalRAMMIB":0
			},
			"browser_monitoring.loader":"rum"
		},
		"app_name":["my appname"],
		"high_security":false,
		"environment":[
			["runtime.NumCPU",8],
			["runtime.Compiler","comp"],
			["runtime.GOARCH","arch"],
			["runtime.GOOS","goos"],
			["runtime.Version","vers"],
			["Modules", null]
		],
		"identifier":"my appname",
		"utilization":{
			"metadata_version":5,
			"logical_processors":16,
			"total_ram_mib":1024,
			"hostname":"my-hostname"
		},
		"metadata":{},
		"event_harvest_config": {
			"report_period_ms": 60000,
			"harvest_limits": {
				"analytic_event_data": 10000,
				"custom_event_data": %d,
				"log_event_data": %d,
				"error_event_data": 100,
				"span_event_data": %d
			}
		}
	}]`, internal.MaxLogEvents, internal.MaxCustomEvents, internal.MaxSpanEvents, internal.MaxTxnEvents, internal.MaxCustomEvents, internal.MaxTxnEvents, internal.MaxSpanEvents))

	metadata := map[string]string{}
	js, err := configConnectJSONInternal(cp, 123, &utilization.SampleData, sampleEnvironment, "0.2.2", nil, metadata)
	if nil != err {
		t.Fatal(err)
	}
	out := standardizeNumbers(string(js))
	if out != expect {
		t.Error(string(js))
	}
}

func TestValidate(t *testing.T) {
	c := Config{
		License: "0123456789012345678901234567890123456789",
		AppName: "my app",
		Enabled: true,
	}
	if err := c.validate(); err != nil {
		t.Error(err)
	}
	c = Config{
		License: "",
		AppName: "my app",
		Enabled: true,
	}
	if err := c.validate(); err != errLicenseLen {
		t.Error(err)
	}
	c = Config{
		License: "",
		AppName: "my app",
		Enabled: false,
	}
	if err := c.validate(); err != nil {
		t.Error(err)
	}
	c = Config{
		License: "wronglength",
		AppName: "my app",
		Enabled: true,
	}
	if err := c.validate(); err != errLicenseLen {
		t.Error(err)
	}
	c = Config{
		License: "0123456789012345678901234567890123456789",
		AppName: "too;many;app;names",
		Enabled: true,
	}
	if err := c.validate(); err != errAppNameLimit {
		t.Error(err)
	}
	c = Config{
		License: "0123456789012345678901234567890123456789",
		AppName: "",
		Enabled: true,
	}
	if err := c.validate(); err != errAppNameMissing {
		t.Error(err)
	}
	c = Config{
		License: "0123456789012345678901234567890123456789",
		AppName: "",
		Enabled: false,
	}
	if err := c.validate(); err != nil {
		t.Error(err)
	}
	c = Config{
		License:      "0123456789012345678901234567890123456789",
		AppName:      "my app",
		Enabled:      true,
		HighSecurity: true,
	}
	if err := c.validate(); err != nil {
		t.Error(err)
	}
}

func TestValidateCalled(t *testing.T) {
	// Test that config validation is actually done when creating an
	// application.
	app, err := NewApplication(func(cfg *Config) {
		cfg.License = ""
		cfg.AppName = "my app"
		cfg.Enabled = true
	})
	if app != nil {
		t.Error(app)
	}
	if err != errLicenseLen {
		t.Error(err)
	}
}

func TestValidateWithPoliciesToken(t *testing.T) {
	c := Config{
		License:               "0123456789012345678901234567890123456789",
		AppName:               "my app",
		Enabled:               true,
		HighSecurity:          true,
		SecurityPoliciesToken: "0123456789",
	}
	if err := c.validate(); err != errHighSecurityWithSecurityPolicies {
		t.Error(err)
	}
	c = Config{
		License:               "0123456789012345678901234567890123456789",
		AppName:               "my app",
		Enabled:               true,
		SecurityPoliciesToken: "0123456789",
	}
	if err := c.validate(); err != nil {
		t.Error(err)
	}
}

func TestGatherMetadata(t *testing.T) {
	metadata := gatherMetadata(nil)
	if !reflect.DeepEqual(metadata, map[string]string{}) {
		t.Error(metadata)
	}
	metadata = gatherMetadata([]string{
		"NEW_RELIC_METADATA_ZIP=zap",
		"NEW_RELIC_METADATA_PIZZA=cheese",
		"NEW_RELIC_METADATA_=hello",
		"NEW_RELIC_METADATA_LOTS_OF_EQUALS=one=two",
		"NEW_RELIC_METADATA_",
		"NEW_RELIC_METADATA_NO_EQUALS",
		"NEW_RELIC_METADATA_EMPTY=",
		"NEW_RELIC_",
		"hello=world",
	})
	if !reflect.DeepEqual(metadata, map[string]string{
		"NEW_RELIC_METADATA_ZIP":            "zap",
		"NEW_RELIC_METADATA_PIZZA":          "cheese",
		"NEW_RELIC_METADATA_":               "hello",
		"NEW_RELIC_METADATA_LOTS_OF_EQUALS": "one=two",
		"NEW_RELIC_METADATA_EMPTY":          "",
	}) {
		t.Error(metadata)
	}
}

func TestValidateServerless(t *testing.T) {
	// AppName and License can be empty in serverless mode.
	c := defaultConfig()
	c.ServerlessMode.Enabled = true
	if err := c.validate(); nil != err {
		t.Error(err)
	}
}

func TestPreconnectHost(t *testing.T) {
	testcases := []struct {
		license  string
		override string
		expect   string
	}{
		{ // non-region license
			license:  "0123456789012345678901234567890123456789",
			override: "",
			expect:   preconnectHostDefault,
		},
		{ // override present
			license:  "0123456789012345678901234567890123456789",
			override: "other-collector.newrelic.com",
			expect:   "other-collector.newrelic.com",
		},
		{ // four letter region
			license:  "eu01xx6789012345678901234567890123456789",
			override: "",
			expect:   "collector.eu01.nr-data.net",
		},
		{ // five letter region
			license:  "gov01x6789012345678901234567890123456789",
			override: "",
			expect:   "collector.gov01.nr-data.net",
		},
		{ // six letter region
			license:  "foo001x6789012345678901234567890123456789",
			override: "",
			expect:   "collector.foo001.nr-data.net",
		},
	}
	for idx, tc := range testcases {
		cfg := config{Config: Config{
			License: tc.license,
			Host:    tc.override,
		}}
		if got := cfg.preconnectHost(); got != tc.expect {
			t.Error("testcase", idx, got, tc.expect)
		}
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
		if tc.EnvKey != "" {
			configKey = tc.EnvKey
		}
		overrideHost := tc.ConfigOverrideHost
		if tc.EnvOverrideHost != "" {
			overrideHost = tc.EnvOverrideHost
		}

		cfg := config{Config: Config{
			License: configKey,
			Host:    overrideHost,
		}}
		if host := cfg.preconnectHost(); host != tc.ExpectHostname {
			t.Errorf(`test="%s" got="%s" expected="%s"`, tc.Name, host, tc.ExpectHostname)
		}
	}
}

func TestConfigMaxTxnEvents(t *testing.T) {
	cfg := defaultConfig()
	if n := cfg.maxTxnEvents(); n != internal.MaxTxnEvents {
		t.Error(n)
	}

	cfg = defaultConfig()
	cfg.TransactionEvents.MaxSamplesStored = 434
	if n := cfg.maxTxnEvents(); n != 434 {
		t.Error(n)
	}

	cfg = defaultConfig()
	cfg.TransactionEvents.MaxSamplesStored = 0
	if n := cfg.maxTxnEvents(); n != 0 {
		t.Error(n)
	}

	cfg = defaultConfig()
	cfg.TransactionEvents.MaxSamplesStored = -1
	if n := cfg.maxTxnEvents(); n != internal.MaxTxnEvents {
		t.Error(n)
	}

	cfg = defaultConfig()
	cfg.TransactionEvents.MaxSamplesStored = internal.MaxTxnEvents + 1
	if n := cfg.maxTxnEvents(); n != internal.MaxTxnEvents {
		t.Error(n)
	}
}

func TestComputeDynoHostname(t *testing.T) {
	testcases := []struct {
		useDynoNames     bool
		dynoNamePrefixes []string
		envVarValue      string
		expected         string
	}{
		{
			useDynoNames: false,
			envVarValue:  "dynoname",
			expected:     "",
		},
		{
			useDynoNames: true,
			envVarValue:  "",
			expected:     "",
		},
		{
			useDynoNames: true,
			envVarValue:  "dynoname",
			expected:     "dynoname",
		},
		{
			useDynoNames:     true,
			dynoNamePrefixes: []string{"example"},
			envVarValue:      "dynoname",
			expected:         "dynoname",
		},
		{
			useDynoNames:     true,
			dynoNamePrefixes: []string{""},
			envVarValue:      "dynoname",
			expected:         "dynoname",
		},
		{
			useDynoNames:     true,
			dynoNamePrefixes: []string{"example", "ex"},
			envVarValue:      "example.asdfasdfasdf",
			expected:         "example.*",
		},
		{
			useDynoNames:     true,
			dynoNamePrefixes: []string{"example", "ex"},
			envVarValue:      "exampleasdfasdfasdf",
			expected:         "exampleasdfasdfasdf",
		},
	}

	for _, test := range testcases {
		getenv := func(string) string { return test.envVarValue }
		cfg := Config{}
		cfg.Heroku.UseDynoNames = test.useDynoNames
		cfg.Heroku.DynoNamePrefixesToShorten = test.dynoNamePrefixes
		if actual := cfg.computeDynoHostname(getenv); actual != test.expected {
			t.Errorf("unexpected output: actual=%s expected=%s", actual, test.expected)
		}
	}
}

func TestNewInternalConfig(t *testing.T) {
	labels := map[string]string{"zip": "zap"}
	cfg := defaultConfig()
	cfg.License = "0123456789012345678901234567890123456789"
	cfg.AppName = "my app"
	cfg.Logger = nil
	cfg.Labels = labels
	c, err := newInternalConfig(cfg, func(s string) string {
		switch s {
		case "DYNO":
			return "mydyno"
		}
		return ""
	}, []string{"NEW_RELIC_METADATA_ZIP=ZAP"})
	if err != nil {
		t.Error(err)
	}
	if c.Logger == nil {
		t.Error("non nil Logger expected")
	}
	labels["zip"] = "1234"
	if c.Labels["zip"] != "zap" {
		t.Error("labels should have been copied", c.Labels)
	}
	if c.hostname != "mydyno" {
		t.Error(c.hostname)
	}
	if !reflect.DeepEqual(c.metadata, map[string]string{
		"NEW_RELIC_METADATA_ZIP": "ZAP",
	}) {
		t.Error(c.metadata)
	}
}

func TestConfigurableMaxCustomEvents(t *testing.T) {
	expected := 1000
	cfg := config{Config: defaultConfig()}
	cfg.CustomInsightsEvents.MaxSamplesStored = expected
	result := cfg.maxCustomEvents()
	if result != expected {
		t.Errorf("Unexpected max number of custom events, expected %d but got %d", expected, result)
	}
}

func TestCLMScopeLabels(t *testing.T) {
	for i, tc := range []struct {
		L  []string
		LL string
		V  CodeLevelMetricsScope
		OK bool
	}{
		{V: AllCLM, OK: true},
		{L: []string{"all"}, LL: "all", V: AllCLM, OK: true},
		{L: []string{"transactions"}, LL: "transactions", V: TransactionCLM, OK: true},
		{L: []string{"transaction"}, LL: "transaction", V: TransactionCLM, OK: true},
		{L: []string{"txn"}, LL: "txn", V: TransactionCLM, OK: true},
		{L: []string{"all", "txn"}, LL: "all,txn", V: AllCLM, OK: true},
		{L: []string{"undefined"}, LL: "undefined", OK: false},
	} {
		s, ok := CodeLevelMetricsScopeLabelToValue(tc.L...)
		if ok != tc.OK {
			t.Errorf("#%d for \"%v\" expected ok=%v", i, tc.L, tc.OK)
		}
		if s != tc.V {
			t.Errorf("#%d for \"%v\" expected output %v, but got %v", i, tc.L, tc.V, s)
		}

		ss, ok := CodeLevelMetricsScopeLabelListToValue(tc.LL)
		if ok != tc.OK {
			t.Errorf("#%d for \"%v\" expected ok=%v", i, tc.L, tc.OK)
		}
		if ss != tc.V {
			t.Errorf("#%d for \"%v\" expected output %v, but got %v", i, tc.L, tc.V, ss)
		}
	}
}

func TestCLMJsonMarshalling(t *testing.T) {
	var s CodeLevelMetricsScope

	for i, tc := range []struct {
		S CodeLevelMetricsScope
		J string
		E bool
	}{
		{S: AllCLM, J: `"all"`},
		{S: TransactionCLM, J: `"transaction"`},
		{S: 0x500, E: true},
	} {
		s = tc.S
		j, err := json.Marshal(s)
		if err != nil {
			if !tc.E {
				t.Errorf("#%d generated unexpected error %v", i, err)
			}
		} else {
			if tc.E {
				t.Errorf("#%d was supposed to generate an error but didn't", i)
			}
			if tc.J != string(j) {
				t.Errorf("#%d expected \"%v\" but got \"%v\"", i, tc.J, string(j))
			}
		}
	}
}

func TestCLMJsonUnmarshalling(t *testing.T) {
	var s CodeLevelMetricsScope

	for i, tc := range []struct {
		S CodeLevelMetricsScope
		J string
		E bool
	}{
		{S: AllCLM, J: `"all"`},
		{S: TransactionCLM, J: `"transaction"`},
		{S: TransactionCLM, J: `"transaction,"`},
		{S: TransactionCLM, J: `"transaction,txn"`},
		{S: AllCLM, J: `"transaction,all,txn"`},
		{S: AllCLM, J: `""`},
		{S: AllCLM, J: `null`},
		{S: AllCLM, J: `"blorfl"`, E: true},
	} {
		err := json.Unmarshal([]byte(tc.J), &s)

		if err != nil {
			if !tc.E {
				t.Errorf("#%d generated unexpected error %v", i, err)
			}
		} else {
			if tc.E {
				t.Errorf("#%d was supposed to generate an error but didn't", i)
			}
			if tc.S != s {
				t.Errorf("#%d expected \"%v\" but got \"%v\"", i, tc.S, s)
			}
		}
	}
}
