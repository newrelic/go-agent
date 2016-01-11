package newrelic

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"go.datanerd.us/p/will/newrelic/internal"
	"go.datanerd.us/p/will/newrelic/internal/utilization"
	"go.datanerd.us/p/will/newrelic/version"
)

func NewConfig(appname, license string) Config {
	c := Config{}

	c.AppName = appname
	c.License = license
	c.Collector = "collector.newrelic.com"
	c.Labels = make(map[string]string)
	c.CustomEvents.Enabled = true
	c.TransactionEvents.Enabled = true
	c.HighSecurity = false
	c.UseSSL = true
	c.ErrorCollector.Enabled = true
	c.ErrorCollector.CaptureEvents = true
	c.Utilization.DetectAWS = true
	c.Utilization.DetectDocker = true

	return c
}

type Config struct {
	// AppName determines the application record in your New Relic dashboard
	// into which data will be reported.  Collecting data by app name allows
	// you to run an application on more than one server and have all the
	// data aggregated under the same name.
	//
	// https://docs.newrelic.com/docs/apm/new-relic-apm/installation-configuration/naming-your-application
	AppName string

	// License is your 40 digit hexadecimal New Relic license key.
	//
	// https://docs.newrelic.com/docs/accounts-partnerships/accounts/account-setup/license-key
	License string

	// Development can be used in testing and staging situations to stub out
	// the application.  If this bool is set to true, the agent will not
	// collect information, communicate with the New Relic servers, or spawn
	// goroutines.
	Development bool

	// Labels are key value pairs which can be used to roll up applications
	// into specific categories.
	//
	// https://docs.newrelic.com/docs/apm/new-relic-apm/maintenance/labels-categories-organizing-your-apps-servers
	Labels map[string]string

	// HighSecurity mode is an account level feature.  It must be enabled in
	// the New Relic UI before being used here.  HighSecurity mode will
	// guarantee that certain agent settings can not be made more
	// permissive.
	//
	// https://docs.newrelic.com/docs/accounts-partnerships/accounts/security/high-security
	HighSecurity bool

	// CustomEvents.Enabled controls whether the App.RecordCustomEvent() method
	// will collect custom analytics events. This feature will be disabled
	// if HighSecurity mode is enabled.
	//
	// https://docs.newrelic.com/docs/insights/new-relic-insights/adding-querying-data/inserting-custom-events-new-relic-apm-agents
	CustomEvents struct {
		Enabled bool
	}

	// TransactionEvents.Enabled controls the collection of transaction
	// analytics event data.  Event data allows the New Relic UI to show
	// additional information such as histograms.
	TransactionEvents struct {
		Enabled bool
	}

	ErrorCollector struct {
		Enabled       bool
		CaptureEvents bool
	}

	// HostDisplayName sets a custom display name for your application
	// server in the New Relic UI.  Servers are normally identified by host
	// and port number.  This setting allows you to give your hosts more
	// recognizable names.
	HostDisplayName string

	// UseSSL controls whether http or https is used to send data to New
	// Relic servers.
	UseSSL bool

	// Transport may be provided to customize communication with the New
	// Relic servers.  This may be used to configure a proxy.
	Transport http.RoundTripper

	// Collector controls the endpoint to which your application will report
	// data.  You should not need to alter this value.
	Collector string

	Utilization struct {
		DetectAWS    bool
		DetectDocker bool
	}
}

const (
	licenseLength = 40
	appNameLimit  = 3
)

var (
	licenseLenErr      = fmt.Errorf("license length is not %d", licenseLength)
	highSecuritySSLErr = errors.New("high security requires SSL")
	appNameMissing     = errors.New("AppName required")
	appNameLimitErr    = fmt.Errorf("max of %d rollup application names", appNameLimit)
)

func (c Config) Validate() error {
	if len(c.License) != licenseLength {
		return licenseLenErr
	}
	if c.HighSecurity && !c.UseSSL {
		return highSecuritySSLErr
	}
	if "" == c.AppName {
		return appNameMissing
	}
	if strings.Count(c.AppName, ";") > appNameLimit {
		return appNameLimitErr
	}

	return nil
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

func configConnectJSONInternal(c *Config, pid int, util *utilization.Data, e internal.Environment) ([]byte, error) {
	return json.Marshal([]interface{}{struct {
		Pid             int                  `json:"pid"`
		Language        string               `json:"language"`
		Version         string               `json:"agent_version"`
		Host            string               `json:"host"`
		HostDisplayName string               `json:"display_host,omitempty"`
		Settings        interface{}          `json:"settings"`
		AppName         []string             `json:"app_name"`
		HighSecurity    bool                 `json:"high_security"`
		Labels          labels               `json:"labels,omitempty"`
		Environment     internal.Environment `json:"environment"`
		Identifier      string               `json:"identifier"`
		Util            *utilization.Data    `json:"utilization"`
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

func configConnectJSON(c *Config) ([]byte, error) {
	env := internal.NewEnvironment()
	util := utilization.Gather(utilization.Config{
		DetectAWS:    c.Utilization.DetectAWS,
		DetectDocker: c.Utilization.DetectDocker,
	})
	return configConnectJSONInternal(c, os.Getpid(), util, env)
}
