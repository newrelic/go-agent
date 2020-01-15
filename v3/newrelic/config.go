package newrelic

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/newrelic/go-agent/v3/internal"
)

// Config contains Application and Transaction behavior settings.
type Config struct {
	// AppName is used by New Relic to link data across servers.
	//
	// https://docs.newrelic.com/docs/apm/new-relic-apm/installation-configuration/naming-your-application
	AppName string

	// License is your New Relic license key.
	//
	// https://docs.newrelic.com/docs/accounts/install-new-relic/account-setup/license-key
	License string

	// Logger controls Go Agent logging.
	//
	// See https://github.com/newrelic/go-agent/blob/master/GUIDE.md#logging
	// for more examples and logging integrations.
	Logger Logger

	// Enabled controls whether the agent will communicate with the New Relic
	// servers and spawn goroutines.  Setting this to be false is useful in
	// testing and staging situations.
	Enabled bool

	// Labels are key value pairs used to roll up applications into specific
	// categories.
	//
	// https://docs.newrelic.com/docs/using-new-relic/user-interface-functions/organize-your-data/labels-categories-organize-apps-monitors
	Labels map[string]string

	// HighSecurity guarantees that certain agent settings can not be made
	// more permissive.  This setting must match the corresponding account
	// setting in the New Relic UI.
	//
	// https://docs.newrelic.com/docs/agents/manage-apm-agents/configuration/high-security-mode
	HighSecurity bool

	// SecurityPoliciesToken enables security policies if set to a non-empty
	// string.  Only set this if security policies have been enabled on your
	// account.  This cannot be used in conjunction with HighSecurity.
	//
	// https://docs.newrelic.com/docs/agents/manage-apm-agents/configuration/enable-configurable-security-policies
	SecurityPoliciesToken string

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
		// MaxSamplesStored allows you to limit the number of Transaction
		// Events stored/reported in a given 60-second period
		MaxSamplesStored int
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
		// greater than or equal to 400 or less than 100 -- with the exception
		// of 0, 5, and 404 -- are turned into errors.
		IgnoreStatusCodes []int
		// Attributes controls the attributes included with errors.
		Attributes AttributeDestinationConfig
		// RecordPanics controls whether or not a deferred
		// Transaction.End will attempt to recover panics, record them
		// as errors, and then re-panic them.  By default, this is
		// set to false.
		RecordPanics bool
	}

	// TransactionTracer controls the capture of transaction traces.
	TransactionTracer struct {
		// Enabled controls whether transaction traces are captured.
		Enabled bool
		// Threshold controls whether a transaction trace will be
		// considered for capture.  Of the traces exceeding the
		// threshold, the slowest trace every minute is captured.
		Threshold struct {
			// If IsApdexFailing is true then the trace threshold is
			// four times the apdex threshold.
			IsApdexFailing bool
			// If IsApdexFailing is false then this field is the
			// threshold, otherwise it is ignored.
			Duration time.Duration
		}
		// Attributes controls the attributes included with transaction
		// traces.
		Attributes AttributeDestinationConfig
		// Segments contains fields which control the behavior of
		// transaction trace segments.
		Segments struct {
			// StackTraceThreshold is the threshold at which
			// segments will be given a stack trace in the
			// transaction trace.  Lowering this setting will
			// increase overhead.
			StackTraceThreshold time.Duration
			// Threshold is the threshold at which segments will be
			// added to the trace.  Lowering this setting may
			// increase overhead.  Decrease this duration if your
			// transaction traces are missing segments.
			Threshold time.Duration
			// Attributes controls the attributes included with each
			// trace segment.
			Attributes AttributeDestinationConfig
		}
	}

	// BrowserMonitoring contains settings which control the behavior of
	// Transaction.BrowserTimingHeader.
	BrowserMonitoring struct {
		// Enabled controls whether or not the Browser monitoring feature is
		// enabled.
		Enabled bool
		// Attributes controls the attributes included with Browser monitoring.
		// BrowserMonitoring.Attributes.Enabled is false by default, to include
		// attributes in the Browser timing Javascript:
		//
		//	cfg.BrowserMonitoring.Attributes.Enabled = true
		Attributes AttributeDestinationConfig
	}

	// HostDisplayName gives this server a recognizable name in the New
	// Relic UI.  This is an optional setting.
	HostDisplayName string

	// Transport customizes communication with the New Relic servers.  This may
	// be used to configure a proxy.
	Transport http.RoundTripper

	// Utilization controls the detection and gathering of system
	// information.
	Utilization struct {
		// DetectAWS controls whether the Application attempts to detect
		// AWS.
		DetectAWS bool
		// DetectAzure controls whether the Application attempts to detect
		// Azure.
		DetectAzure bool
		// DetectPCF controls whether the Application attempts to detect
		// PCF.
		DetectPCF bool
		// DetectGCP controls whether the Application attempts to detect
		// GCP.
		DetectGCP bool
		// DetectDocker controls whether the Application attempts to
		// detect Docker.
		DetectDocker bool
		// DetectKubernetes controls whether the Application attempts to
		// detect Kubernetes.
		DetectKubernetes bool

		// These settings provide system information when custom values
		// are required.
		LogicalProcessors int
		TotalRAMMIB       int
		BillingHostname   string
	}

	// Heroku controls the behavior of Heroku specific features.
	Heroku struct {
		// UseDynoNames controls if Heroku dyno names are reported as the
		// hostname.  Default is true.
		UseDynoNames bool
		// DynoNamePrefixesToShorten allows you to shorten and combine some
		// Heroku dyno names into a single value.  Ordinarily the agent reports
		// dyno names with a trailing dot and process ID (for example,
		// worker.3). You can remove this trailing data by specifying the
		// prefixes you want to report without trailing data (for example,
		// worker.*).  Defaults to shortening "scheduler" and "run" dyno names.
		DynoNamePrefixesToShorten []string
	}

	// CrossApplicationTracer controls behavior relating to cross application
	// tracing (CAT).  In the case where CrossApplicationTracer and
	// DistributedTracer are both enabled, DistributedTracer takes precedence.
	//
	// https://docs.newrelic.com/docs/apm/transactions/cross-application-traces/introduction-cross-application-traces
	CrossApplicationTracer struct {
		Enabled bool
	}

	// DistributedTracer controls behavior relating to Distributed Tracing.  In
	// the case where CrossApplicationTracer and DistributedTracer are both
	// enabled, DistributedTracer takes precedence.
	//
	// https://docs.newrelic.com/docs/apm/distributed-tracing/getting-started/introduction-distributed-tracing
	DistributedTracer struct {
		Enabled bool
		// ExcludeNewRelicHeader allows you to choose whether to insert the New
		// Relic Distributed Tracing header on outbound requests, which by
		// default is emitted along with the W3C trace context headers.  Set
		// this value to true if you do not want to include the New Relic
		// distributed tracing header in your outbound requests.
		//
		// Disabling the New Relic header here does not prevent the agent from
		// accepting *inbound* New Relic headers.
		ExcludeNewRelicHeader bool
	}

	// SpanEvents controls behavior relating to Span Events.  Span Events
	// require that DistributedTracer is enabled.
	SpanEvents struct {
		Enabled bool
		// Attributes controls the attributes included on Spans.
		Attributes AttributeDestinationConfig
	}

	// DatastoreTracer controls behavior relating to datastore segments.
	DatastoreTracer struct {
		// InstanceReporting controls whether the host and port are collected
		// for datastore segments.
		InstanceReporting struct {
			Enabled bool
		}
		// DatabaseNameReporting controls whether the database name is
		// collected for datastore segments.
		DatabaseNameReporting struct {
			Enabled bool
		}
		QueryParameters struct {
			Enabled bool
		}
		// SlowQuery controls the capture of slow query traces.  Slow
		// query traces show you instances of your slowest datastore
		// segments.
		SlowQuery struct {
			Enabled   bool
			Threshold time.Duration
		}
	}

	// Attributes controls which attributes are enabled and disabled globally.
	// This setting affects all attribute destinations: Transaction Events,
	// Error Events, Transaction Traces and segments, Traced Errors, Span
	// Events, and Browser timing header.
	Attributes AttributeDestinationConfig

	// RuntimeSampler controls the collection of runtime statistics like
	// CPU/Memory usage, goroutine count, and GC pauses.
	RuntimeSampler struct {
		// Enabled controls whether runtime statistics are captured.
		Enabled bool
	}

	// ServerlessMode contains fields which control behavior when running in
	// AWS Lambda.
	//
	// https://docs.newrelic.com/docs/serverless-function-monitoring/aws-lambda-monitoring/get-started/introduction-new-relic-monitoring-aws-lambda
	ServerlessMode struct {
		// Enabling ServerlessMode will print each transaction's data to
		// stdout.  No agent goroutines will be spawned in serverless mode, and
		// no data will be sent directly to the New Relic backend.
		// nrlambda.NewConfig sets Enabled to true.
		Enabled bool
		// ApdexThreshold sets the Apdex threshold when in ServerlessMode.  The
		// default is 500 milliseconds.  nrlambda.NewConfig populates this
		// field using the NEW_RELIC_APDEX_T environment variable.
		//
		// https://docs.newrelic.com/docs/apm/new-relic-apm/apdex/apdex-measure-user-satisfaction
		ApdexThreshold time.Duration
		// AccountID, TrustedAccountKey, and PrimaryAppID are used for
		// distributed tracing in ServerlessMode.  AccountID and
		// TrustedAccountKey must be populated for distributed tracing to be
		// enabled. nrlambda.NewConfig populates these fields using the
		// NEW_RELIC_ACCOUNT_ID, NEW_RELIC_TRUSTED_ACCOUNT_KEY, and
		// NEW_RELIC_PRIMARY_APPLICATION_ID environment variables.
		AccountID         string
		TrustedAccountKey string
		PrimaryAppID      string
	}

	// Host can be used to override the New Relic endpoint.
	Host string

	// Error may be populated by the ConfigOptions provided to NewApplication
	// to indicate that setup has failed.  NewApplication will return this
	// error if it is set.
	Error error
}

