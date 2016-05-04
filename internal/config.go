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

func configConnectJSONInternal(c *api.Config, pid int, util *utilization.Data, e Environment) ([]byte, error) {
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
		Environment     Environment       `json:"environment"`
		Identifier      string            `json:"identifier"`
		Util            *utilization.Data `json:"utilization"`
	}{
		Pid:      pid,
		Language: agentLanguage,
		Version:  version.Version,
		Host:     util.Hostname,
		// QUESTION: Should we limit the length of this field here, or
		// check the length of the value in the Config Validate method?
		HostDisplayName: c.HostDisplayName,
		Settings: struct {
			// QUESTION: Should Labels be flattened and included
			// here?
			HighSecurity bool `json:"high_security"`
			// QUESTION: Should CustomEvents.Enabled be changed to
			// CustomInsightsEvents.Enabled for consistency with
			// other agents?
			CustomEventsEnabled         bool `json:"custom_insights_events.enabled"`
			TransactionEventsEnabled    bool `json:"transaction_events.enabled"`
			ErrorCollectorEnabled       bool `json:"error_collector.enabled"`
			ErrorCollectorCaptureEvents bool `json:"error_collector.capture_events"`
			// QUESTION: Should HostDisplayName be duplication here?
			UseSSL                  bool        `json:"ssl"`
			Transport               interface{} `json:"transport"`
			Collector               string      `json:"collector"`
			UtilizationDetectAWS    bool        `json:"utilization.detect_aws"`
			UtilizationDetectDocker bool        `json:"utilization.detect_docker"`
		}{
			HighSecurity:                c.HighSecurity,
			CustomEventsEnabled:         c.CustomEvents.Enabled,
			TransactionEventsEnabled:    c.TransactionEvents.Enabled,
			ErrorCollectorEnabled:       c.ErrorCollector.Enabled,
			ErrorCollectorCaptureEvents: c.ErrorCollector.CaptureEvents,
			UseSSL:                  c.UseSSL,
			Transport:               transportSetting(c.Transport),
			Collector:               c.Collector,
			UtilizationDetectAWS:    c.Utilization.DetectAWS,
			UtilizationDetectDocker: c.Utilization.DetectDocker,
		},
		AppName:      strings.Split(c.AppName, ";"),
		HighSecurity: c.HighSecurity,
		Labels:       labels(c.Labels),
		Environment:  e,
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
	env := NewEnvironment()
	util := utilization.Gather(utilization.Config{
		DetectAWS:    c.Utilization.DetectAWS,
		DetectDocker: c.Utilization.DetectDocker,
	})
	return configConnectJSONInternal(c, os.Getpid(), util, env)
}
