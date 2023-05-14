# v3/integrations/nrsecureagent [![GoDoc](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsecureagent?status.svg)](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsecureagent)

This integration allows you to have the New Relic security agent analyze your application for potentially exploitable vulnerabilities.

**DO NOT** use this integration in your production environment. It is intended only for use in your development and testing phases. Since it will attempt to actually find and exploit vulnerabilities in your code, it may cause data loss or crash the application. Therefore it should only be used with test data in a non-production environment that does not connect to any production services.

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
NEW_RELIC_SECURITY_CONFIG_PATH=/path/to/your/myappsecurity.yaml
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

Generate traffic against your application for the IAST agent to detect vulnerabilities. Once vulnerabilities are detected they will be reported in the vulnerabilities list.

You can also use environment variables or in-app configuration functions to set all of these
configuration values.

For more information, see
[godocs](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsecureagent).