// AttributeDestinationConfig controls the attributes sent to each destination.
// For more information, see:
// https://docs.newrelic.com/docs/agents/manage-apm-agents/agent-data/agent-attributes
type AttributeDestinationConfig struct {
	// Enabled controls whether or not this destination will get any
	// attributes at all.  For example, to prevent any attributes from being
	// added to errors, set:
	//
	//	cfg.ErrorCollector.Attributes.Enabled = false
	//
	Enabled bool
	Include []string
	// Exclude allows you to prevent the capture of certain attributes.  For
	// example, to prevent the capture of the request URL attribute
	// "request.uri", set:
	//
	//	cfg.Attributes.Exclude = append(cfg.Attributes.Exclude, newrelic.AttributeRequestURI)
	//
	// The '*' character acts as a wildcard.  For example, to prevent the
	// capture of all request related attributes, set:
	//
	//	cfg.Attributes.Exclude = append(cfg.Attributes.Exclude, "request.*")
	//
	Exclude []string
}

// ConfigOption configures the Config when provided to NewApplication.
type ConfigOption func(*Config)

// ConfigEnabled sets the whether or not the agent is enabled.
func ConfigEnabled(enabled bool) ConfigOption {
	return func(cfg *Config) { cfg.Enabled = enabled }
}

// ConfigAppName sets the application name.
func ConfigAppName(appName string) ConfigOption {
	return func(cfg *Config) { cfg.AppName = appName }
}

