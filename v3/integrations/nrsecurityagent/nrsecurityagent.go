// Copyright 2022 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrsecurityagent

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	securityAgent "github.com/newrelic/csec-go-agent"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"gopkg.in/yaml.v2"
)

func init() { internal.TrackUsage("integration", "securityagent") }

type SecurityConfig struct {
	securityAgent.SecurityAgentConfig
	Error error
}

// defaultSecurityConfig creates a SecurityConfig value populated with default settings.
func defaultSecurityConfig() SecurityConfig {
	cfg := SecurityConfig{}
	cfg.Security.Enabled = false
	cfg.Security.Validator_service_url = "wss://csec.nr-data.net"
	cfg.Security.Mode = "IAST"
	cfg.Security.Agent.Enabled = true
	cfg.Security.Detection.Rxss.Enabled = true
	cfg.Security.Request.BodyLimit = 300
	return cfg
}

// To completely disable security set NEW_RELIC_SECURITY_AGENT_ENABLED env to false.
// If env is set to false,the security module is not loaded
func isSecurityAgentEnabled() bool {
	if env := os.Getenv("NEW_RELIC_SECURITY_AGENT_ENABLED"); env != "" {
		if b, err := strconv.ParseBool(env); err == nil {
			return b
		}
	}
	return true
}

// InitSecurityAgent initializes the nrsecurityagent integration package from user-supplied
// configuration values.
func InitSecurityAgent(app *newrelic.Application, opts ...ConfigOption) error {
	if app == nil {
		return fmt.Errorf("Newrelic application value cannot be nil; did you call newrelic.NewApplication?")
	}
	c := defaultSecurityConfig()
	for _, fn := range opts {
		if fn != nil {
			fn(&c)
			if c.Error != nil {
				return c.Error
			}
		}
	}

	appConfig, isValid := app.Config()
	if !isValid {
		return fmt.Errorf("Newrelic application value cannot be read; did you call newrelic.NewApplication?")
	}
	if !appConfig.HighSecurity && isSecurityAgentEnabled() {
		secureAgent := securityAgent.InitSecurityAgent(c.Security, appConfig.AppName, appConfig.License, appConfig.Logger.DebugEnabled())
		app.RegisterSecurityAgent(secureAgent)
	}
	return nil
}

// ConfigOption functions are used to programmatically provide configuration values to the
// nrsecurityagent integration package.
type ConfigOption func(*SecurityConfig)

// ConfigSecurityFromYaml directs the nrsecurityagent integration to read an external
// YAML-formatted file to obtain its configuration values.
//
// The path to this file must be provided by setting the environment variable NEW_RELIC_SECURITY_CONFIG_PATH.
func ConfigSecurityFromYaml() ConfigOption {
	return func(cfg *SecurityConfig) {
		confgFilePath := os.Getenv("NEW_RELIC_SECURITY_CONFIG_PATH")
		if confgFilePath == "" {
			cfg.Error = fmt.Errorf("Invalid value: NEW_RELIC_SECURITY_CONFIG_PATH can't be empty")
			return
		}
		data, err := ioutil.ReadFile(confgFilePath)
		if err == nil {
			err = yaml.Unmarshal(data, &cfg.Security)
			if err != nil {
				cfg.Error = fmt.Errorf("Error while interpreting config file \"%s\" value: %v", confgFilePath, err)
				return
			}
		} else {
			cfg.Error = fmt.Errorf("Error while reading config file \"%s\" , %v", confgFilePath, err)
			return
		}
	}
}

// ConfigSecurityFromEnvironment directs the nrsecurityagent integration to obtain all of its
// configuration information from environment variables:
//
//	NEW_RELIC_SECURITY_ENABLED					(boolean)
//	NEW_RELIC_SECURITY_VALIDATOR_SERVICE_URL    provides URL for the security validator service
//	NEW_RELIC_SECURITY_MODE						scanning mode: "IAST" for now
//	NEW_RELIC_SECURITY_AGENT_ENABLED			(boolean)
//	NEW_RELIC_SECURITY_DETECTION_RXSS_ENABLED	(boolean)
//	NEW_RELIC_SECURITY_REQUEST_BODY_LIMIT		(integer) set limit on read request body in kb. By default, this is "300"

func ConfigSecurityFromEnvironment() ConfigOption {
	return func(cfg *SecurityConfig) {
		assignBool := func(field *bool, name string) {
			if env := os.Getenv(name); env != "" {
				if b, err := strconv.ParseBool(env); nil != err {
					cfg.Error = fmt.Errorf("invalid %s value: %s", name, env)
				} else {
					*field = b
				}
			}
		}
		assignString := func(field *string, name string) {
			if env := os.Getenv(name); env != "" {
				*field = env
			}
		}

		assignInt := func(field *int, name string) {
			if env := os.Getenv(name); env != "" {
				if i, err := strconv.Atoi(env); nil != err {
					cfg.Error = fmt.Errorf("invalid %s value: %s", name, env)
				} else {
					*field = i
				}
			}
		}

		assignBool(&cfg.Security.Enabled, "NEW_RELIC_SECURITY_ENABLED")
		assignString(&cfg.Security.Validator_service_url, "NEW_RELIC_SECURITY_VALIDATOR_SERVICE_URL")
		assignString(&cfg.Security.Mode, "NEW_RELIC_SECURITY_MODE")
		assignBool(&cfg.Security.Agent.Enabled, "NEW_RELIC_SECURITY_AGENT_ENABLED")
		assignBool(&cfg.Security.Detection.Rxss.Enabled, "NEW_RELIC_SECURITY_DETECTION_RXSS_ENABLED")
		assignInt(&cfg.Security.Request.BodyLimit, "NEW_RELIC_SECURITY_REQUEST_BODY_LIMIT")
	}
}

// ConfigSecurityMode sets the security mode to use. By default, this is "IAST".
func ConfigSecurityMode(mode string) ConfigOption {
	return func(cfg *SecurityConfig) {
		cfg.Security.Mode = mode
	}
}

// ConfigSecurityValidatorServiceEndPointUrl sets the security validator service endpoint.
func ConfigSecurityValidatorServiceEndPointUrl(url string) ConfigOption {
	return func(cfg *SecurityConfig) {
		cfg.Security.Validator_service_url = url
	}
}

// ConfigSecurityDetectionDisableRxss is used to enable or disable RXSS validation.
func ConfigSecurityDetectionDisableRxss(isEnabled bool) ConfigOption {
	return func(cfg *SecurityConfig) {
		cfg.Security.Detection.Rxss.Enabled = isEnabled
	}
}

// ConfigSecurityEnable enables or disables the security integration.
func ConfigSecurityEnable(isEnabled bool) ConfigOption {
	return func(cfg *SecurityConfig) {
		cfg.Security.Enabled = isEnabled
	}
}

// ConfigSecurityRequestBodyLimit set limit on read request body in kb. By default, this is "300"
func ConfigSecurityRequestBodyLimit(bodyLimit int) ConfigOption {
	return func(cfg *SecurityConfig) {
		cfg.Security.Request.BodyLimit = bodyLimit
	}
}
