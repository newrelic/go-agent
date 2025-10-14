// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"reflect"
	"testing"
)

func TestConfigFromEnvironment(t *testing.T) {
	cfgOpt := configFromEnvironment(func(s string) string {
		switch s {
		case "NEW_RELIC_APP_NAME":
			return "my app"
		case "NEW_RELIC_LICENSE_KEY":
			return "my license"
		case "NEW_RELIC_DISTRIBUTED_TRACING_ENABLED":
			return "true"
		case "NEW_RELIC_ENABLED":
			return "false"
		case "NEW_RELIC_HIGH_SECURITY":
			return "1"
		case "NEW_RELIC_SECURITY_POLICIES_TOKEN":
			return "my token"
		case "NEW_RELIC_HOST":
			return "my host"
		case "NEW_RELIC_PROCESS_HOST_DISPLAY_NAME":
			return "my display host"
		case "NEW_RELIC_UTILIZATION_BILLING_HOSTNAME":
			return "my billing hostname"
		case "NEW_RELIC_UTILIZATION_LOGICAL_PROCESSORS":
			return "123"
		case "NEW_RELIC_UTILIZATION_TOTAL_RAM_MIB":
			return "456"
		case "NEW_RELIC_LABELS":
			return "star:car;far:bar"
		case "NEW_RELIC_ATTRIBUTES_INCLUDE":
			return "zip,zap"
		case "NEW_RELIC_ATTRIBUTES_EXCLUDE":
			return "zop,zup,zep"
		case "NEW_RELIC_INFINITE_TRACING_TRACE_OBSERVER_HOST":
			return "myhost.com"
		case "NEW_RELIC_INFINITE_TRACING_TRACE_OBSERVER_PORT":
			return "456"
		case "NEW_RELIC_INFINITE_TRACING_SPAN_EVENTS_QUEUE_SIZE":
			return "98765"
		case "NEW_RELIC_CODE_LEVEL_METRICS_SCOPE":
			return "all"
		case "NEW_RELIC_CODE_LEVEL_METRICS_PATH_PREFIX":
			return "/foo/bar,/spam/spam/spam/frotz"
		case "NEW_RELIC_CODE_LEVEL_METRICS_IGNORED_PREFIX":
			return "/a/b,/c/d"
		case "NEW_RELIC_APPLICATION_LOGGING_ENABLED":
			return "false"
		}
		return ""
	})
	expect := defaultConfig()
	expect.AppName = "my app"
	expect.License = "my license"
	expect.DistributedTracer.Enabled = true
	expect.Enabled = false
	expect.HighSecurity = true
	expect.SecurityPoliciesToken = "my token"
	expect.Host = "my host"
	expect.HostDisplayName = "my display host"
	expect.Utilization.BillingHostname = "my billing hostname"
	expect.Utilization.LogicalProcessors = 123
	expect.Utilization.TotalRAMMIB = 456
	expect.Labels = map[string]string{"star": "car", "far": "bar"}
	expect.Attributes.Include = []string{"zip", "zap"}
	expect.Attributes.Exclude = []string{"zop", "zup", "zep"}
	expect.InfiniteTracing.TraceObserver.Host = "myhost.com"
	expect.InfiniteTracing.TraceObserver.Port = 456
	expect.InfiniteTracing.SpanEvents.QueueSize = 98765
	expect.CodeLevelMetrics.Scope = AllCLM
	expect.CodeLevelMetrics.PathPrefixes = []string{"/foo/bar", "/spam/spam/spam/frotz"}
	expect.CodeLevelMetrics.IgnoredPrefixes = []string{"/a/b", "/c/d"}

	expect.ApplicationLogging.Enabled = false
	expect.ApplicationLogging.Forwarding.Enabled = true
	expect.ApplicationLogging.Metrics.Enabled = true
	expect.ApplicationLogging.LocalDecorating.Enabled = false

	cfg := defaultConfig()
	cfgOpt(&cfg)

	if !reflect.DeepEqual(expect, cfg) {
		t.Errorf("%+v", cfg)
	}
}