// ConfigLicense sets the license.
func ConfigLicense(license string) ConfigOption {
	return func(cfg *Config) { cfg.License = license }
}

// ConfigDistributedTracerEnabled populates the Config's
// DistributedTracer.Enabled setting.
func ConfigDistributedTracerEnabled(enabled bool) ConfigOption {
	return func(cfg *Config) { cfg.DistributedTracer.Enabled = enabled }
}

// ConfigLogger populates the Config's Logger.
func ConfigLogger(l Logger) ConfigOption {
	return func(cfg *Config) { cfg.Logger = l }
}

// ConfigInfoLogger populates the config with basic Logger at info level.
func ConfigInfoLogger(w io.Writer) ConfigOption {
	return ConfigLogger(NewLogger(w))
}

// ConfigDebugLogger populates the config with a Logger at debug level.
func ConfigDebugLogger(w io.Writer) ConfigOption {
	return ConfigLogger(NewDebugLogger(w))
}

// ConfigFromEnvironment populates the config based on environment variables:
//
//  NEW_RELIC_APP_NAME                       sets AppName
//  NEW_RELIC_LICENSE_KEY                    sets License
//  NEW_RELIC_DISTRIBUTED_TRACING_ENABLED    sets DistributedTracer.Enabled using strconv.ParseBool
//  NEW_RELIC_ENABLED                        sets Enabled using strconv.ParseBool
//  NEW_RELIC_HIGH_SECURITY                  sets HighSecurity using strconv.ParseBool
//  NEW_RELIC_SECURITY_POLICIES_TOKEN        sets SecurityPoliciesToken
//  NEW_RELIC_HOST                           sets Host
//  NEW_RELIC_PROCESS_HOST_DISPLAY_NAME      sets HostDisplayName
//  NEW_RELIC_UTILIZATION_BILLING_HOSTNAME   sets Utilization.BillingHostname
//  NEW_RELIC_UTILIZATION_LOGICAL_PROCESSORS sets Utilization.LogicalProcessors using strconv.Atoi
//  NEW_RELIC_UTILIZATION_TOTAL_RAM_MIB      sets Utilization.TotalRAMMIB using strconv.Atoi
//  NEW_RELIC_LABELS                         sets Labels using a semi-colon delimited string of colon-separated pairs, eg. "Server:One;DataCenter:Primary"
//  NEW_RELIC_ATTRIBUTES_EXCLUDE             sets Attributes.Exclude using a comma-separated list, eg. "request.headers.host,request.method"
//  NEW_RELIC_ATTRIBUTES_INCLUDE             sets Attributes.Include using a comma-separated list
//  NEW_RELIC_LOG                            sets Logger to log to either "stdout" or "stderr" (filenames are not supported)
//  NEW_RELIC_LOG_LEVEL                      controls the NEW_RELIC_LOG level, must be "debug" for debug, or empty for info
//
// This function is strict and will assign Config.Error if any of the
// environment variables cannot be parsed.
func ConfigFromEnvironment() ConfigOption {
	return configFromEnvironment(os.Getenv)
}

