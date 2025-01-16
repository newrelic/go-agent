# v3/integrations/nrsecurityagent [![GoDoc](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsecurityagent?status.svg)](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsecurityagent)

The New Relic security agent analyzes your application for potentially exploitable vulnerabilities.

**DO NOT** use this integration in your production environment. It is intended only for use in your development and testing phases. Since it will attempt to actually find and exploit vulnerabilities in your code, it may cause data loss or crash the application. Therefore it should only be used with test data in a non-production environment that does not connect to any production services.


## Learn More About IAST

 To learn how to use IAST with the New Relic Go Agent, [check out our documentation](https://docs.newrelic.com/docs/iast/use-iast/).

## Setup Instructions

* Add this integration to your application by importing
```
import "github.com/newrelic/go-agent/v3/integrations/nrsecurityagent"
```
* Then, add code to initialize the integration after your call to `newrelic.NewApplication`:

```
app, err := newrelic.NewApplication( ... )
err := nrsecurityagent.InitSecurityAgent(app,
       	nrsecurityagent.ConfigSecurityMode("IAST"),
        nrsecurityagent.ConfigSecurityValidatorServiceEndPointUrl("wss://csec.nr-data.net"),
        nrsecurityagent.ConfigSecurityEnable(true),
    )
```

You can also configure the `nrsecurityagent` integration using a YAML-formatted configuration file:
```
err := nrsecurityagent.InitSecurityAgent(app,
        nrsecurityagent.ConfigSecurityFromYaml(),
)
```

In this case, you need to put the path to your YAML file in an environment variable:
```
NEW_RELIC_SECURITY_CONFIG_PATH={YOUR_PATH}/myappsecurity.yaml
```

The YAML file should have these contents (adjust as needed for your application):
```
# Determines whether the security data is sent to New Relic or not. When this is disabled and agent.enabled is
# true, the security module will run but data will not be sent. Default is false.
enabled: true

# New Relic Security provides two modes: IAST and RASP
# Default is IAST. Due to the invasive nature of IAST scanning, DO NOT enable this mode in either a
# production environment or an environment where production data is processed.
mode: IAST

# New Relic Security's SaaS connection URL
validator_service_url: wss://csec.nr-data.net

# These are the category of security events that can be detected. Set to false to disable detection of
# individual event types. Default is true for each event type.
# This config is deprecated,
detection:
   rci:
      enabled: true
   rxss:
      enabled: true
   deserialization:
      enabled: true

# Unique test identifier when runnning IAST with CI/CD
iast_test_identifier: ""

# IAST scan controllers to get more control over IAST analysis
scan_controllers:
   # maximum number of replay requests IAST Agent 
   # can fire in a minute. Default is 3600. Minimum is 12 and maximum is 3600
   iast_scan_request_rate_limit: 3600
   # The number of application instances for a specific entity where IAST analysis is performed.
   # Values are 0 or 1, 0 signifies run on all application instances
   scan_instance_count: 0

# The scan_schedule configuration allows to specify when IAST scans should be executed
scan_schedule:
   # The delay field specifies the delay in minutes before the IAST scan starts. 
   # This allows to schedule the scan to start at a later time. In minutes, default is 0 min
   delay: 0
   # The duration field specifies the duration of the IAST scan in minutes. 
   # This determines how long the scan will run. In minutes, default is forever
   duration: 0
   # The schedule field specifies a cron expression that defines when the IAST scan should start.
   schedule: ""
   # Allow continuously sample collection of IAST events regardless of scan schedule. Default is false
   always_sample_traces: false

# The exclude_from_iast_scan configuration allows to specify APIs, parameters, 
# and categories that should not be scanned by Security Agents.
exclude_from_iast_scan:
   # The api field specifies list of APIs using regular expression (regex) patterns that follow the syntax of Perl 5.
   # The regex pattern should provide a complete match for the URL without the endpoint.
   api: []
   # The http_request_parameters configuration allows users to specify headers, query parameters,
   # and body keys that should be excluded from IAST scans.
   http_request_parameters:
      # A list of HTTP header keys. If a request includes any headers with these keys,
      # the corresponding IAST scan will be skipped.
      header: []
      # A list of query parameter keys. The presence of these parameters in the request's query string
      # will lead to skipping the IAST scan.
      query: []
      # A list of keys within the request body. If these keys are found in the body content,
      # the IAST scan will be omitted.
      body: []
   # The iast_detection_category configuration allows to specify which categories
   # of vulnerabilities should not be detected by Security Agents.
   iast_detection_category:
      insecure_settings: false
      invalid_file_access: false
      sql_injection: false
      nosql_injection: false
      ldap_injection: false
      javascript_injection: false
      command_injection: false
      xpath_injection: false
      ssrf: false
      rxss: false
```

* Based on additional packages imported by the user application, add suitable instrumentation package imports. 
  For more information, see https://github.com/newrelic/csec-go-agent#instrumentation-packages

**Note**: To completely disable security, set `NEW_RELIC_SECURITY_AGENT_ENABLED` env to false. (Otherwise, there are some security hooks that will already be in place before any of the other configuration settings can be taken into account. This environment variable setting will prevent that from happening.)

## Instrument security-sensitive areas in your application
If you are using the `nrgin`, `nrgrpc`, `nrmicro`, and/or `nrmongo` integrations, they now contain code to support security analysis of the data they handle.

Additionally, the agent will inject vulnerability scanning to instrumented functions wherever possible, including datastore segments, SQL operations, and transactions.

If you are opening an HTTP protocol endpoint, place the `newrelic.WrapListen` function around the endpoint name to enable vulnerability scanning against that endpoint. For example,
```
http.ListenAndServe(newrelic.WrapListen(":8000"), nil)
```

## Start your application in your test environment
Generate traffic against your application for the IAST agent to detect vulnerabilities. Once vulnerabilities are detected they will be reported in the vulnerabilities list.

For more information, see
[godocs](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsecurityagent).
