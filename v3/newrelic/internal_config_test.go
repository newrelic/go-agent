package newrelic

import (
	"encoding/json"
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

	expect := internal.CompactJSONString(`[
	{
		"pid":123,
		"language":"go",
		"agent_version":"0.2.2",
		"host":"my-hostname",
		"settings":{
			"AppName":"my appname",
			"Attributes":{"Enabled":true,"Exclude":["2"],"Include":["1"]},
			"BrowserMonitoring":{
				"Attributes":{"Enabled":false,"Exclude":["10"],"Include":["9"]},
				"Enabled":true
			},
			"CrossApplicationTracer":{"Enabled":true},
			"CustomInsightsEvents":{"Enabled":true},
			"DatastoreTracer":{
				"DatabaseNameReporting":{"Enabled":true},
				"InstanceReporting":{"Enabled":true},
				"QueryParameters":{"Enabled":true},
				"SlowQuery":{
					"Enabled":true,
					"Threshold":10000000
				}
			},
			"DistributedTracer":{"Enabled":false},
			"Enabled":true,
			"Error":null,
			"ErrorCollector":{
				"Attributes":{"Enabled":true,"Exclude":["6"],"Include":["5"]},
				"CaptureEvents":true,
				"Enabled":true,
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
			"Labels":{"zip":"zap"},
			"Logger":"*logger.logFile",
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
				"MaxSamplesStored": 10000
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
			["runtime.Compiler","comp"],
			["runtime.GOARCH","arch"],
			["runtime.GOOS","goos"],
			["runtime.Version","vers"],
			["runtime.NumCPU",8]
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
				"custom_event_data": 10000,
				"error_event_data": 100
			}
		}
	}]`)

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
	js, err := configConnectJSONInternal(cp, 123, &utilization.SampleData, internal.SampleEnvironment, "0.2.2", sp.PointerIfPopulated(), metadata)
	if nil != err {
		t.Fatal(err)
	}
	out := standardizeNumbers(string(js))
	if out != expect {
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

	expect := internal.CompactJSONString(`[
	{
		"pid":123,
		"language":"go",
		"agent_version":"0.2.2",
		"host":"my-hostname",
		"settings":{
			"AppName":"my appname",
			"Attributes":{"Enabled":true,"Exclude":null,"Include":null},
			"BrowserMonitoring":{
				"Attributes":{
					"Enabled":false,
					"Exclude":null,
					"Include":null
				},
				"Enabled":true
			},
			"CrossApplicationTracer":{"Enabled":true},
			"CustomInsightsEvents":{"Enabled":true},
			"DatastoreTracer":{
				"DatabaseNameReporting":{"Enabled":true},
				"InstanceReporting":{"Enabled":true},
				"QueryParameters":{"Enabled":true},
				"SlowQuery":{
					"Enabled":true,
					"Threshold":10000000
				}
			},
			"DistributedTracer":{"Enabled":false},
			"Enabled":true,
			"Error":null,
			"ErrorCollector":{
				"Attributes":{"Enabled":true,"Exclude":null,"Include":null},
				"CaptureEvents":true,
				"Enabled":true,
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
			"Labels":null,
			"Logger":null,
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
				"MaxSamplesStored": 10000
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
			["runtime.Compiler","comp"],
			["runtime.GOARCH","arch"],
			["runtime.GOOS","goos"],
			["runtime.Version","vers"],
			["runtime.NumCPU",8]
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
				"custom_event_data": 10000,
				"error_event_data": 100
			}
		}
	}]`)

	metadata := map[string]string{}
	js, err := configConnectJSONInternal(cp, 123, &utilization.SampleData, internal.SampleEnvironment, "0.2.2", nil, metadata)
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
	if err := c.validate(); nil != err {
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
	if err := c.validate(); nil != err {
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
		if got := cfg.PreconnectHost(); got != tc.expect {
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
		if "" != tc.EnvKey {
			configKey = tc.EnvKey
		}
		overrideHost := tc.ConfigOverrideHost
		if "" != tc.EnvOverrideHost {
			overrideHost = tc.EnvOverrideHost
		}

		cfg := config{Config: Config{
			License: configKey,
			Host:    overrideHost,
		}}
		if host := cfg.PreconnectHost(); host != tc.ExpectHostname {
			t.Errorf(`test="%s" got="%s" expected="%s"`, tc.Name, host, tc.ExpectHostname)
		}
	}
}

func TestConfigSecurityPoliciesTokenMethod(t *testing.T) {
	token := "mySecurityToken01234"
	cfg := config{Config: Config{SecurityPoliciesToken: token}}
	if got := cfg.SecurityPoliciesToken(); got != token {
		t.Error(got, token)
	}
}

func TestConfigHighSecurityMethod(t *testing.T) {
	cfg := config{Config: Config{HighSecurity: true}}
	if s := cfg.HighSecurity(); !s {
		t.Error(s)
	}
	cfg = config{Config: Config{HighSecurity: false}}
	if s := cfg.HighSecurity(); s {
		t.Error(s)
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
	c := newInternalConfig(defaultConfig(), func(s string) string {
		switch s {
		case "DYNO":
			return "mydyno"
		}
		return ""
	}, []string{"NEW_RELIC_METADATA_ZIP=ZAP"})
	if c.hostname != "mydyno" {
		t.Error(c.hostname)
	}
	if !reflect.DeepEqual(c.metadata, map[string]string{
		"NEW_RELIC_METADATA_ZIP": "ZAP",
	}) {
		t.Error(c.metadata)
	}
}