func configFromEnvironment(getenv func(string) string) ConfigOption {
	return func(cfg *Config) {
		// Because fields could have been assigned in a previous
		// ConfigOption, we only want to assign fields using environment
		// variables that have been populated.  This is especially
		// relevant for the string case where no processing occurs.
		assignBool := func(field *bool, name string) {
			if env := getenv(name); env != "" {
				if b, err := strconv.ParseBool(env); nil != err {
					cfg.Error = fmt.Errorf("invalid %s value: %s", name, env)
				} else {
					*field = b
				}
			}
		}
		assignInt := func(field *int, name string) {
			if env := getenv(name); env != "" {
				if i, err := strconv.Atoi(env); nil != err {
					cfg.Error = fmt.Errorf("invalid %s value: %s", name, env)
				} else {
					*field = i
				}
			}
		}
		assignString := func(field *string, name string) {
			if env := getenv(name); env != "" {
				*field = env
			}
		}

		assignString(&cfg.AppName, "NEW_RELIC_APP_NAME")
		assignString(&cfg.License, "NEW_RELIC_LICENSE_KEY")
		assignBool(&cfg.DistributedTracer.Enabled, "NEW_RELIC_DISTRIBUTED_TRACING_ENABLED")
		assignBool(&cfg.Enabled, "NEW_RELIC_ENABLED")
		assignBool(&cfg.HighSecurity, "NEW_RELIC_HIGH_SECURITY")
		assignString(&cfg.SecurityPoliciesToken, "NEW_RELIC_SECURITY_POLICIES_TOKEN")
		assignString(&cfg.Host, "NEW_RELIC_HOST")
		assignString(&cfg.HostDisplayName, "NEW_RELIC_PROCESS_HOST_DISPLAY_NAME")
		assignString(&cfg.Utilization.BillingHostname, "NEW_RELIC_UTILIZATION_BILLING_HOSTNAME")
		assignInt(&cfg.Utilization.LogicalProcessors, "NEW_RELIC_UTILIZATION_LOGICAL_PROCESSORS")
		assignInt(&cfg.Utilization.TotalRAMMIB, "NEW_RELIC_UTILIZATION_TOTAL_RAM_MIB")

		if env := getenv("NEW_RELIC_LABELS"); env != "" {
			if labels := getLabels(getenv("NEW_RELIC_LABELS")); len(labels) > 0 {
				cfg.Labels = labels
			} else {
				cfg.Error = fmt.Errorf("invalid NEW_RELIC_LABELS value: %s", env)
			}
		}

		if env := getenv("NEW_RELIC_ATTRIBUTES_INCLUDE"); env != "" {
			cfg.Attributes.Include = strings.Split(env, ",")
		}
		if env := getenv("NEW_RELIC_ATTRIBUTES_EXCLUDE"); env != "" {
			cfg.Attributes.Exclude = strings.Split(env, ",")
		}

		if env := getenv("NEW_RELIC_LOG"); env != "" {
			if dest := getLogDest(env); dest != nil {
				if isDebugEnv(getenv("NEW_RELIC_LOG_LEVEL")) {
					cfg.Logger = NewDebugLogger(dest)
				} else {
					cfg.Logger = NewLogger(dest)
				}
			} else {
				cfg.Error = fmt.Errorf("invalid NEW_RELIC_LOG value %s", env)
			}
		}
	}
}

