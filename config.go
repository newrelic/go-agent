package newrelic

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Config contains Application and Transaction behavior settings.
// Use NewConfig to create a Config with proper defaults.
type Config struct {
	// AppName is used by New Relic to link data across servers.
	//
	// https://docs.newrelic.com/docs/apm/new-relic-apm/installation-configuration/naming-your-application
	AppName string

	// License is your New Relic license key.
	//
	// https://docs.newrelic.com/docs/accounts-partnerships/accounts/account-setup/license-key
	License string

	// Logger controls go-agent logging.  See log.go.
	Logger Logger

	// BetaToken exists to ensure that you have signed the beta agreement
	// available here:
	//
	//   http://goo.gl/forms/Rcv1b10Qvt1ENLlr1
	//
	// This field will be removed once the Beta is complete.
	BetaToken string

	// Enabled determines whether the agent will communicate with the New
	// Relic servers and spawn goroutines.  Setting this to be false can be
	// useful in testing and staging situations.
	Enabled bool

	// Labels are key value pairs used to roll up applications into specific
	// categories.
	//
	// https://docs.newrelic.com/docs/apm/new-relic-apm/maintenance/labels-categories-organizing-your-apps-servers
	Labels map[string]string

	// HighSecurity guarantees that certain agent settings can not be made
	// more permissive.  This setting must match the corresponding account
	// setting in the New Relic UI.
	//
	// https://docs.newrelic.com/docs/accounts-partnerships/accounts/security/high-security
	HighSecurity bool

	// CustomInsightsEvents controls the behavior of
	// Application.RecordCustomEvent.
	//
	// https://docs.newrelic.com/docs/insights/new-relic-insights/adding-querying-data/inserting-custom-events-new-relic-apm-agents
	CustomInsightsEvents struct {
		// Enabled controls whether RecordCustomEvent will collect
		// custom analytics events.  High security mode overrides this
		// setting.
		Enabled bool
	}

	// TransactionEvents controls the behavior of transaction analytics
	// events.
	TransactionEvents struct {
		// Enabled controls whether transaction events are captured.
		Enabled bool
		// Attributes controls the attributes included with transaction
		// events.
		Attributes AttributeDestinationConfig
	}

	// ErrorCollector controls the capture of errors.
	ErrorCollector struct {
		// Enabled controls whether errors are captured.  This setting
		// affects both traced errors and error analytics events.
		Enabled bool
		// CaptureEvents controls whether error analytics events are
		// captured.
		CaptureEvents bool
		// IgnoreStatusCodes controls which http response codes are
		// automatically turned into errors.  By default, response codes
		// greater than or equal to 400, with the exception of 404, are
		// turned into errors.
		IgnoreStatusCodes []int
		// Attributes controls the attributes included with errors.
		Attributes AttributeDestinationConfig
	}

	// HostDisplayName gives this server a recognizable name in the New
	// Relic UI.  This is an optional setting.
	HostDisplayName string

	// UseTLS controls whether http or https is used to send data to New
	// Relic servers.
	UseTLS bool

	// Transport customizes http.Client communication with New Relic
	// servers.  This may be used to configure a proxy.
	Transport http.RoundTripper

	// Utilization controls the detection and gathering of system
	// information.
	Utilization struct {
		// DetectAWS controls whether the Application attempts to detect
		// AWS.
		DetectAWS bool
		// DetectDocker controls whether the Application attempts to
		// detect Docker.
		DetectDocker bool
	}

	// Attributes controls the attributes included with errors and
	// transaction events.
	Attributes AttributeDestinationConfig

	// RuntimeSampler controls the collection of runtime statistics like
	// CPU/Memory usage, goroutine count, and GC pauses.
	RuntimeSampler struct {
		// Enabled controls whether runtime statistics are captured.
		Enabled bool
	}
}

// AttributeDestinationConfig controls the attributes included with errors and
// transaction events.
type AttributeDestinationConfig struct {
	Enabled bool
	Include []string
	Exclude []string
}

// NewConfig creates an Config populated with the given appname, license,
// and expected default values.
func NewConfig(appname, license string) Config {
	c := Config{}

	c.AppName = appname
	c.License = license
	c.Enabled = true
	c.Labels = make(map[string]string)
	c.CustomInsightsEvents.Enabled = true
	c.TransactionEvents.Enabled = true
	c.TransactionEvents.Attributes.Enabled = true
	c.HighSecurity = false
	c.UseTLS = true
	c.ErrorCollector.Enabled = true
	c.ErrorCollector.CaptureEvents = true
	c.ErrorCollector.IgnoreStatusCodes = []int{
		http.StatusNotFound, // 404
	}
	c.ErrorCollector.Attributes.Enabled = true
	c.Utilization.DetectAWS = true
	c.Utilization.DetectDocker = true
	c.Attributes.Enabled = true
	c.RuntimeSampler.Enabled = true

	return c
}

const (
	licenseLength = 40
	appNameLimit  = 3
)

// The following errors will be returned if your Config fails to validate.
var (
	errLicenseLen      = fmt.Errorf("license length is not %d", licenseLength)
	errHighSecurityTLS = errors.New("high security requires TLS")
	errAppNameMissing  = errors.New("AppName required")
	errAppNameLimit    = fmt.Errorf("max of %d rollup application names", appNameLimit)
)

// Validate checks the config for improper fields.  If the config is invalid,
// newrelic.NewApplication returns an error.
func (c Config) Validate() error {
	if c.Enabled {
		if len(c.License) != licenseLength {
			return errLicenseLen
		}
	} else {
		// The License may be empty when the agent is not enabled.
		if len(c.License) != licenseLength && len(c.License) != 0 {
			return errLicenseLen
		}
	}
	if c.HighSecurity && !c.UseTLS {
		return errHighSecurityTLS
	}
	if "" == c.AppName {
		return errAppNameMissing
	}
	if strings.Count(c.AppName, ";") >= appNameLimit {
		return errAppNameLimit
	}
	return nil
}
