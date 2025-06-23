package nrsecurityagent

import (
	"os"
	"testing"
)

func TestDefaultSecurityConfig(t *testing.T) {
	cfg := defaultSecurityConfig()
	if cfg.Security.Enabled {
		t.Error("Expected Security.Enabled to be false by default")
	}
	if cfg.Security.Validator_service_url != "wss://csec.nr-data.net" {
		t.Errorf("Unexpected Validator_service_url: %s", cfg.Security.Validator_service_url)
	}
	if cfg.Security.Mode != "IAST" {
		t.Errorf("Unexpected Mode: %s", cfg.Security.Mode)
	}
	if !cfg.Security.Agent.Enabled {
		t.Error("Expected Agent.Enabled to be true by default")
	}
	if !cfg.Security.Detection.Rxss.Enabled {
		t.Error("Expected Detection.Rxss.Enabled to be true by default")
	}
	if cfg.Security.Request.BodyLimit != 300 {
		t.Errorf("Unexpected Request.BodyLimit: %d", cfg.Security.Request.BodyLimit)
	}
}

func TestIsSecurityAgentEnabled(t *testing.T) {
	os.Setenv("NEW_RELIC_SECURITY_AGENT_ENABLED", "false")
	defer os.Unsetenv("NEW_RELIC_SECURITY_AGENT_ENABLED")
	if isSecurityAgentEnabled() {
		t.Error("Expected isSecurityAgentEnabled to return false when env is set to false")
	}
	os.Setenv("NEW_RELIC_SECURITY_AGENT_ENABLED", "true")
	if !isSecurityAgentEnabled() {
		t.Error("Expected isSecurityAgentEnabled to return true when env is set to true")
	}
	os.Unsetenv("NEW_RELIC_SECURITY_AGENT_ENABLED")
	if !isSecurityAgentEnabled() {
		t.Error("Expected isSecurityAgentEnabled to return true when env is unset")
	}
}

func TestConfigSecurityFromEnvironment(t *testing.T) {
	os.Setenv("NEW_RELIC_SECURITY_ENABLED", "true")
	os.Setenv("NEW_RELIC_SECURITY_MODE", "TEST")
	defer os.Unsetenv("NEW_RELIC_SECURITY_ENABLED")
	defer os.Unsetenv("NEW_RELIC_SECURITY_MODE")

	cfg := defaultSecurityConfig()
	ConfigSecurityFromEnvironment()(&cfg)
	if !cfg.Security.Enabled {
		t.Error("Expected Security.Enabled to be true from env")
	}
	if cfg.Security.Mode != "TEST" {
		t.Errorf("Expected Security.Mode to be TEST, got %s", cfg.Security.Mode)
	}
}

func TestConfigSecurityMode(t *testing.T) {
	cfg := defaultSecurityConfig()
	ConfigSecurityMode("CUSTOM")(&cfg)
	if cfg.Security.Mode != "CUSTOM" {
		t.Errorf("Expected Security.Mode to be CUSTOM, got %s", cfg.Security.Mode)
	}
}

func TestConfigSecurityEnable(t *testing.T) {
	cfg := defaultSecurityConfig()
	ConfigSecurityEnable(true)(&cfg)
	if !cfg.Security.Enabled {
		t.Error("Expected Security.Enabled to be true")
	}
}