func getLogDest(env string) io.Writer {
	switch env {
	case "stdout", "Stdout", "STDOUT":
		return os.Stdout
	case "stderr", "Stderr", "STDERR":
		return os.Stderr
	default:
		return nil
	}
}

func isDebugEnv(env string) bool {
	switch env {
	case "debug", "Debug", "DEBUG", "d", "D":
		return true
	default:
		return false
	}
}

// getLabels reads Labels from the env string, expressed as a semi-colon
// delimited string of colon-separated pairs (for example, "Server:One;Data
// Center:Primary").  Label keys and values must be 255 characters or less in
// length.  No more than 64 Labels can be set.
func getLabels(env string) map[string]string {
	out := make(map[string]string)
	env = strings.Trim(env, ";\t\n\v\f\r ")
	for _, entry := range strings.Split(env, ";") {
		if entry == "" {
			return nil
		}
		split := strings.Split(entry, ":")
		if len(split) != 2 {
			return nil
		}
		left := strings.TrimSpace(split[0])
		right := strings.TrimSpace(split[1])
		if left == "" || right == "" {
			return nil
		}
		if utf8.RuneCountInString(left) > 255 {
			runes := []rune(left)
			left = string(runes[:255])
		}
		if utf8.RuneCountInString(right) > 255 {
			runes := []rune(right)
			right = string(runes[:255])
		}
		out[left] = right
		if len(out) >= 64 {
			return out
		}
	}
	return out
}

