package newrelic

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/newrelic/go-agent/v3/internal/crossagent"
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

func TestConfigFromEnvironment(t *testing.T) {
	// test that configFromEnvironment ignores unset env vars
	cfgOpt := configFromEnvironment(func(string) string { return "" })
	cfg := defaultConfig()
	cfg.AppName = "something"
	cfg.Labels = map[string]string{"hello": "world"}
	cfg.Attributes.Include = []string{"zip", "zap"}
	cfg.Attributes.Exclude = []string{"zop", "zup", "zep"}
	cfg.License = "something"
	cfg.DistributedTracer.Enabled = true
	cfg.HighSecurity = true
	cfg.Host = "something"
	cfg.HostDisplayName = "something"
	cfg.SecurityPoliciesToken = "something"
	cfg.Utilization.BillingHostname = "something"
	cfg.Utilization.LogicalProcessors = 42
	cfg.Utilization.TotalRAMMIB = 42

	cfgOpt(&cfg)

	if cfg.AppName != "something" {
		t.Error("config value changed:", cfg.AppName)
	}
	if len(cfg.Labels) != 1 {
		t.Error("config value changed:", cfg.Labels)
	}
	if cfg.License != "something" {
		t.Error("config value changed:", cfg.License)
	}
	if !cfg.DistributedTracer.Enabled {
		t.Error("config value changed:", cfg.DistributedTracer.Enabled)
	}
	if !cfg.HighSecurity {
		t.Error("config value changed:", cfg.HighSecurity)
	}
	if cfg.Host != "something" {
		t.Error("config value changed:", cfg.Host)
	}
	if cfg.HostDisplayName != "something" {
		t.Error("config value changed:", cfg.HostDisplayName)
	}
	if cfg.SecurityPoliciesToken != "something" {
		t.Error("config value changed:", cfg.SecurityPoliciesToken)
	}
	if cfg.Utilization.BillingHostname != "something" {
		t.Error("config value changed:", cfg.Utilization.BillingHostname)
	}
	if cfg.Utilization.LogicalProcessors != 42 {
		t.Error("config value changed:", cfg.Utilization.LogicalProcessors)
	}
	if cfg.Utilization.TotalRAMMIB != 42 {
		t.Error("config value changed:", cfg.Utilization.TotalRAMMIB)
	}
	if len(cfg.Attributes.Include) != 2 {
		t.Error("config value changed:", cfg.Attributes.Include)
	}
	if len(cfg.Attributes.Exclude) != 3 {
		t.Error("config value changed:", cfg.Attributes.Exclude)
	}
}

func TestConfigFromEnvironmentAttributes(t *testing.T) {
	cfgOpt := configFromEnvironment(func(s string) string {
		switch s {
		case "NEW_RELIC_ATTRIBUTES_INCLUDE":
			return "zip,zap"
		case "NEW_RELIC_ATTRIBUTES_EXCLUDE":
			return "zop,zup,zep"
		default:
			return ""
		}
	})
	cfg := defaultConfig()
	cfgOpt(&cfg)
	if !reflect.DeepEqual(cfg.Attributes.Include, []string{"zip", "zap"}) {
		t.Error("incorrect config value:", cfg.Attributes.Include)
	}
	if !reflect.DeepEqual(cfg.Attributes.Exclude, []string{"zop", "zup", "zep"}) {
		t.Error("incorrect config value:", cfg.Attributes.Exclude)
	}
}
