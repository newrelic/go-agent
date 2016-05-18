package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Config contains settings affecting Application and Transaction behavior.
// NewConfig should be used to create a Config with proper defaults.
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

	// CustomInsightsEvents.Enabled controls whether the App.RecordCustomEvent() method
	// will collect custom analytics events. This feature will be disabled
	// if HighSecurity mode is enabled.
	//
	// https://docs.newrelic.com/docs/insights/new-relic-insights/adding-querying-data/inserting-custom-events-new-relic-apm-agents
	CustomInsightsEvents struct {
		Enabled bool
	}

	// TransactionEvents contains settings affecting transaction analytics
	// events.  This data allows the New Relic UI to show additional
	// information such as histograms.
	TransactionEvents struct {
		// Enabled controls whether transaction analytics events are
		// captured.
		Enabled bool
	}

	// ErrorCollector contains settings which control the capture of errors.
	ErrorCollector struct {
		// Enabled controls whether errors are captured.  This setting
		// affects both traced errors and error analytics events.
		Enabled bool
		// CaptureEvents controls whether error anlytics events are
		// captured.  This data is used on the error analytics page.
		CaptureEvents bool
		// IgnoreStatusCodes controls which http response codes are
		// automatically turned into errors.  By default, response codes
		// greater than or equal to 400, with the exception of 404, are
		// turned into errors.
		IgnoreStatusCodes []int
	}

	// HostDisplayName sets a custom display name for your application
	// server in the New Relic UI.  Servers are normally identified by host
	// and port number.  This setting allows you to give your hosts more
	// recognizable names.
	HostDisplayName string

	// UseTLS controls whether http or https is used to send data to New
	// Relic servers.
	UseTLS bool

	// Transport may be provided to customize communication with the New
	// Relic servers.  This may be used to configure a proxy.
	Transport http.RoundTripper

	// Utilization contains settings controlling system information
	// detection and gathering.
	Utilization struct {
		// DetectAWS controls whether the Application will attempt to
		// detect whether the system is running on AWS.  This will make
		// a small number of local network calls at Application
		// creation.
		DetectAWS bool
		// DetectDocker controls whether the Application will attempt to
		// detect Docker by parsing /proc/self/cgroup.
		DetectDocker bool
	}
}

func NewConfig(appname, license string) Config {
	c := Config{}

	c.AppName = appname
	c.License = license
	c.Labels = make(map[string]string)
	c.CustomInsightsEvents.Enabled = true
	c.TransactionEvents.Enabled = true
	c.HighSecurity = false
	c.UseTLS = true
	c.ErrorCollector.Enabled = true
	c.ErrorCollector.CaptureEvents = true
	c.ErrorCollector.IgnoreStatusCodes = []int{
		http.StatusNotFound, // 404
	}
	c.Utilization.DetectAWS = true
	c.Utilization.DetectDocker = true

	return c
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
	if c.HighSecurity && !c.UseTLS {
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
