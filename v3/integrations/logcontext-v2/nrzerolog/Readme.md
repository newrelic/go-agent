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
in order to use this tool. Make sure to set `newrelic.ConfigAppLogForwardingEnabled(true)`
in your config settings for the application. This will enable log forwarding
in the go agent. If you want to disable metrics, set `newrelic.ConfigAppLogMetricsEnabled(false),`.
Note that the agent sets the default number of logs per harverst cycle to 10000, but that
number may be reduced by the server. You can manually set this number by setting
`newrelic.ConfigAppLogForwardingMaxSamplesStored(123),`.

The following example will shows how to install and set up your code to send logs to new relic from zerolog.

```go

import (
    "github.com/rs/zerolog"
    "github.com/newrelic/go-agent/v3/newrelic"
    "github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrzerolog"
)

func main() {
    // Initialize a zerolog logger
	baseLogger := zerolog.New(os.Stdout)

	app, err := newrelic.NewApplication(
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigAppName("NRZerolog Example"),
		newrelic.ConfigInfoLogger(os.Stdout),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Send logs to New Relic outside of a transaction
	nrHook := nrzerolog.NewRelicHook{
		App: app,
	}

	// Wrap logger with New Relic Hook
	nrLogger := baseLogger.Hook(nrHook)
	nrLogger.Info().Msg("Hello World")

	// Send logs to New Relic inside of a transaction
	txn := app.StartTransaction("My Transaction")
	ctx := newrelic.NewContext(context.Background(), txn)

	nrTxnHook := nrzerolog.NewRelicHook{
		App:     app,
		Context: ctx,
	}

	txnLogger := baseLogger.Hook(nrTxnHook)
	txnLogger.Debug().Msg("This is a transaction log")

	txn.End()
}
```

## Usage

Please enable the agent to ingest your logs by calling newrelic.ConfigAppLogForwardingEnabled(true),
when setting up your application. This is not enabled by default.

This integration for the zerolog logging frameworks uses a built in feature
of the zerolog framework called hook functions. Zerolog loggers can be modified
to have hook functions run on them before each time a write is executed. When a
logger is hooked, meaning a hook function was added to that logger with the Hook() 
funciton, a copy of that logger is created with those changes. Note that zerolog
will *never* attempt to verify that any hook functions have not been not duplicated, or 
that fields are not repeated in any way. As a result, we recommend that you create
a base logger that is configured in the way you prefer to use zerolog. Then you
create hooked loggers to send log data to New Relic from that base logger.

The plugin captures the log level, and the message from zerolog. It will also collect
distributed tracing data from your transaction context. At the moment the hook function is
called in zerolog, a timestamp will be generated for your log. In most cases, this
timestamp will be the same as the time posted in the zerolog log message, however it is possible that
there could be a slight offset depending on the the performance of your system.


