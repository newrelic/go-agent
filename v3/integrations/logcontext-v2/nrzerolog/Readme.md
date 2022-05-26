# Zerolog In Context

This plugin for zerolog implements the logs in context tooling for the go agent. This hook
function can be added to any zerolog logger, and will automatically collect the log data
from zerolog, and send it to New Relic through the go agent. The following Logging features
are supported by this plugin in the current release:

| Logging Feature | Supported |
| ------- | --------- |
| Forwarding | :heavy_check_mark: |
| Metrics | :heavy_check_mark: |
| Enrichment | :x: |

## Installation

The nrzerolog plugin, and the go-agent need to be integrated into your code
in order to use this tool. Make sure to set `newrelic.ConfigZerologPluginEnabled(true)`
in your config settings for the application. This will enable log forwarding and metrics
in the go agent, as well as let the agent know that the zerolog pluging is in use.
If you want to disable metrics, set `newrelic.ConfigAppLogMetricsEnabled(false),`.
If you want to disable log forwarding, set `newrelic.ConfigAppLogForwardingEnabled(false),`.
Note that the agent sets the default number of logs per harverst cycle to 10000, but that
number may be reuced by the server. You can manually set this number by setting
`newrelic.ConfigAppLogForwardingMaxSamplesStored(123),`.

The following example will shows how to install and set up your code to send logs to new relic from zerolog.

```go

import (
    "github.com/rs/zerolog"
    "github.com/newrelic/go-agent/v3/newrelic"
    "github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrzerolog"
)

func main() {
    baseLogger := zerolog.New(os.Stdout)

	app, err := newrelic.NewApplication(
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigAppName("NRZerolog Example"),
		newrelic.ConfigInfoLogger(os.Stdout),
		newrelic.ConfigZerologPluginEnabled(true),
		newrelic.ConfigDistributedTracerEnabled(true),
	)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	nrHook := nrzerolog.NewRelicHook{
		App: app,
	}

	nrLogger := baseLogger.Hook(nrHook)
	nrLogger.Info().Msg("Hello World")
}
```

## Usage

When zerolog hooks a logger object, a copy of that logger is made and the 
hook is appended to it. Zerolog will *Never* check if you duplicate information
in your logger, so it is very important to treat each logger as an immutable step
in how you generate your logs. If you apply a hook function to a logger that is
already hooked, it will capture all logs generated from that logger twice.  
To avoid that issue, we recommend that you create a base logger object with the 
formatting settings you prefer, then new hooked loggers from that base logger.

The plugin captures the log level, and the message from zerolog. It will generate a
timestamp at the moment the hook function is called in zerolog. In most cases, this
timestamp will be the same as the time posted in zerolog, however in some corner
cases, a very small amount of offset is possible.


