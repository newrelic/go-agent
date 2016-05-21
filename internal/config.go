package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/newrelic/go-sdk/api"
	"github.com/newrelic/go-sdk/internal/utilization"
	"github.com/newrelic/go-sdk/version"
)

func copyConfigReferenceFields(cfg api.Config) api.Config {
	cp := cfg
	if nil != cfg.Labels {
		cp.Labels = make(map[string]string, len(cfg.Labels))
		for key, val := range cfg.Labels {
			cp.Labels[key] = val
		}
	}
	if nil != cfg.ErrorCollector.IgnoreStatusCodes {
		ignored := make([]int, len(cfg.ErrorCollector.IgnoreStatusCodes))
		copy(ignored, cfg.ErrorCollector.IgnoreStatusCodes)
		cp.ErrorCollector.IgnoreStatusCodes = ignored
	}
	return cp
}

type labels map[string]string

func (l labels) MarshalJSON() ([]byte, error) {
	ls := make([]struct {
		Key   string `json:"label_type"`
		Value string `json:"label_value"`
	}, len(l))

	i := 0
	for key, val := range l {
		ls[i].Key = key
		ls[i].Value = val
		i++
	}

	return json.Marshal(ls)
}

const (
	agentLanguage = "go"
)

func transportSetting(t http.RoundTripper) interface{} {
	if nil == t {
		return nil
	}
	return fmt.Sprintf("%T", t)
}

const (
	// https://source.datanerd.us/agents/agent-specs/blob/master/Custom-Host-Names.md
	hostByteLimit = 255
)

type settings api.Config

func (s *settings) MarshalJSON() ([]byte, error) {
	c := (*api.Config)(s)
	js, err := json.Marshal(c)
	if nil != err {
		return nil, err
	}
	fields := make(map[string]interface{})
	err = json.Unmarshal(js, &fields)
	if nil != err {
		return nil, err
	}
	// The License field is not simply ignored by adding the `json:"-"` tag
	// to it since we want to allow consumers to populate Config from JSON.
	delete(fields, `License`)
	fields[`Transport`] = transportSetting(c.Transport)
	return json.Marshal(fields)
}

func configConnectJSONInternal(c *api.Config, pid int, util *utilization.Data, e environment, version string) ([]byte, error) {
	return json.Marshal([]interface{}{struct {
		Pid             int               `json:"pid"`
		Language        string            `json:"language"`
		Version         string            `json:"agent_version"`
		Host            string            `json:"host"`
		HostDisplayName string            `json:"display_host,omitempty"`
		Settings        interface{}       `json:"settings"`
		AppName         []string          `json:"app_name"`
		HighSecurity    bool              `json:"high_security"`
		Labels          labels            `json:"labels,omitempty"`
		Environment     environment       `json:"environment"`
		Identifier      string            `json:"identifier"`
		Util            *utilization.Data `json:"utilization"`
	}{
		Pid:             pid,
		Language:        agentLanguage,
		Version:         version,
		Host:            stringLengthByteLimit(util.Hostname, hostByteLimit),
		HostDisplayName: stringLengthByteLimit(c.HostDisplayName, hostByteLimit),
		Settings:        (*settings)(c),
		AppName:         strings.Split(c.AppName, ";"),
		HighSecurity:    c.HighSecurity,
		Labels:          labels(c.Labels),
		Environment:     e,
		// This identifier field is provided to avoid:
		// https://newrelic.atlassian.net/browse/DSCORE-778
		//
		// This identifier is used by the collector to look up the real
		// agent. If an identifier isn't provided, the collector will
		// create its own based on the first appname, which prevents a
		// single daemon from connecting "a;b" and "a;c" at the same
		// time.
		//
		// Providing the identifier below works around this issue and
		// allows users more flexibility in using application rollups.
		Identifier: c.AppName,
		Util:       util,
	}})
}

func configConnectJSON(c *api.Config) ([]byte, error) {
	env := newEnvironment()
	util := utilization.Gather(utilization.Config{
		DetectAWS:    c.Utilization.DetectAWS,
		DetectDocker: c.Utilization.DetectDocker,
	})
	return configConnectJSONInternal(c, os.Getpid(), util, env, version.Version)
}