func TestConfigFromEnvironmentIgnoresUnset(t *testing.T) {
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

func TestConfigFromEnvironmentInvalidBool(t *testing.T) {
	cfgOpt := configFromEnvironment(func(s string) string {
		switch s {
		case "NEW_RELIC_ENABLED":
			return "BOGUS"
		default:
			return ""
		}
	})
	cfg := defaultConfig()
	cfgOpt(&cfg)
	if cfg.Error == nil {
		t.Error("error expected")
	}
}

func TestConfigFromEnvironmentInvalidInt(t *testing.T) {
	cfgOpt := configFromEnvironment(func(s string) string {
		switch s {
		case "NEW_RELIC_UTILIZATION_LOGICAL_PROCESSORS":
			return "BOGUS"
		default:
			return ""
		}
	})
	cfg := defaultConfig()
	cfgOpt(&cfg)
	if cfg.Error == nil {
		t.Error("error expected")
	}
}

func TestConfigFromEnvironmentInvalidLogger(t *testing.T) {
	cfgOpt := configFromEnvironment(func(s string) string {
		switch s {
		case "NEW_RELIC_LOG":
			return "BOGUS"
		default:
			return ""
		}
	})
	cfg := defaultConfig()
	cfgOpt(&cfg)
	if cfg.Error == nil {
		t.Error("error expected")
	}
}

func TestConfigFromEnvironmentInvalidLabels(t *testing.T) {
	cfgOpt := configFromEnvironment(func(s string) string {
		switch s {
		case "NEW_RELIC_LABELS":
			return ";;;"
		default:
			return ""
		}
	})
	cfg := defaultConfig()
	cfgOpt(&cfg)
	if cfg.Error == nil {
		t.Error("error expected")
	}
}

func TestConfigFromEnvironmentLabelsSuccess(t *testing.T) {
	cfgOpt := configFromEnvironment(func(s string) string {
		switch s {
		case "NEW_RELIC_LABELS":
			return "zip:zap; zop:zup"
		default:
			return ""
		}
	})
	cfg := defaultConfig()
	cfgOpt(&cfg)
	if !reflect.DeepEqual(cfg.Labels, map[string]string{"zip": "zap", "zop": "zup"}) {
		t.Error(cfg.Labels)
	}
}

func TestConfigRemoteParentSamplingDefaults(t *testing.T) {
	cfg := defaultConfig()
	if cfg.DistributedTracer.Sampler.RemoteParentNotSampled != "default" {
		t.Error("incorrect config value for DistributedTracer.Sampler.RemoteParentNotSampled:", cfg.DistributedTracer.Sampler.RemoteParentNotSampled)
	}
	if cfg.DistributedTracer.Sampler.RemoteParentSampled != "default" {
		t.Error("incorrect config value for DistributedTracer.Sampler.RemoteParentSampled:", cfg.DistributedTracer.Sampler.RemoteParentSampled)
	}
}

func TestConfigRemoteParentSampledOn(t *testing.T) {
	cfgOpt := ConfigRemoteParentSampled(AlwaysOn)
	cfg := defaultConfig()
	cfgOpt(&cfg)
	if cfg.DistributedTracer.Sampler.RemoteParentSampled != "always_on" {
		t.Error("incorrect config value for DistributedTracer.Sampler.RemoteParentSampled:", cfg.DistributedTracer.Sampler.RemoteParentSampled)
	}
}

func TestConfigRemoteParentSampledOff(t *testing.T) {
	cfgOpt := ConfigRemoteParentSampled(AlwaysOff)
	cfg := defaultConfig()
	cfgOpt(&cfg)
	if cfg.DistributedTracer.Sampler.RemoteParentSampled != "always_off" {
		t.Error("incorrect config value for DistributedTracer.Sampler.RemoteParentSampled:", cfg.DistributedTracer.Sampler.RemoteParentSampled)
	}
}

func TestConfigRemoteParentNotSampledOn(t *testing.T) {
	cfgOpt := ConfigRemoteParentNotSampled(AlwaysOn)
	cfg := defaultConfig()
	cfgOpt(&cfg)
	if cfg.DistributedTracer.Sampler.RemoteParentNotSampled != "always_on" {
		t.Error("incorrect config value for DistributedTracer.Sampler.RemoteParentNotSampled:", cfg.DistributedTracer.Sampler.RemoteParentNotSampled)
	}
}

func TestConfigRemoteParentNotSampledOff(t *testing.T) {
	cfgOpt := ConfigRemoteParentNotSampled(AlwaysOff)
	cfg := defaultConfig()
	cfgOpt(&cfg)
	if cfg.DistributedTracer.Sampler.RemoteParentNotSampled != "always_off" {
		t.Error("incorrect config value for DistributedTracer.Sampler.RemoteParentNotSampled:", cfg.DistributedTracer.Sampler.RemoteParentNotSampled)
	}
}

func TestConfigSpanEventsMaxSamplesStored(t *testing.T) {
	// these tests assume internal.MaxSpanEvents = 2000
	tests := []struct {
		name  string // description of this test case
		limit int    // limit that is being passed in
		want  int
	}{
		{
			name:  "MaxSamplesStored is less than 0",
			limit: -1,
			want:  2000,
		},
		{
			name:  "MaxSamplesStored is greater than 2000",
			limit: 2001,
			want:  2000,
		},
		{
			name:  "MaxSamplesStored is much greater than 2000",
			limit: 100000,
			want:  2000,
		},
		{
			name:  "MaxSamplesStored is between 0 and 2000",
			limit: 500,
			want:  500,
		},
		{
			name:  "MaxSamplesStored is 0",
			limit: 0,
			want:  0,
		},
		{
			name:  "MaxSamplesStored is 2000",
			limit: 2000,
			want:  2000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgOpt := ConfigSpanEventsMaxSamplesStored(tt.limit)
			cfg := defaultConfig()
			cfgOpt(&cfg)
			if cfg.SpanEvents.MaxSamplesStored != tt.want {
				t.Errorf("cfg.SpanEvents.MaxSamplesStored = %v, want %v", cfg.SpanEvents.MaxSamplesStored, tt.want)
			}
		})
		// Should be the same result if using the wrapped function
		t.Run(tt.name, func(t *testing.T) {
			cfgOpt := ConfigDistributedTracerReservoirLimit(tt.limit)
			cfg := defaultConfig()
			cfgOpt(&cfg)
			if cfg.SpanEvents.MaxSamplesStored != tt.want {
				t.Errorf("cfg.SpanEvents.MaxSamplesStored = %v, want %v", cfg.SpanEvents.MaxSamplesStored, tt.want)
			}
		})
	}
}