// defaultConfig creates a Config populated with default settings.
func defaultConfig() Config {
	c := Config{}

	c.Enabled = true
	c.Labels = make(map[string]string)
	c.CustomInsightsEvents.Enabled = true
	c.TransactionEvents.Enabled = true
	c.TransactionEvents.Attributes.Enabled = true
	c.TransactionEvents.MaxSamplesStored = internal.MaxTxnEvents
	c.HighSecurity = false
	c.ErrorCollector.Enabled = true
	c.ErrorCollector.CaptureEvents = true
	c.ErrorCollector.IgnoreStatusCodes = []int{
		// https://github.com/grpc/grpc/blob/master/doc/statuscodes.md
		0,                   // gRPC OK
		5,                   // gRPC NOT_FOUND
		http.StatusNotFound, // 404
	}
	c.ErrorCollector.Attributes.Enabled = true
	c.Utilization.DetectAWS = true
	c.Utilization.DetectAzure = true
	c.Utilization.DetectPCF = true
	c.Utilization.DetectGCP = true
	c.Utilization.DetectDocker = true
	c.Utilization.DetectKubernetes = true
	c.Attributes.Enabled = true
	c.RuntimeSampler.Enabled = true

	c.TransactionTracer.Enabled = true
	c.TransactionTracer.Threshold.IsApdexFailing = true
	c.TransactionTracer.Threshold.Duration = 500 * time.Millisecond
	c.TransactionTracer.Segments.Threshold = 2 * time.Millisecond
	c.TransactionTracer.Segments.StackTraceThreshold = 500 * time.Millisecond
	c.TransactionTracer.Attributes.Enabled = true
	c.TransactionTracer.Segments.Attributes.Enabled = true

	c.BrowserMonitoring.Enabled = true
	// browser monitoring attributes are disabled by default
	c.BrowserMonitoring.Attributes.Enabled = false

	c.CrossApplicationTracer.Enabled = true
	c.DistributedTracer.Enabled = false
	c.SpanEvents.Enabled = true
	c.SpanEvents.Attributes.Enabled = true

	c.DatastoreTracer.InstanceReporting.Enabled = true
	c.DatastoreTracer.DatabaseNameReporting.Enabled = true
	c.DatastoreTracer.QueryParameters.Enabled = true
	c.DatastoreTracer.SlowQuery.Enabled = true
	c.DatastoreTracer.SlowQuery.Threshold = 10 * time.Millisecond

	c.ServerlessMode.ApdexThreshold = 500 * time.Millisecond
	c.ServerlessMode.Enabled = false

	c.Heroku.UseDynoNames = true
	c.Heroku.DynoNamePrefixesToShorten = []string{"scheduler", "run"}

	return c
}

const (
	licenseLength = 40
	appNameLimit  = 3
)

// The following errors will be returned if your Config fails to validate.
var (
	errLicenseLen                       = fmt.Errorf("license length is not %d", licenseLength)
	errAppNameMissing                   = errors.New("string AppName required")
	errAppNameLimit                     = fmt.Errorf("max of %d rollup application names", appNameLimit)
	errHighSecurityWithSecurityPolicies = errors.New("SecurityPoliciesToken and HighSecurity are incompatible; please ensure HighSecurity is set to false if SecurityPoliciesToken is a non-empty string and a security policy has been set for your account")
)

// validate checks the config for improper fields.  If the config is invalid,
// newrelic.NewApplication returns an error.
func (c Config) validate() error {
	if c.Enabled && !c.ServerlessMode.Enabled {
		if len(c.License) != licenseLength {
			return errLicenseLen
		}
	} else {
		// The License may be empty when the agent is not enabled.
		if len(c.License) != licenseLength && len(c.License) != 0 {
			return errLicenseLen
		}
	}
	if "" == c.AppName && c.Enabled && !c.ServerlessMode.Enabled {
		return errAppNameMissing
	}
	if c.HighSecurity && "" != c.SecurityPoliciesToken {
		return errHighSecurityWithSecurityPolicies
	}
	if strings.Count(c.AppName, ";") >= appNameLimit {
		return errAppNameLimit
	}
	return nil
}

// maxTxnEvents returns the configured maximum number of Transaction Events if it has been configured
// and is less than the default maximum; otherwise it returns the default max.
func (c Config) maxTxnEvents() int {
	configured := c.TransactionEvents.MaxSamplesStored
	if configured < 0 || configured > internal.MaxTxnEvents {
		return internal.MaxTxnEvents
	}
	return configured
}
