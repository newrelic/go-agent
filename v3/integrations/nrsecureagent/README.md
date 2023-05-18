# v3/integrations/nrsecureagent [![GoDoc](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsecureagent?status.svg)](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsecureagent)

This integration allows you to have the New Relic security agent analyze your application for potentially exploitable vulnerabilities.

**DO NOT** use this integration in your production environment. It is intended only for use in your development and testing phases. Since it will attempt to actually find and exploit vulnerabilities in your code, it may cause data loss or crash the application. Therefore it should only be used with test data in a non-production environment that does not connect to any production services.

# Special Note for this Preview Version
**Please read the following carefully before proceeding.**

This is not yet publicly released on github, so you will need to add a `replace` statement to your application's `go.mod` file to point the go toolchain to the location where you downloaded and unpacked the preview version of this integration and the agent. At minimum:
```
replace (
    github.com/newrelic/go-agent/v3/integrations/nrsecureagent => {YOUR_PATH}/go-agent/v3/integrations/nrsecureagent
    github.com/newrelic/go-agent/v3 => {YOUR_PATH}/go-agent/v3
    github.com/newrelic/csec-go-agent => {YOUR_PATH}/csec-go-agent
)
```
Additionally, add `replace` lines for each integration you are using so they are taken from the preview version of the agent and not downloaded from the latest public release.

## Setup Instructions

* Add this integration to your application by importing
```
import "github.com/newrelic/go-agent/v3/integrations/nrsecureagent"
```
* Then, add code to initialize the integration after your call to `newrelic.NewApplication`:

```
app, err := newrelic.NewApplication( ... )
err := nrsecureagent.InitSecurityAgent(app,
       	nrsecureagent.ConfigSecurityMode("IAST"),
        nrsecureagent.ConfigSecurityValidatorServiceEndPointUrl("wss://csec.nr-data.net"),
        nrsecureagent.ConfigSecurityEnable(true),
    )
```

You can also configure the `nrsecureagent` integration using a YAML-formatted configuration file:
```
err := nrsecureagent.InitSecurityAgent(app,
        nrsecureagent.ConfigSecurityFromYaml(),
)
```

In this case, you need to put the path to your YAML file in an environment variable:
```
NEW_RELIC_SECURITY_CONFIG_PATH={YOUR_PATH}/myappsecurity.yaml
```

The YAML file should have these contents (adjust as needed for your application):
```
enabled: true
    mode: IAST
  validator_service_url: wss://csec.nr-data.net
  agent:
    enabled: true
  detection:
    rci:
      enabled: true
    rxss:
      enabled: true
    deserialization:
      enabled: true
```

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
[godocs](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsecureagent).
**(note that link will only be live once this feature is fully released to the public)**
