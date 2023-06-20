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
enabled: true

 # NR security provides two modes IAST and RASP
 # Default is IAST
mode: IAST

 # New Relic’s SaaS connection URLs
validator_service_url: wss://csec.nr-data.net

 # Following category of security events
 # can be disabled from generating.
detection:
  rxss:
    enabled: true
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
