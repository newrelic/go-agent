## 3.20.4

### Fixed
* nrmssql driver updated to use version maintained by Microsoft
* bug where error messages were not truncated to the maximum size, and would get dropped if they were too large
* bug where number of span events was hard coded to 1000, and config setting was being ignored

### Added
* improved performance of ignore error code checks in agent
* HTTP error codes can be set as expected by adding them to ErrorCollector.ExpectStatusCodes in the config

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.

We also recommend using the latest version of the Go language. At minimum, you should at least be using no version of Go older than what is supported by the Go team themselves.

See the [Go Agent EOL Policy](https://docs.newrelic.com/docs/apm/agents/go-agent/get-started/go-agent-eol-policy/) for details about supported versions of the Go Agent and third-party components.

## 3.20.3

Please note that the v2 go agent is no longer supported according to our EOL policy. 

### Fixed
* Performance Improvements for compression
* nrsnowflake updated to golang 1.17 versions of packages

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.

We also recommend using the latest version of the Go language. At minimum, you should at least be using no version of Go older than what is supported by the Go team themselves.

See the [Go Agent EOL Policy](https://docs.newrelic.com/docs/apm/agents/go-agent/get-started/go-agent-eol-policy/) for details about supported versions of the Go Agent and third-party components.


## 3.20.2

### Added
* New `NoticeExpectedError()` method allows you to capture errors that you are expecting to handle, without triggering alerts

### Fixed
* More defensive harvest cycle code that will avoid crashing even in the event of a panic.
* Update `nats-server` version to avoid known zip-slip exploit
* Update `labstack/echo` version to mitigate known open redirect exploit

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.

We also recommend using the latest version of the Go language. At minimum, you should at least be using no version of Go older than what is supported by the Go team themselves.

See the [Go Agent EOL Policy](https://docs.newrelic.com/docs/apm/agents/go-agent/get-started/go-agent-eol-policy/) for details about supported versions of the Go Agent and third-party components.

## 3.20.1

### Added
* New integration `nrpgx5` v1.0.0 to instrument `github.com/jackc/pgx/v5`. 

### Changed

* Changed the following `TraceOption` function to be consistent with their usage and other related identifier names. The old names remain for backward compatibility, but new code should use the new names. 
   * `WithIgnoredPrefix` -> `WithIgnoredPrefixes`
   * `WithPathPrefix` -> `WithPathPrefixes`
* Implemented better handling of Code Level Metrics reporting when the data (e.g., function names) are excessively long, so that those attributes are suppressed rather than being reported with truncated names. Specifically:
   * Attributes with values longer than 255 characters are dropped.
   * No CLM attributes at all will be attached to a trace if the `code.function` attribute is empty or is longer than 255 characters.
   * No CLM attributes at all will be attached to a trace if both `code.namespace` and `code.filepath` are longer than 255 characters.

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.

We also recommend using the latest version of the Go language. At minimum, you should at least be using no version of Go older than what is supported by the Go team themselves.

See the [Go Agent EOL Policy](https://docs.newrelic.com/docs/apm/agents/go-agent/get-started/go-agent-eol-policy/) for details about supported versions of the Go Agent and third-party components.

## 3.20.0

**PLEASE READ** these changes, and verify your config settings to ensure your application behaves how you intend it to. This release changes some default behaviors in the go agent.

### Added
* The Module Dependency Metrics feature was added. This collects the list of modules imported into your application, to aid in management of your application dependencies, enabling easier vulnerability detection and response, etc.
   * This feature is enabled by default, but may be disabled by explicitly including `ConfigModuleDependencyMetricsEnable(false)` in your application, or setting the equivalent environment variable or `Config` field direclty.
   * Modules may be explicitly excluded from the report via the `ConfigModuleDependencyMetricsIgnoredPrefixes` option.
   * Excluded module names may be redacted via the `ConfigModuleDependencyMetricsRedactIgnoredPrefixes` option. This is enabled by default.
* Application Log Forwarding will now be **ENABLED** by default
   * Automatic application log forwarding is now enabled by default. This means that logging frameworks wrapped with one of the [logcontext-v2 integrations](https://docs.newrelic.com/docs/apm/agents/go-agent/get-started/go-agent-compatibility-requirements/) will automatically send enriched application logs to New Relic with this version of the agent. To learn more about this feature, see the [APM logs in context documentation](https://docs.newrelic.com/docs/logs/logs-context/logs-in-context/). For additional configuration options, see the [Go logs in context documentation](https://docs.newrelic.com/docs/logs/logs-context/configure-logs-context-go). To learn about how to toggle log ingestion on or off by account, see our documentation to [disable automatic](https://docs.newrelic.com/docs/logs/logs-context/disable-automatic-logging) logging via the UI or API.
   * If you are using a logcontext-v2 extension, but don't want the agent to automatically forward logs, please configure `ConfigAppLogForwardingEnabled(false)` in your application.
   * Environment variables have been added for all application logging config options:
   	* `NEW_RELIC_APPLICATION_LOGGING_ENABLED`
	* `NEW_RELIC_APPLICATION_LOGGING_FORWARDING_ENABLED`
	* `NEW_RELIC_APPLICATION_LOGGING_FORWARDING_MAX_SAMPLES_STORED`
	* `NEW_RELIC_APPLICATION_LOGGING_METRICS_ENABLED`
	* `NEW_RELIC_APPLICATION_LOGGING_LOCAL_DECORATING_ENABLED`
* Custom Event Limit Increase
   * This version increases the **DEFAULT** limit of custom events from 10,000 events per minute to 30,000 events per minute. In the scenario that custom events were being limited, this change will allow more custom events to be sent to New Relic. There is also a new configurable **MAXIMUM** limit of 100,000 events per minute. To change the limits, set `ConfigCustomInsightsEventsMaxSamplesStored(limit)` to the limit you want in your application. To learn more about the change and how to determine if custom events are being dropped, see our Explorers Hub [post](https://discuss.newrelic.com/t/send-more-custom-events-with-the-latest-apm-agents/190497).
   * New config option `ConfigCustomInsightsEventsEnabled(false)` can be used to disable the collection of custom events in your application.

### Changed
* Changed the following names to be consistent with their usage and other related identifier names. The old names remain for backward compatibility, but new code should use the new names.
   * `ConfigCodeLevelMetricsIgnoredPrefix` -> `ConfigCodeLevelMetricsIgnoredPrefixes`
   * `ConfigCodeLevelMetricsPathPrefix` -> `ConfigCodeLevelMetricsPathPrefixes`
   * `NEW_RELIC_CODE_LEVEL_METRICS_PATH_PREFIX` -> `NEW_RELIC_CODE_LEVEL_METRICS_PATH_PREFIXES`
   * `NEW_RELIC_CODE_LEVEL_METRICS_IGNORED_PREFIX` -> `NEW_RELIC_CODE_LEVEL_METRICS_IGNORED_PREFIXES`

* When excluding information reported from CodeLevelMetrics via the `IgnoredPrefixes` or `PathPrefixes` configuration fields (e.g., by specifying `ConfigCodeLevelMetricsIgnoredPrefixes` or `ConfigCodeLevelMetricsPathPrefixes`), the names of the ignored prefixes and the configured path prefixes may now be redacted from the agent configuration information sent to New Relic.
   * This redaction is enabled by default, but may be disabled by supplying a `false` value to `ConfigCodeLevelMetricsRedactPathPrefixes` or `ConfigCodeLevelMetricsRedactIgnoredPrefixes`, or by setting the corresponding `Config` fields or environment variables to `false`.

### Fixed
* [#583](https://github.com/newrelic/go-agent/issues/583): fixed a bug in zerologWriter where comma separated fields in log message confused the JSON parser and could cause panics.

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.

We also recommend using the latest version of the Go language. At minimum, you should at least be using no version of Go older than what is supported by the Go team themselves.

See the [Go Agent EOL Policy](https://docs.newrelic.com/docs/apm/agents/go-agent/get-started/go-agent-eol-policy/) for details about supported versions of the Go Agent and third-party components.

## 3.19.2

### Changed
* Updated nrgin integration to more accurately report code locations when code level metrics are enabled.
* The Go Agent and all integrations now require Go version 1.17 or later.
* Updated minimum versions for third-party modules.
  * nrawssdk-v2, nrecho-v4, nrgrpc, nrmongo, nrmysql, nrnats, and nrstan now require Go Agent 3.18.2 or later
  * the Go Agent now requires protobuf 1.5.2 and grpc 1.49.0
* Internal dev process and unit test improvements.

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.

We also recommend using the latest version of the Go language. At minimum, you should at least be using no version of Go older than what is supported by the Go team themselves.

See the [Go Agent EOL Policy](https://docs.newrelic.com/docs/apm/agents/go-agent/get-started/go-agent-eol-policy/) for details about supported versions of the Go Agent and third-party components.


## 3.19.1 - Hotfix Release

### Changed
* Moved the v3/internal/logcontext/nrwriter module to v3/integrations/logcontext-v2/nrwriter

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.

See the [Go Agent EOL Policy](https://docs.newrelic.com/docs/apm/agents/go-agent/get-started/go-agent-eol-policy/) for details about supported versions of the Go Agent and third-party components.

## 3.19.0

### Added
* `logcontext-v2/logWriter` plugin: a new logs in context plugin that supports the standard library logging package.
* `logcontext-v2/zerologWriter` plugin: a new logs in context plugin for zerolog that will replace the old logcontext-v2/zerolog plugin. This plugin is more robust, and will be able to support a richer set of features than the previous plugin.
* see the updated [logs in context documentation](https://docs.newrelic.com/docs/logs/logs-context/configure-logs-context-go) for information about configuration and installation.

### Changed
* the logcontext-v2/zerolog plugin will be deprecated once the 3.17.0 release EOLs.

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.

See the [Go Agent EOL Policy](https://docs.newrelic.com/docs/apm/agents/go-agent/get-started/go-agent-eol-policy/) for details about supported versions of the Go Agent and third-party components.

## 3.18.2

### Added
* Added `WithDefaultFunctionLocation` trace option. This allows the caller to indicate a fall-back function to use for CLM in case no other location was found first.
* Added caching versions of the code-level metrics functions `ThisCodeLocation` and `FunctionLocation` , and trace options `WithThisCodeLocation`  and  `WithFunctionLocation`. These improve performance by caching the result of computing the source code location, and reuse that cached result on all subsequent calls.
* Added a `WithCodeLevelMetrics` trace option to force the collection of CLM data even if it would have been excluded as being out of the configured scope. (Note that CLM data are _never_ collected if CLM is turned off globally or if the `WithoutCodeLevelMetrics` option was specified for the same transaction.)
* Added an exported `CodeLevelMetricsScopeLabelToValue` function to convert a list of strings describing CLM scopes in the same manner as the `NEW_RELIC_CODE_LEVEL_METRICS_SCOPE` environment variable (but as individual string parameters), returning the `CodeLevelMetricsScope` value which corresponds to that set of scopes.
* Added a new `CodeLevelMetricsScopeLabelListToValue` function which takes a comma-separated list of scope names exactly as the `NEW_RELIC_CODE_LEVEL_METRICS_SCOPE` environment variable does, and returns the `CodeLevelMetrics` value corresponding to that set of scopes.
* Added text marshaling and unmarshaling for the `CodeLevelMetricsScope` value, allowing the `CodeLevelMetrics` field of the configuration `struct` to be converted to or from JSON or other text-based encoding representations.

### Changed
* The `WithPathPrefix` trace option now takes any number of `string` parameters, allowing multiple path prefixes to be recognized rather than just one.
* The `FunctionLocation` function now accepts any number of function values instead of just a single one. The first such parameter which indicates a valid function, and for which CLM data are successfully obtained, is the one which will be reported.
* The configuration `struct` field `PathPrefix` is now deprecated with the introduction of a new `PathPrefixes` field. This allows for multiple path prefixes to be given to the agent instead of only a single one. 
* The `NEW_RELIC_CODE_LEVEL_METRICS_SCOPE` environment variable now accepts a comma-separated list of pathnames.

### Fixed
* Improved the implementation of CLM internals to improve speed, robustness, and thread safety.
* Corrected the implementation of the `WrapHandle` and `WrapHandleFunc` functions so that they consistently report the function being invoked by the `http` framework, and improved them to use the new caching functions and ensured they are thread-safe.

This release fixes [issue #557](https://github.com/newrelic/go-agent/issues/557).

### Compatibility Notice
As of release 3.18.0, the API was extended by allowing custom options to be added to calls to the `Application.StartTransaction` method and the `WrapHandle`  and `WrapHandleFunc` functions. They are implemented as variadic functions such that the new option parameters are optional (i.e., zero or more options may be added to the end of the function calls) to be backward-compatible with pre-3.18.0 usage of those functions. This prevents the changes from breaking existing code for typical usage of the agent. However, it does mean those functions' call signatures have changed:
 * `StartTransaction(string)` -> `StartTransaction(string, ...TraceOption)`
 *  `WrapHandle(*Application, string, http.Handler)` -> `WrapHandle(*Application, string, http.Handler, ...TraceOption)`
 *  `WrapHandleFunc(*Application, string, func(http.ResponseWriter, *http.Request))`    -> `WrapHandleFunc(*Application, string, func(http.ResponseWriter, *http.Request), ...TraceOption)`
   
If, for example, you created your own custom interface type which includes the `StartTransaction` method or something that depends on these functions' exact  call semantics, that code will need to be updated accordingly before using version 3.18.0 (or later) of the Go Agent.

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.

See the [Go Agent EOL Policy](https://docs.newrelic.com/docs/apm/agents/go-agent/get-started/go-agent-eol-policy/) for details about supported versions of the Go Agent and third-party components.


## 3.18.1
### Added
* Extended the `IgnoredPrefix` configuration value for Code-Level Metrics so that multiple such prefixes may be given instead of a single one. This deprecates the `IgnoredPrefix` configuration field of `Config.CodeLevelMetrics` in favor of a new slice field `IgnoredPrefixes`. The corresponding configuration option-setting functions `ConfigCodeLevelMetricsIgnoredPrefix` and `WithIgnoredPrefix` now take any number of string parameters to set these values. Since those functions used to take a single string value, this change is backward-compatible with pre-3.18.1 code.  Accordingly, the `NEW_RELIC_CODE_LEVEL_METRICS_IGNORED_PREFIX` environment variable is now a comma-separated list of prefixes.  Fixes [Issue #551](https://github.com/newrelic/go-agent/issues/551).

### Fixed
* Corrected some small errors in documentation of package features. Fixes [Issue #550](https://github.com/newrelic/go-agent/issues/550)

### Compatibility Notice
As of release 3.18.0, the API was extended by allowing custom options to be added to calls to the `Application.StartTransaction` method and the `WrapHandle` and `WrapHandleFunc` functions. They are implemented as variadic functions such that the new option parameters are optional (i.e., zero or more options may be added to the end of the function calls) to be backward-compatible with pre-3.18.0 usage of those functions. This prevents the changes from breaking existing code for typical usage of the agent. However, it does mean those functions' call signatures have changed:
 * `StartTransaction(string)` -> `StartTransaction(string, ...TraceOption)`
 *  `WrapHandle(*Application, string, http.Handler)` -> `WrapHandle(*Application, string, http.Handler, ...TraceOption)`
 *  `WrapHandleFunc(*Application, string, func(http.ResponseWriter, *http.Request))`	-> `WrapHandleFunc(*Application, string, func(http.ResponseWriter, *http.Request), ...TraceOption)`
   
If, for example, you created your own custom interface type which includes the `StartTransaction` method or something that depends on these functions' exact call semantics, that code will need to be updated accordingly before using version 3.18.0 (or later) of the Go Agent.

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.

See the [Go Agent EOL Policy](https://docs.newrelic.com/docs/apm/agents/go-agent/get-started/go-agent-eol-policy/) for details about supported versions of the Go Agent and third-party components.

## 3.18.0
### Added
* Code-Level Metrics are now available for instrumented transactions. This is off by default but once enabled via `ConfigCodeLevelMetricsEnabled(true)` transactions will include information about the location in the source code where `StartTransaction` was invoked.
   * Adds information about where in your source code transaction traces originated.
   * See the Go Agent documentation for details on [configuring](https://docs.newrelic.com/docs/apm/agents/go-agent/configuration/go-agent-code-level-metrics-config) Code-Level Metrics and how to [instrument](https://docs.newrelic.com/docs/apm/agents/go-agent/instrumentation/go-agent-code-level-metrics-instrument) your code using them.
* New V2 logs in context plugin is available for Logrus, packed with all the features you didn't know you wanted:
   * Automatic Log Forwarding
   * Log Metrics
   * Capture logs anywhere in your code; both inside or outside of a transaction.
   * Use the Logrus formatting package of your choice
   * Local Log Decorating is now available for the new logcontext-v2/nrlogrus plugin only. This is off by default but can be enabled with `ConfigAppLogForwardingEnabled(true)`.

### Fixed
 * Fixed issue with custom event limits and number of DT Spans to more accurately follow configured limits.

### Compatibility Notice
This release extends the API by allowing custom options to be added to calls to the `Application.StartTransaction` method and the `WrapHandle` and `WrapHandleFunc` functions. They are implemented as variadic functions such that the new option parameters are optional (i.e., zero or more options may be added to the end of the function calls) to be backward-compatible with pre-3.18.0 usage of those functions.
This prevents the changes from breaking existing code for typical usage of the agent. However, it does mean those functions' call signatures have changed:
 * `StartTransaction(string)` -> `StartTransaction(string, ...TraceOption)`
 * `WrapHandle(*Application, string, http.Handler)` -> `WrapHandle(*Application, string, http.Handler, ...TraceOption)`
 * `WrapHandleFunc(*Application, string, func(http.ResponseWriter, *http.Request))` -> `WrapHandleFunc(*Application, string, func(http.ResponseWriter, *http.Request), ...TraceOption)`

If, for example, you created your own custom interface type which includes the `StartTransaction` method or something that depends on these functions' exact call semantics, that code will need to be updated accordingly before using version 3.18.0 of the Go Agent.

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.
* Note that the oldest supported version of the Go Agent is 3.6.0.

## 3.17.0
### Added
* Logs in context now supported for zerolog.
* This is a quick way to view logs no matter where you are in the platform.
	* Adds support for logging metrics which shows the rate of log messages by severity in the Logs chart in the APM Summary view. This is enabled by default in this release.
	* Adds support for forwarding application logs to New Relic. This automatically sends application logs that have been enriched to power APM logs in context. This is disabled by default in this release. This will be on by default in a future release.
	* To learn more about APM logs in context see the documentation [here](https://docs.newrelic.com/docs/logs/logs-context/logs-in-context).
	* Includes the `RecordLog` function for recording log data from a single log entry
	* An integrated plugin for zerolog to automatically ingest log data with the Go Agent.
	* Resolves [issue 178](https://github.com/newrelic/go-agent/issues/178), [issue 488](https://github.com/newrelic/go-agent/issues/488), [issue 489](https://github.com/newrelic/go-agent/issues/489), [issue 490](https://github.com/newrelic/go-agent/issues/490), and [issue 491](https://github.com/newrelic/go-agent/issues/491) .
* Added integration for MS SQL Server ([PR 425](https://github.com/newrelic/go-agent/pull/425); thanks @ishahid91!)
	* This introduces the `nrmssql` integration v1.0.0.
* Added config function `ConfigCustomInsightsEventsMaxSamplesStored` for limiting the number of samples stored in a custom insights event. Fixes [issue 476](https://github.com/newrelic/go-agent/issues/476)

### Fixed
* Improved speed of building distributed trace header JSON payload. Fixes [issue 505](https://github.com/newrelic/go-agent/issues/505).
* Renamed the gRPC attribute names from  `GrpcStatusLevel`, `GrpcStatusMessage`, and `GrpcStatusCode` to `grpcStatusLevel`, `grpcStatusMessage`, and `grpcStatusCode` respectively, to conform to existing naming conventions for New Relic agents. Fixes [issue 492](https://github.com/newrelic/go-agent/issues/492).
* Updated `go.mod` for the `nrgin` integration to mitigate security issue in 3rd party dependency.
* Updated `go.mod` for the `nrawssdk-v1` integration to properly reflect its dependency on version 3.16.0 of the Go Agent.
* Updated `go.mod` for the `nrlambda` integration to require `aws-lambda-go` version 1.20.0. ([PR 356](https://github.com/newrelic/go-agent/pull/356); thanks MattWhelan!)

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.
* Note that the oldest supported version of the Go Agent is 3.6.0.

# ChangeLog
## 3.16.1
### Fixed
* Changed dependency on gRPC from v1.27.0 to v1.39.0. This in turn changes gRPC's dependency on `x/crypto` to v0.0.0-20200622213623-75b288015ac9, which fixes a security vulnerability in the `x/crypto` standard library module. Fixes [issue #451](https://github.com/newrelic/go-agent/issues/451).
* Incremented version number of the `nrawssdk-v1` integration from v1.0.1 to v1.1.0 to resolve an incompatibility issue due to changes to underlying code. Fixes [issue #499](https://github.com/newrelic/go-agent/issues/499)

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.

## 3.16.0
### Added
* Distributed Tracing is now the default mode of operation. It may be disabled by user configuration if so desired. [PR #495](https://github.com/newrelic/go-agent/pull/495)
   * To disable DT, add `newrelic.ConfigDistributedTracerEnabled(false)` to your application configuration.
   * To change the reservoir limit for how many span events are to be collected per harvest cycle from the default, add `newrelic.ConfigDistributedTracerReservoirLimit(`*newlimit*`)` to your application configuration.
   * The reservoir limit's default was increased from 1000 to 2000.
   * The maximum reservoir limit supported is 10,000.
* Note that Cross Application Tracing is now deprecated.
* Added support for gathering memory statistics via `PhysicalMemoryBytes` functions for OpenBSD.

### Fixed
* Corrected some example code to be cleaner.
* Updated version of nats-streaming-server. [PR #458](https://github.com/newrelic/go-agent/pull/458)
* Correction to nrpkgerrors so that `nrpkgerrors.Wrap`  now checks if the error it is passed has attributes, and if it does, copies them into the New Relic error it creates.
This fixes [issue #409](https://github.com/newrelic/go-agent/issues/409) via [PR #441](https://github.com/newrelic/go-agent/pull/441).
   * This increments the `nrpkgerrors` version to v1.1.0.

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.


## 3.15.2
### Added
* Strings logged via the Go Agent's built-in logger will have strings of the form `license_key=`*hex-string* changed to `license_key=[redacted]` before they are output, regardless of severity level, where *hex-string* means a sequence of upper- or lower-case hexadecimal digits and dots ('.'). This incorporates [PR #415](https://github.com/newrelic/go-agent/pull/415).

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.

## 3.15.1

### Fixed

* Updated support for SQL database instrumentation across the board for the Go Agent’s database integrations to more accurately extract the database table name from SQL queries. Fixes [Issue #397](https://github.com/newrelic/go-agent/issues/397).

* Updated the `go.mod` file in the `nrecho-v4` integration to require version 4.5.0 of the `github.com/labstack/echo` package. This addresses a security concern arising from downstream dependencies in older versions of the echo package, as described in the [release notes](https://github.com/labstack/echo/releases/tag/v4.5.0) for `echo` v4.5.0.

### ARM64 Compatibility Note

The New Relic Go Agent is implemented in platform-independent Go, and supports (among the other platforms which run Go) ARM64/Graviton2 using Go 1.17+.

### Support Statement

New Relic recommends that you upgrade the agent regularly to ensure that you’re getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.



## 3.15.0

### Fixed
* Updated mongodb driver version to 1.5.1 to fix security issue in external dependency. Fixes [Issue #358](https://github.com/newrelic/go-agent/issues/358) and [Issue #370](https://github.com/newrelic/go-agent/pull/370).

* Updated the `go.mod` file in the `nrgin` integration to require version 1.7.0 of the `github.com/gin-gonic/gin` package. This addresses [CVE-2020-28483](https://github.com/advisories/GHSA-h395-qcrw-5vmq) which documents a vulnerability in versions of `github.com/gin-gonic/gin` earlier than 1.7.0. 


### Added
* New integration `nrpgx` added to provide the same functionality for instrumenting Postgres database queries as the existing `nrpq` integration, but using the [pgx](https://github.com/jackc/pgx) driver instead. This only covers (at present) the use case of the `pgx` driver with the standard library `database/sql`. Fixes [Issue #142](https://github.com/newrelic/go-agent/issues/142) and [Issue #292](https://github.com/newrelic/go-agent/issues/292)

### Changed
* Enhanced debugging logs so that New Relic license keys are redacted from the log output. Fixes [Issue #353](https://github.com/newrelic/go-agent/issues/353).

* Updated the advice in `GUIDE.md` to have correct `go get` commands with explicit reference to `v3`. 

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you're getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.

## 3.14.1

### Fixed
* A typographical error in the nrgrpc unit tests was fixed. Fixes [Issue #344](https://github.com/newrelic/go-agent/issues/344).
  This updates the nrgrpc integration to version 1.3.1.

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you're getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.


## 3.14.0

### Fixed
* Integration tags and `go.mod` files for integrations were updated so that [pkg.go.dev]() displays the documentation for each integration correctly.
* The `nrgrpc` server integration was reporting all non-`OK` grpc statuses as errors. This has now been changed so that only selected grpc status codes will be reported as errors. Others are shown (via transaction attributes) as "warnings" or "informational" messages. There is a built-in set of defaults as to which status codes are reported at which severity levels, but this may be overridden by the caller as desired. Also supports custom grpc error handling functions supplied by the user.
   * This is implemented by adding `WithStatusHandler()` options to the end of the `UnaryServerInterceptor()` and `StreamServerInterceptor()` calls, thus extending the capability of those functions while retaining the existing functionality and usage syntax for backward compatibility.
* Added advice on the recommended usage of the `app.WaitForConnection()` method. Fixes [Issue #296](https://github.com/newrelic/go-agent/issues/296)

### Added
* Added a convenience function to build distributed trace header set from a JSON string for use with the `AcceptDistributedTraceHeaders()` method. Normally, you must create a valid set of HTTP headers representing the trace identification information from the other trace so the new trace will be associated with it. This needs to be in a Go `http.Header` type value.
   * If working only in Go, this may be just fine as it is. However, if the other trace information came from another source, possibly in a different language or environment, it is often the case that the trace data is already presented to you in the form of a JSON string.
   * This new function, `DistributedTraceHeadersFromJSON()`, creates the required `http.Header` value from the JSON string without requiring manual effort on your part. 
   * We also provide a new all-in-one method `AcceptDistributedTraceHeadersFromJSON()` to be used in place of `AcceptDistributedTraceHeaders()`. It accepts a JSON string rather than an `http.Header`, adding its trace info to the new transaction in one step.
   * Fixes [Issue #331](https://github.com/newrelic/go-agent/issues/331)

### Changed
* Improved the NR AWS SDK V2 integration to use the current transaction rather than the one passed in during middleware creation, if `nil` is passed into nrawssdk-v2.AppendMiddlewares. Thanks to @HenriBeck for noticing and suggesting improvement, and thanks to @nc-wittj for the fantastic PR! [#328](https://github.com/newrelic/go-agent/pull/328)

### Support Statement
New Relic recommends that you upgrade the agent regularly to ensure that you're getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach end-of-life.

## 3.13.0

### Fixed
* Replaced the NR AWS SDK V2 integration for the v3 agent with a new version that works. See the v3/integrations/nrawssdk-v2/example/main.go file for an example of how to use it. Issues [#250](https://github.com/newrelic/go-agent/issues/250) and [#288](https://github.com/newrelic/go-agent/issues/288) are fixed by this PR. [#309](https://github.com/newrelic/go-agent/pull/309)

* Fixes issue [#221](https://github.com/newrelic/go-agent/issues/221): grpc errors reported in code watched by `UnaryServerInterceptor()` or `StreamServerInterceptor()` now create error events which are reported to the UI with the error message string included.  [#317](https://github.com/newrelic/go-agent/pull/317)

* Fixes documentation in `GUIDE.md` for `txn.StartExternalSegment()` to reflect the v3 usage. Thanks to @abeltay for calling this to our attention and submitting PR [#320](https://github.com/newrelic/go-agent/pull/320).

### Changes
* The v3/examples/server/main.go example now uses `newrelic.ConfigFromEnvironment()`, rather than explicitly pulling in the license key with `newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY"))`. The team is starting to use this as a general systems integration testing script, and this facilitates testing with different settings enabled.

### Support Statement
* New Relic recommends that you upgrade the agent regularly to ensure that you're getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach [end-of-life](https://docs.newrelic.com/docs/using-new-relic/cross-product-functions/install-configure/notification-changes-new-relic-saas-features-distributed-software).

## 3.12.0

### Changes
* Updated `CHANGELOG.md` release notes language, to correct typographical errors and
clean up grammar. [#289](https://github.com/newrelic/go-agent/issues/289)

### Fixed
* When using DAX to query a dynamodb table, the New Relic instrumentation
panics with a `nil dereference` error. This was due to the way that the
request is made internally such that there is no `HTTPRequest.Header` 
defined, but one was expected. This correction checks for the existence
of that header and takes an appropriate course of action if one is not
found. [#287](https://github.com/newrelic/go-agent/issues/287) Thanks to
@odannyc for reporting the issue and providing a pull request with a suggested
fix.

### Support Statement
* New Relic recommends that you upgrade the agent regularly to ensure that you're getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach [end-of-life](https://docs.newrelic.com/docs/using-new-relic/cross-product-functions/install-configure/notification-changes-new-relic-saas-features-distributed-software).

## 3.11.0

### New Features
* Aerospike is now included on the list of recognized datastore names. Thanks @vkartik97 for your PR! [#233](https://github.com/newrelic/go-agent/pull/233)
* Added support for verison 8 of go-redis. Thanks @ilmimris for adding this instrumentation! [#251](https://github.com/newrelic/go-agent/pull/251)

### Changes
* Changed logging level for messages resulting from Infinite Tracing load balancing operations. These were previously logged as errors, and now they are debugging messages. [#276](https://github.com/newrelic/go-agent/pull/276)

### Fixed
* When the agent is configured with `cfg.ErrorCollector.RecordPanics` set to `true`, panics would be recorded by New Relic, but stack traces would not be logged as the Go Runtime usually does. The agent now logs stack traces from within its panic handler, providing similar functionality. [#278](https://github.com/newrelic/go-agent/pull/278)
* Added license files to some integrations packages to ensure compatibility with package.go.dev. Now the documentation for our integrations shows up again on go.docs.

### Support statement
* New Relic recommends that you upgrade the agent regularly to ensure that you're getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach [end-of-life](https://docs.newrelic.com/docs/using-new-relic/cross-product-functions/install-configure/notification-changes-new-relic-saas-features-distributed-software).

## 3.10.0

### New Features
* To keep up with the latest security protocols implemented by Amazon Web
  Services, the agent now uses [AWS
  IMDSv2](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/configuring-instance-metadata-service.html)
  to find utilization data. [#249](https://github.com/newrelic/go-agent/pull/249)

### Changes
* Updated the locations of our license files so that Go docs https://pkg.go.dev 
  will display our agent. Thanks @tydavis for your PR to fix this! [#254](https://github.com/newrelic/go-agent/pull/254)
* Added an Open Source repo linter GitHub action that runs on push. [#262](https://github.com/newrelic/go-agent/pull/262)
* Updated the README.md file to correctly show the support resources from New Relic. [#255](https://github.com/newrelic/go-agent/pull/255)

### Support statement
* New Relic recommends that you upgrade the agent regularly to ensure that you're getting the latest features and performance benefits. Additionally, older releases will no longer be supported when they reach [end-of-life](https://docs.newrelic.com/docs/using-new-relic/cross-product-functions/install-configure/notification-changes-new-relic-saas-features-distributed-software).

## 3.9.0

### Changes
* When sending Serverless telemetry using the `nrlambda` integration, support an externally-managed named pipe.

## 3.8.1

### Bug Fixes

* Fixed an issue that could cause orphaned Distributed Trace spans when using
  SQL instrumentation like `nrmysql`.

## 3.8.0

### Changes
* When marking a transaction as a web transaction using 
[Transaction.SetWebRequest](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.SetWebRequest), 
it is now possible to include a `Host` field in the 
[WebRequest](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#WebRequest)
 struct, which defaults to the empty string.

### Bug Fixes

* The `Host` header is now being correctly captured and recorded in the 
 `request.headers.host` attribute, as described 
 [here](https://docs.newrelic.com/docs/agents/go-agent/instrumentation/go-agent-attributes#requestHeadersHost).
*  Previously, the timestamps on Spans and Transactions were being written
   using different data types, which sometimes caused rounding errors that
   could cause spans to be offset incorrectly in the UI. This has been fixed.

## 3.7.0

### Changes

* When `Config.Transport` is nil, no longer use the `http.DefaultTransport`
  when communicating with the New Relic backend.  This addresses an issue with
  shared transports as described in https://github.com/golang/go/issues/33006.

* If a timeout occurs when attempting to send data to the New Relic backend,
  instead of dropping the data, we save it and attempt to send it with the
  next harvest.  Note data retention limits still apply and the agent will
  still start to drop data when these limits are reached. We attempt to keep
  the highest priority events and traces.

## 3.6.0

### New Features

* Added support for [adding custom attributes directly to
  spans](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Segment.AddAttribute).
  These attributes will be visible when looking at spans in the Distributed
  Tracing UI.

  Example:
  ```go
  txn := newrelic.FromContext(r.Context())
  sgmt := txn.StartSegment("segment1")
  defer sgmt.End()
  sgmt.AddAttribute("mySpanString", "hello")
  sgmt.AddAttribute("mySpanInt", 123)
  ```

* Custom attributes added to the transaction with `txn.AddAttribute` are now
  also added to the root Span Event and will be visible when looking at the
  span in the Distributed Tracing UI. These custom attributes can be disabled
  from all destinations using `Config.Attributes.Exclude` or disabled from Span
  Events specifically using `Config.SpanEvents.Attributes.Exclude`.

* Agent attributes added to the transaction are now also added to the root Span
  Event and will be visible when looking at the span in the Distributed Tracing
  UI. These attributes include the `request.uri` and the `request.method` along
  with all other attributes listed in the [attributes section of our
  godocs](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#pkg-constants).
  These agent attributes can be disabled from all destinations using
  `Config.Attributes.Exclude` or disabled from Span Events specifically using
  `Config.SpanEvents.Attributes.Exclude`.

### Bug Fixes

* Fixed an issue where it was impossible to exclude the attributes
  `error.class` and `error.message` from the root Span Event. This issue has
  now been fixed. These attributes can now be excluded from all Span Events
  using `Config.Attributes.Exclude` or `Config.SpanEvents.Attributes.Exclude`.
  
* Fixed an issue that caused Go's data race warnings to trigger in certain situations 
  when using the `newrelic.NewRoundTripper`. There were no reports of actual data corruption, 
  but now the warnings should be resolved. Thank you to @blixt for bringing this to our 
  attention!

## 3.5.0

### New Features

* Added support for [Infinite Tracing on New Relic
  Edge](https://docs.newrelic.com/docs/understand-dependencies/distributed-tracing/enable-configure/enable-distributed-tracing).

  Infinite Tracing observes 100% of your distributed traces and provides
  visualizations for the most actionable data so you have the examples of errors
  and long-running traces so you can better diagnose and troubleshoot your systems.

  You [configure your
  agent](https://docs.newrelic.com/docs/agents/go-agent/configuration/go-agent-configuration#infinite-tracing)
  to send traces to a trace observer in New Relic Edge.  You view your
  distributed traces through the New Relic’s UI. There is no need to install a
  collector on your network.

  Infinite Tracing is currently available on a sign-up basis. If you would like to
  participate, please contact your sales representative.
  
  **As part of this change, the Go Agent now has an added dependency on gRPC.** 
  This is true whether or not you enable the Infinite Tracing feature. The gRPC dependencies include these two libraries:
  * [github.com/golang/protobuf](https://github.com/golang/protobuf) v1.3.3
  * [google.golang.org/grpc](https://github.com/grpc/grpc-go) v1.27.0

  You can see the changes in the [go.mod file](v3/go.mod) 

  **As part of this change, the Go Agent now has an added dependency on gRPC.** 
  This is true whether or not you enable the Infinite Tracing feature. The gRPC dependencies include these two libraries:
  * [github.com/golang/protobuf](https://github.com/golang/protobuf) v1.3.3
  * [google.golang.org/grpc](https://github.com/grpc/grpc-go) v1.27.0

  You can see the changes in the [go.mod file](v3/go.mod) 

### Changes

* [`nrgin.Middleware`](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgin#Middleware)
  uses
  [`Context.FullPath()`](https://godoc.org/github.com/gin-gonic/gin#Context.FullPath)
  for transaction names when using Gin version 1.5.0 or greater.  Gin
  transactions were formerly named after the
  [`Context.HandlerName()`](https://godoc.org/github.com/gin-gonic/gin#Context.HandlerName),
  which uses reflection.  This change improves transaction naming and reduces
  overhead.  Please note that because your transaction names will change, you
  may have to update any related dashboards and alerts to match the new name.
  If you wish to continue using `Context.HandlerName()` for your transaction
  names, use
  [`nrgin.MiddlewareHandlerTxnNames`](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgin#MiddlewareHandlerTxnNames)
  instead.

  ```go
  // Transactions previously named
  "GET main.handleGetUsers"
  // will be change to something like this match the full path
  "GET /user/:id"
  ```

  Note: As part of agent release v3.4.0, a v2.0.0 tag was added to the nrgin
  package.  When using go modules however, it was impossible to install this
  latest version of nrgin.  The v2.0.0 tag has been removed and replaced with
  v1.1.0.

## 3.4.0

### New Features

* Attribute `http.statusCode` has been added to external span events
  representing the status code on an http response.  This attribute will be
  included when added to an ExternalSegment in one of these three ways:

  1. Using
     [`NewRoundTripper`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewRoundTripper)
     with your http.Client
  2. Including the http.Response as a field on your
     [`ExternalSegment`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ExternalSegment)
  3. Using the new
     [`ExternalSegment.SetStatusCode`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ExternalSegment.SetStatusCode)
     API to set the status code directly

  To exclude the `http.statusCode` attribute from span events, update your
  agent configuration like so, where `cfg` is your [`newrelic.Config`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Config) object.

  ```go
  cfg.SpanEvents.Attributes.Exclude = append(cfg.SpanEvents.Attributes.Exclude, newrelic.SpanAttributeHTTPStatusCode)
  ```

* Error attributes `error.class` and `error.message` are now included on the
 span event in which the error was noticed, or on the root span if an error
 occurs in a transaction with no segments (no chid spans). Only the most recent error
 information is added to the attributes; prior errors on the same span are
 overwritten.

  To exclude the `error.class` and/or `error.message` attributes from span events, update your
  agent configuration like so, where `cfg` is your [`newrelic.Config`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Config) object.

  ```go
  cfg.SpanEvents.Attributes.Exclude = append(cfg.SpanEvents.Attributes.Exclude, newrelic.newrelic.SpanAttributeErrorClass, newrelic.SpanAttributeErrorMessage)
  ```

### Changes

* Use
  [`Context.FullPath()`](https://godoc.org/github.com/gin-gonic/gin#Context.FullPath)
  for transaction names when using Gin version 1.5.0 or greater.  Gin
  transactions were formerly named after the
  [`Context.HandlerName()`](https://godoc.org/github.com/gin-gonic/gin#Context.HandlerName),
  which uses reflection.  This change improves transaction naming and reduces
  overhead.  Please note that because your transaction names will change, you
  may have to update any related dashboards and alerts to match the new name.

  ```go
  // Transactions previously named
  "GET main.handleGetUsers"
  // will be change to something like this match the full path
  "GET /user/:id"
  ```
* If you are using any of these integrations, you must upgrade them when you
 upgrade the agent:
    * [nrlambda v1.1.0](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrlambda)
    * [nrmicro v1.1.0](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrmicro)
    * [nrnats v1.1.0](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrnats)
    * [nrstan v1.1.0](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrstan)
    
### Known Issues and Workarounds

* If a .NET agent is initiating distributed traces as the root service, you must 
  update that .NET agent to version 8.24 or later before upgrading your downstream 
  Go New Relic agents to this agent release.

## 3.3.0

### New Features

* Added support for GraphQL in two new integrations:
  * [graph-gophers/graphql-go](https://github.com/graph-gophers/graphql-go)
  with
  [v3/integrations/nrgraphgophers](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgraphgophers).
    * [Documentation](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgraphgophers)
    * [Example](v3/integrations/nrgraphgophers/example/main.go)
  * [graphql-go/graphql](https://github.com/graphql-go/graphql)
  with
  [v3/integrations/nrgraphqlgo](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgraphqlgo).
    * [Documentation](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgraphqlgo)
    * [Example](v3/integrations/nrgraphqlgo/example/main.go)

* Added database instrumentation support for
  [snowflakedb/gosnowflake](https://github.com/snowflakedb/gosnowflake).
  * [Documentation](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsnowflake)
  * [Example](v3/integrations/nrsnowflake/example/main.go)

### Changes

* When using
  [`newrelic.StartExternalSegment`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#StartExternalSegment)
  or
  [`newrelic.NewRoundTripper`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewRoundTripper),
  if existing cross application tracing or distributed tracing headers are
  present on the request, they will be replaced instead of added.

* The
  [`FromContext`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#FromContext)
  API which allows you to pull a Transaction from a context.Context will no
  longer panic if the provided context is nil.  In this case, a nil is
  returned.
  
### Known Issues and Workarounds

* If a .NET agent is initiating distributed traces as the root service, you must 
  update that .NET agent to version 8.24 or later before upgrading your downstream 
  Go New Relic agents to this agent release.

## 3.2.0

### New Features

* Added support for `v7` of [go-redis/redis](https://github.com/go-redis/redis)
  in the new [v3/integrations/nrredis-v7](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrredis-v7)
  package.
  * [Documentation](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrredis-v7)
  * [Example](v3/integrations/nrredis-v7/example/main.go)

### Changes

* Updated Gorilla instrumentation to include request time spent in middlewares.
  Added new `nrgorilla.Middleware` and deprecated `nrgorilla.InstrumentRoutes`.
  Register the new middleware as your first middleware using
  [`Router.Use`](https://godoc.org/github.com/gorilla/mux#Router.Use). See the
  [godocs
  examples](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgorilla)
  for more details.

  ```go
  r := mux.NewRouter()
  // Always register the nrgorilla.Middleware first.
  r.Use(nrgorilla.Middleware(app))

  // All handlers and custom middlewares will be instrumented.  The
  // transaction will be available in the Request's context.
  r.Use(MyCustomMiddleware)
  r.Handle("/", makeHandler("index"))

  // The NotFoundHandler and MethodNotAllowedHandler must be instrumented
  // separately using newrelic.WrapHandle.  The second argument to
  // newrelic.WrapHandle is used as the transaction name; the string returned
  // from newrelic.WrapHandle should be ignored.
  _, r.NotFoundHandler = newrelic.WrapHandle(app, "NotFoundHandler", makeHandler("not found"))
  _, r.MethodNotAllowedHandler = newrelic.WrapHandle(app, "MethodNotAllowedHandler", makeHandler("method not allowed"))

  http.ListenAndServe(":8000", r)
  ```

### Known Issues and Workarounds

* If a .NET agent is initiating distributed traces as the root service, you must 
  update that .NET agent to version 8.24 or later before upgrading your downstream 
  Go New Relic agents to this agent release.

## 3.1.0

### New Features

* Support for W3C Trace Context, with easy upgrade from New Relic trace context.

  Distributed Tracing now supports W3C Trace Context headers for HTTP and
  gRPC protocols when distributed tracing is enabled.  Our implementation can
  accept and emit both W3C trace header format and New Relic trace header
  format.  This simplifies agent upgrades, allowing trace context to be
  propagated between services with older and newer releases of New Relic
  agents.  W3C trace header format will always be accepted and emitted.  New
  Relic trace header format will be accepted, and you can optionally disable
  emission of the New Relic trace header format.

  When distributed tracing is enabled with
  `Config.DistributedTracer.Enabled = true`, the Go agent will now accept
  W3C's `traceparent` and `tracestate` headers when calling
  [`Transaction.AcceptDistributedTraceHeaders`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.AcceptDistributedTraceHeaders).  When calling
  [`Transaction.InsertDistributedTraceHeaders`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.InsertDistributedTraceHeaders), the Go agent will include the
  W3C headers along with the New Relic distributed tracing header, unless
  the New Relic trace header format is disabled using
  `Config.DistributedTracer.ExcludeNewRelicHeader = true`.

* Added support for [elastic/go-elasticsearch](https://github.com/elastic/go-elasticsearch)
  in the new [v3/integrations/nrelasticsearch-v7](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrelasticsearch-v7)
  package.

* At this time, the New Relic backend has enabled support for real time
  streaming.  Versions 2.8 and above will now send data to New Relic every five
  seconds, instead of every minute.  As a result, transaction, error, and custom
  events will now be available in New Relic One and Insights dashboards in near
  real time.
  
### Known Issues and Workarounds

* If a .NET agent is initiating distributed traces as the root service, you must update 
  that .NET agent to version 8.24 or later before upgrading your downstream 
  Go New Relic agents to this agent release.

## 3.0.0

We are pleased to announce the release of Go Agent v3.0.0!  This is a major release
that includes some breaking changes that will simplify your future use of the Go
Agent.

Please pay close attention to the list of Changes.

### Changes

* A full list of changes and a step by step checklist on how to upgrade can
  be found in the [v3 Migration Guide](MIGRATION.md).

### New Features

* Support for Go Modules.  Our Go agent integration packages support frameworks
  and libraries which are changing over time. With support for Go Modules, we
  are now able to release instrumentation packages for multiple versions of
  frameworks and libraries with a single agent release; and support operation
  of the Go agent in Go Modules environments.   This affects naming of our
  integration packages, as described in the v3 Migration Guide (see under
  "Changes" above).

* Detect and set hostnames based on Heroku dyno names.  When deploying an
  application in Heroku, the hostnames collected will now match the dyno name.
  This serves to greatly improve the usability of the servers list in APM since
  dyno names are often sporadic or fleeting in nature.  The feature is
  controlled by two new configuration options `Config.Heroku.UseDynoNames` and
  `Config.Heroku.DynoNamePrefixesToShorten`.

## 2.16.3

### New Relic's Go agent v3.0 is currently available for review and beta testing.  Your use of this pre-release is at your own risk. New Relic disclaims all warranties, express or implied, regarding the beta release.

### If you do not manually take steps to use the new v3 folder you will not see any changes in your agent.

This is the third release of the pre-release of Go agent v3.0.  It includes
changes due to user feedback during the pre-release. The existing agent in
`"github.com/newrelic/go-agent"` is unchanged.  The Go agent v3.0 code in the v3
folder has the following changes:

* A [ConfigFromEnvironment](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigFromEnvironment)
  bug has been fixed.

## 2.16.2

### New Relic's Go agent v3.0 is currently available for review and beta testing. Your use of this pre-release is at your own risk. New Relic disclaims all warranties, express or implied, regarding the beta release.

### If you do not manually take steps to use the new v3 folder, as described below, you will not see any changes in your agent.

This is the second release of the pre-release of Go agent v3.0.  It includes changes due to user feedback during the pre-release. The existing
agent in `"github.com/newrelic/go-agent"` is unchanged.  The Go agent v3.0 code
in the v3 folder has the following changes:

* Transaction names created by [`WrapHandle`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#WrapHandle),
[`WrapHandleFunc`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#WrapHandleFunc),
[nrecho-v3](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrecho-v3),
[nrecho-v4](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrecho-v4),
[nrgorilla](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgorilla), and
[nrgin](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgin) now
include the HTTP method.  For example, the following code:

  ```go
  http.HandleFunc(newrelic.WrapHandleFunc(app, "/users", usersHandler))
  ```

  now creates a metric called `WebTransaction/Go/GET /users` instead of
  `WebTransaction/Go/users`.  As a result of this change, you may need to update
  your alerts and dashboards.

* The [ConfigFromEnvironment](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigFromEnvironment)
  config option is now strict.  If one of the environment variables, such as
  `NEW_RELIC_DISTRIBUTED_TRACING_ENABLED`, cannot be parsed, then `Config.Error`
  will be populated and [NewApplication](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewApplication)
  will return an error.

* [ConfigFromEnvironment](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigFromEnvironment)
  now processes `NEW_RELIC_ATTRIBUTES_EXCLUDE` and `NEW_RELIC_ATTRIBUTES_INCLUDE`.

## 2.16.1

### New Relic's Go agent v3.0 is currently available for review and beta testing. Your use of this pre-release is at your own risk. New Relic disclaims all warranties, express or implied, regarding the beta release.

### If you do not manually take steps to use the new v3 folder, as described below, you will not see any changes in your agent.

This 2.16.1 release includes a new v3.0 folder which contains the pre-release of
Go agent v3.0; Go agent v3.0 includes breaking changes. We are seeking
feedback and hope that you will look this over and test out the changes prior
to the official release.

**This is not an official 3.0 release, it is just a vehicle to gather feedback
on proposed changes**. It is not tagged as 3.0 in Github and the 3.0 release is
not yet available to update in your Go mod file. In order to test out these
changes, you will need to clone this repo in your Go source directory, under
`[go-src-dir]/src/github.com/newrelic/go-agent`. Once you have the source
checked out, you will need to follow the steps in the second section of
[v3/MIGRATION.md](v3/MIGRATION.md).

A list of changes and installation instructions is included in the v3 folder
and can be found [here](v3/MIGRATION.md)

For this pre-release (beta) version of Go agent v3.0, please note:
* The changes in the v3 folder represent what we expect to release in ~2 weeks
as our major 3.0 release. However, as we are soliciting feedback on the changes
and there is the possibility of some breaking changes before the official
release.
* This is not an official 3.0 release; it is not tagged as 3.0 in Github and
the 3.0 release is not yet available to update in your Go mod file.
* If you test out these changes and encounter issues, questions, or have
feedback that you would like to pass along, please open up an issue
[here](https://github.com/newrelic/go-agent/issues/new) and be sure to include
the label `3.0`.
  * For normal (non-3.0) issues/questions we request that you report them via
   our [support site](http://support.newrelic.com/) or our
   [community forum](https://discuss.newrelic.com). Please only report
   questions related to the 3.0 pre-release directly via GitHub.


### New Features

* V3 will add support for Go Modules. The go.mod files exist in the v3 folder,
but they will not be usable until we have fully tagged the 3.0 release
officially. Examples of version tags we plan to use for different modules
include:
  * `v3.0.0`
  * `v3/integrations/nrecho-v3/v1.0.0`
  * `v3/integrations/nrecho-v4/v1.0.0`

### Changes

* The changes are the ones that we have requested feedback previously in
[this issue](https://github.com/newrelic/go-agent/issues/106).  
* A full list of changes that are included, along with a checklist for
 upgrading, is available in [v3/MIGRATION.md](v3/MIGRATION.md).

## 2.16.0

### Upcoming

* The next release of the Go Agent is expected to be a major version release
  to improve the API and incorporate Go modules.
  Details available here: https://github.com/newrelic/go-agent/issues/106
  We would love your feedback!

### Bug Fixes

* Fixed an issue in the
  [`nrhttprouter`](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrhttprouter)
  integration where the transaction was not being added to the requests
  context.  This resulted in an inability to access the transaction from within
  an
  [`httprouter.Handle`](https://godoc.org/github.com/julienschmidt/httprouter#Handle)
  function.  This issue has now been fixed.

## 2.15.0

### New Features

* Added support for monitoring [MongoDB](https://github.com/mongodb/mongo-go-driver/) queries with the new
[_integrations/nrmongo](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmongo)
package.

  * [Example application](https://github.com/newrelic/go-agent/blob/master/_integrations/nrmongo/example/main.go)
  * [Full godocs Documentation](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmongo)

* Added new method `Transaction.IsSampled()` that returns a boolean that
  indicates if the transaction is sampled.  A sampled transaction records a
  span event for each segment.  Distributed tracing must be enabled for
  transactions to be sampled.  `false` is returned if the transaction has
  finished.  This sampling flag is needed for B3 trace propagation and
  future support of W3C Trace Context.

* Added support for adding [B3
  Headers](https://github.com/openzipkin/b3-propagation) to outgoing requests.
  This is helpful if the service you are calling uses B3 for trace state
  propagation (for example, it uses Zipkin instrumentation).  You can use the
  new
  [_integrations/nrb3](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrb3)
  package's
  [`nrb3.NewRoundTripper`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrb3#NewRoundTripper)
  like this:

  ```go
  // When defining the client, set the Transport to the NewRoundTripper. This
  // will create ExternalSegments and add B3 headers for each request.
  client := &http.Client{
      Transport: nrb3.NewRoundTripper(nil),
  }

  // Distributed Tracing must be enabled for this application.
  // (see https://docs.newrelic.com/docs/understand-dependencies/distributed-tracing/enable-configure/enable-distributed-tracing)
  txn := currentTxn()

  req, err := http.NewRequest("GET", "http://example.com", nil)
  if nil != err {
      log.Fatalln(err)
  }

  // Be sure to add the transaction to the request context.  This step is
  // required.
  req = newrelic.RequestWithTransactionContext(req, txn)
  resp, err := client.Do(req)
  if nil != err {
      log.Fatalln(err)
  }

  defer resp.Body.Close()
  fmt.Println(resp.StatusCode)
  ```

### Bug Fixes

* Fixed an issue where the
  [`nrgin`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrgin/v1)
  integration was not capturing the correct response code in the case where no
  response body was sent.  This issue has now been fixed but requires Gin
  greater than v1.4.0.

## 2.14.1

### Bug Fixes

* Removed the hidden `"NEW_RELIC_DEBUG_LOGGING"` environment variable setting
  which was broken in release 2.14.0.

## 2.14.0

### New Features

* Added support for a new segment type,
  [`MessageProducerSegment`](https://godoc.org/github.com/newrelic/go-agent#MessageProducerSegment),
  to be used to track time spent adding messages to message queuing systems like
  RabbitMQ or Kafka.

  ```go
  seg := &newrelic.MessageProducerSegment{
      StartTime:       newrelic.StartSegmentNow(txn),
      Library:         "RabbitMQ",
      DestinationType: newrelic.MessageExchange,
      DestinationName: "myExchange",
  }
  // add message to queue here
  seg.End()
  ```

* Added new attribute constants for use with message consumer transactions.
  These attributes can be used to add more detail to a transaction that tracks
  time spent consuming a message off a message queuing system like RabbitMQ or Kafka.
  They can be added using
  [`txn.AddAttribute`](https://godoc.org/github.com/newrelic/go-agent#Transaction).

  ```go
  // The routing key of the consumed message.
  txn.AddAttribute(newrelic.AttributeMessageRoutingKey, "myRoutingKey")
  // The name of the queue the message was consumed from.
  txn.AddAttribute(newrelic.AttributeMessageQueueName, "myQueueName")
  // The type of exchange used for the consumed message (direct, fanout,
  // topic, or headers).
  txn.AddAttribute(newrelic.AttributeMessageExchangeType, "myExchangeType")
  // The callback queue used in RPC configurations.
  txn.AddAttribute(newrelic.AttributeMessageReplyTo, "myReplyTo")
  // The application-generated identifier used in RPC configurations.
  txn.AddAttribute(newrelic.AttributeMessageCorrelationID, "myCorrelationID")
  ```

  It is recommended that at most one message is consumed per transaction.

* Added support for [Go 1.13's Error wrapping](https://golang.org/doc/go1.13#error_wrapping).
  `Transaction.NoticeError` now uses [Unwrap](https://golang.org/pkg/errors/#Unwrap)
  recursively to identify the error's cause (the deepest wrapped error) when generating
  the error's class field.  This functionality will help group your errors usefully.

  For example, when using Go 1.13, the following code:

  ```go
  type socketError struct{}

  func (e socketError) Error() string { return "socket error" }

  func gamma() error { return socketError{} }
  func beta() error  { return fmt.Errorf("problem in beta: %w", gamma()) }
  func alpha() error { return fmt.Errorf("problem in alpha: %w", beta()) }

  func execute(txn newrelic.Transaction) {
  	err := alpha()
  	txn.NoticeError(err)
  }
  ```
  captures an error with message `"problem in alpha: problem in beta: socket error"`
  and class `"main.socketError"`.  Previously, the class was recorded as `"*fmt.wrapError"`.

* A `Stack` field has been added to [Error](https://godoc.org/github.com/newrelic/go-agent#Error),
  which can be assigned using the new
  [NewStackTrace](https://godoc.org/github.com/newrelic/go-agent#NewStackTrace) function.
  This allows your error stack trace to show where the error happened, rather
  than the location of the `NoticeError` call.

  `Transaction.NoticeError` not only checks for a stack trace (using
  [StackTracer](https://godoc.org/github.com/newrelic/go-agent#StackTracer)) in
  the error parameter, but in the error's cause as well.  This means that you
  can create an [Error](https://godoc.org/github.com/newrelic/go-agent#Error)
  where your error occurred, wrap it multiple times to add information, notice it
  with `NoticeError`, and still have a useful stack trace. Take a look!

  ```go
  func gamma() error {
  	return newrelic.Error{
  		Message: "something went very wrong",
  		Class:   "socketError",
  		Stack:   newrelic.NewStackTrace(),
  	}
  }

  func beta() error  { return fmt.Errorf("problem in beta: %w", gamma()) }
  func alpha() error { return fmt.Errorf("problem in alpha: %w", beta()) }

  func execute(txn newrelic.Transaction) {
  	err := alpha()
  	txn.NoticeError(err)
  }
  ```

  In this example, the topmost stack trace frame recorded is `"gamma"`,
  rather than `"execute"`.

* Added support for configuring a maximum number of transaction events per minute to be sent to New Relic.
It can be configured as follows:

  ```go
  config := newrelic.NewConfig("Application Name", os.Getenv("NEW_RELIC_LICENSE_KEY"))  
  config.TransactionEvents.MaxSamplesStored = 100
  ```
    * For additional configuration information, see our [documentation](https://docs.newrelic.com/docs/agents/go-agent/configuration/go-agent-configuration)


### Miscellaneous

* Updated the
  [`nrmicro`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmicro)
  package to use the new segment type
  [`MessageProducerSegment`](https://godoc.org/github.com/newrelic/go-agent#MessageProducerSegment)
  and the new attribute constants:
  * [`nrmicro.ClientWrapper`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmicro#ClientWrapper)
    now uses `newrelic.MessageProducerSegment`s instead of
    `newrelic.ExternalSegment`s for calls to
    [`Client.Publish`](https://godoc.org/github.com/micro/go-micro/client#Client).
  * [`nrmicro.SubscriberWrapper`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmicro#SubscriberWrapper)
    updates transaction names and adds the attribute `message.routingKey`.

* Updated the
  [`nrnats`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrnats)
  and
  [`nrstan`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrstan)
  packages to use the new segment type
  [`MessageProducerSegment`](https://godoc.org/github.com/newrelic/go-agent#MessageProducerSegment)
  and the new attribute constants:
  * [`nrnats.StartPublishSegment`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrnats#StartPublishSegment)
    now starts and returns a `newrelic.MessageProducerSegment` type.
  * [`nrnats.SubWrapper`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrnats#SubWrapper)
    and
    [`nrstan.StreamingSubWrapper`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrstan#StreamingSubWrapper)
    updates transaction names and adds the attributes `message.routingKey`,
    `message.queueName`, and `message.replyTo`.

## 2.13.0

### New Features

* Added support for [HttpRouter](https://github.com/julienschmidt/httprouter) in
  the new [_integrations/nrhttprouter](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrhttprouter) package.  This package allows you to easily instrument inbound requests through the HttpRouter framework.

  * [Documentation](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrhttprouter)
  * [Example](_integrations/nrhttprouter/example/main.go)

* Added support for [github.com/uber-go/zap](https://github.com/uber-go/zap) in
  the new
  [_integrations/nrzap](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrzap)
  package.  This package allows you to send agent log messages to `zap`.

## 2.12.0

### New Features

* Added new methods to expose `Transaction` details:

  * `Transaction.GetTraceMetadata()` returns a
    [TraceMetadata](https://godoc.org/github.com/newrelic/go-agent#TraceMetadata)
    which contains distributed tracing identifiers.

  * `Transaction.GetLinkingMetadata()` returns a
    [LinkingMetadata](https://godoc.org/github.com/newrelic/go-agent#LinkingMetadata)
    which contains the fields needed to link data to a trace or entity.

* Added a new plugin for the [Logrus logging
  framework](https://github.com/sirupsen/logrus) with the new
  [_integrations/logcontext/nrlogrusplugin](https://github.com/newrelic/go-agent/go-agent/tree/master/_integrations/logcontext/nrlogrusplugin)
  package. This plugin leverages the new `GetTraceMetadata` and
  `GetLinkingMetadata` above to decorate logs.

  To enable, set your log's formatter to the `nrlogrusplugin.ContextFormatter{}`

  ```go
  logger := logrus.New()
  logger.SetFormatter(nrlogrusplugin.ContextFormatter{})
  ```

  The logger will now look for a `newrelic.Transaction` inside its context and
  decorate logs accordingly.  Therefore, the Transaction must be added to the
  context and passed to the logger.  For example, this logging call

  ```go
  logger.Info("Hello New Relic!")
  ```

  must be transformed to include the context, such as:

  ```go
  ctx := newrelic.NewContext(context.Background(), txn)
  logger.WithContext(ctx).Info("Hello New Relic!")
  ```

  For full documentation see the
  [godocs](https://godoc.org/github.com/newrelic/go-agent/_integrations/logcontext/nrlogrusplugin)
  or view the
  [example](https://github.com/newrelic/go-agent/blob/master/_integrations/logcontext/nrlogrusplugin/example/main.go).

* Added support for [NATS](https://github.com/nats-io/nats.go) and [NATS Streaming](https://github.com/nats-io/stan.go)
monitoring with the new [_integrations/nrnats](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrnats) and
[_integrations/nrstan](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrstan) packages.  These packages
support instrumentation of publishers and subscribers.

  * [NATS Example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrnats/examples/main.go)
  * [NATS Streaming Example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrstan/examples/main.go)

* Enables ability to migrate to [Configurable Security Policies (CSP)](https://docs.newrelic.com/docs/agents/manage-apm-agents/configuration/enable-configurable-security-policies) on a per agent basis for accounts already using [High Security Mode (HSM)](https://docs.newrelic.com/docs/agents/manage-apm-agents/configuration/high-security-mode).
  * Previously, if CSP was configured for an account, New Relic would not allow an agent to connect without the `security_policies_token`. This led to agents not being able to connect during the period between when CSP was enabled for an account and when each agent is configured with the correct token.
  * With this change, when both HSM and CSP are enabled for an account, an agent (this version or later) can successfully connect with either `high_security: true` or the appropriate `security_policies_token` configured - allowing the agent to continue to connect after CSP is configured on the account but before the appropriate `security_policies_token` is configured for each agent.

## 2.11.0

### New Features

* Added support for [Micro](https://github.com/micro/go-micro) monitoring with the new
[_integrations/nrmicro](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmicro)
package.  This package supports instrumentation for servers, clients, publishers, and subscribers.

  * [Server Example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrmicro/example/server/server.go)
  * [Client Example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrmicro/example/client/client.go)
  * [Publisher and Subscriber Example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrmicro/example/pubsub/main.go)
  * [Full godocs Documentation](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmicro)

* Added support for creating static `WebRequest` instances manually via the `NewStaticWebRequest` function. This can be useful when you want to create a web transaction but don't have an `http.Request` object. Here's an example of creating a static `WebRequest` and using it to mark a transaction as a web transaction:
  ```go
  hdrs := http.Headers{}
  u, _ := url.Parse("http://example.com")
  webReq := newrelic.NewStaticWebRequest(hdrs, u, "GET", newrelic.TransportHTTP)
  txn := app.StartTransaction("My-Transaction", nil, nil)
  txn.SetWebRequest(webReq)
  ```

## 2.10.0

### New Features

* Added support for custom events when using
  [nrlambda](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrlambda).
  Example Lambda handler which creates custom event:

   ```go
   func handler(ctx context.Context) {
		if txn := newrelic.FromContext(ctx); nil != txn {
			txn.Application().RecordCustomEvent("myEvent", map[string]interface{}{
				"zip": "zap",
			})
		}
		fmt.Println("hello world!")
   }
   ```

## 2.9.0

### New Features

* Added support for [gRPC](https://github.com/grpc/grpc-go) monitoring with the new
[_integrations/nrgrpc](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrgrpc)
package.  This package supports instrumentation for servers and clients.

  * [Server Example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrgrpc/example/server/server.go)
  * [Client Example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrgrpc/example/client/client.go)

* Added new
  [ExternalSegment](https://godoc.org/github.com/newrelic/go-agent#ExternalSegment)
  fields `Host`, `Procedure`, and `Library`.  These optional fields are
  automatically populated from the segment's `URL` or `Request` if unset.  Use
  them if you don't have access to a request or URL but still want useful external
  metrics, transaction segment attributes, and span attributes.
  * `Host` is used for external metrics, transaction trace segment names, and
    span event names.  The host of segment's `Request` or `URL` is the default.
  * `Procedure` is used for transaction breakdown metrics.  If set, it should be
    set to the remote procedure being called.  The HTTP method of the segment's `Request` is the default.
  * `Library` is used for external metrics and the `"component"` span attribute.
    If set, it should be set to the framework making the call. `"http"` is the default.

  With the addition of these new fields, external transaction breakdown metrics
  are changed: `External/myhost.com/all` will now report as
  `External/myhost.com/http/GET` (provided the HTTP method is `GET`).

* HTTP Response codes below `100`, except `0` and `5`, are now recorded as
  errors.  This is to support `gRPC` status codes.  If you start seeing
  new status code errors that you would like to ignore, add them to
  `Config.ErrorCollector.IgnoreStatusCodes` or your server side configuration
  settings.

* Improve [logrus](https://github.com/sirupsen/logrus) support by introducing
  [nrlogrus.Transform](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrlogrus#Transform),
  a function which allows you to turn a
  [logrus.Logger](https://godoc.org/github.com/sirupsen/logrus#Logger) instance into a
  [newrelic.Logger](https://godoc.org/github.com/newrelic/go-agent#Logger).
  Example use:

  ```go
  l := logrus.New()
  l.SetLevel(logrus.DebugLevel)
  cfg := newrelic.NewConfig("Your Application Name", "__YOUR_NEW_RELIC_LICENSE_KEY__")
  cfg.Logger = nrlogrus.Transform(l)
  ```

  As a result of this change, the
  [nrlogrus](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrlogrus)
  package requires [logrus](https://github.com/sirupsen/logrus) version `v1.1.0`
  and above.

## 2.8.1

### Bug Fixes

* Removed `nrmysql.NewConnector` since
  [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) has not yet
  released `mysql.NewConnector`.

## 2.8.0

### New Features

* Support for Real Time Streaming

  * The agent now has support for sending event data to New Relic every five
    seconds, instead of every minute.  As a result, transaction, error, and
    custom events will now be available in New Relic One and Insights dashboards
    in near real time. For more information on how to view your events with a
    five-second refresh, see the documentation.

  * Note that the overall limits on how many events can be sent per minute have
    not changed. Also, span events, metrics, and trace data is unaffected, and
    will still be sent every minute.

* Introduce support for databases using
  [database/sql](https://golang.org/pkg/database/sql/).  This new functionality
  allows you to instrument MySQL, PostgreSQL, and SQLite calls without manually
  creating
  [DatastoreSegment](https://godoc.org/github.com/newrelic/go-agent#DatastoreSegment)s.

  | Database Library Supported | Integration Package |
  | ------------- | ------------- |
  | [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) | [_integrations/nrmysql](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmysql) |
  | [lib/pq](https://github.com/lib/pq) | [_integrations/nrpq](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrpq) |
  | [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) | [_integrations/nrsqlite3](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrsqlite3) |

  Using these database integration packages is easy!  First replace the driver
  with our integration version:

  ```go
  import (
  	// import our integration package in place of "github.com/go-sql-driver/mysql"
  	_ "github.com/newrelic/go-agent/_integrations/nrmysql"
  )

  func main() {
  	// open "nrmysql" in place of "mysql"
  	db, err := sql.Open("nrmysql", "user@unix(/path/to/socket)/dbname")
  }
  ```

  Second, use the `ExecContext`, `QueryContext`, and `QueryRowContext` methods of
  [sql.DB](https://golang.org/pkg/database/sql/#DB),
  [sql.Conn](https://golang.org/pkg/database/sql/#Conn),
  [sql.Tx](https://golang.org/pkg/database/sql/#Tx), and
  [sql.Stmt](https://golang.org/pkg/database/sql/#Stmt) and provide a
  transaction-containing context.  Calls to `Exec`, `Query`, and `QueryRow` do not
  get instrumented.

  ```go
  ctx := newrelic.NewContext(context.Background(), txn)
  row := db.QueryRowContext(ctx, "SELECT count(*) from tables")
  ```

  If you are using a [database/sql](https://golang.org/pkg/database/sql/) database
  not listed above, you can write your own instrumentation for it using
  [InstrumentSQLConnector](https://godoc.org/github.com/newrelic/go-agent#InstrumentSQLConnector),
  [InstrumentSQLDriver](https://godoc.org/github.com/newrelic/go-agent#InstrumentSQLDriver),
  and
  [SQLDriverSegmentBuilder](https://godoc.org/github.com/newrelic/go-agent#SQLDriverSegmentBuilder).
  The integration packages act as examples of how to do this.

  For more information, see the [Go agent documentation on instrumenting datastore segments](https://docs.newrelic.com/docs/agents/go-agent/instrumentation/instrument-go-segments#go-datastore-segments).

### Bug Fixes

* The [http.RoundTripper](https://golang.org/pkg/net/http/#RoundTripper) returned
  by [NewRoundTripper](https://godoc.org/github.com/newrelic/go-agent#NewRoundTripper)
  no longer modifies the request.  Our thanks to @jlordiales for the contribution.

## 2.7.0

### New Features

* Added support for server side configuration.  Server side configuration allows
 you to set the following configuration settings in the New Relic APM UI:

  * `Config.TransactionTracer.Enabled`
  * `Config.ErrorCollector.Enabled`
  * `Config.CrossApplicationTracer.Enabled`
  * `Config.TransactionTracer.Threshold`
  * `Config.TransactionTracer.StackTraceThreshold`
  * `Config.ErrorCollector.IgnoreStatusCodes`

  For more information see the [server side configuration documentation](https://docs.newrelic.com/docs/agents/manage-apm-agents/configuration/server-side-agent-configuration).

* Added support for AWS Lambda functions in the new
  [nrlambda](_integrations/nrlambda)
  package.  Please email <lambda_preview@newrelic.com> if you are interested in
  learning more or previewing New Relic Lambda monitoring.  This instrumentation
  package requires `aws-lambda-go` version
  [v1.9.0](https://github.com/aws/aws-lambda-go/releases) and above.

  * [documentation](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrlambda)
  * [working example](_integrations/nrlambda/example/main.go)

## 2.6.0

### New Features

* Added support for async: the ability to instrument multiple concurrent
  goroutines, or goroutines that access or manipulate the same Transaction.

  The new `Transaction.NewGoroutine() Transaction` method allows
  transactions to create segments in multiple goroutines!

  `NewGoroutine` returns a new reference to the `Transaction`.  This must be
  called any time you are passing the `Transaction` to another goroutine which
  makes segments.  Each segment-creating goroutine must have its own `Transaction`
  reference.  It does not matter if you call this before or after the other
  goroutine has started.

  All `Transaction` methods can be used in any `Transaction` reference.  The
  `Transaction` will end when `End()` is called in any goroutine.

  Example passing a new `Transaction` reference directly to another goroutine:

  ```go
  	go func(txn newrelic.Transaction) {
  		defer newrelic.StartSegment(txn, "async").End()
  		time.Sleep(100 * time.Millisecond)
  	}(txn.NewGoroutine())
  ```

  Example passing a new `Transaction` reference on a channel to another
  goroutine:

  ```go
  	ch := make(chan newrelic.Transaction)
  	go func() {
  		txn := <-ch
  		defer newrelic.StartSegment(txn, "async").End()
  		time.Sleep(100 * time.Millisecond)
  	}()
  	ch <- txn.NewGoroutine()
  ```

* Added integration support for
  [`aws-sdk-go`](https://github.com/aws/aws-sdk-go) and
  [`aws-sdk-go-v2`](https://github.com/aws/aws-sdk-go-v2).

  When using these SDKs, a segment will be created for each out going request.
  For DynamoDB calls, these will be Datastore segments and for all others they
  will be External segments.
  * [v1 Documentation](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrawssdk/v1)
  * [v2 Documentation](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrawssdk/v2)

* Added span event and transaction trace segment attribute configuration.  You
  may control which attributes are captured in span events and transaction trace
  segments using the `Config.SpanEvents.Attributes` and
  `Config.TransactionTracer.Segments.Attributes` settings. For example, if you
  want to disable the collection of `"db.statement"` in your span events, modify
  your config like this:

  ```go
  cfg.SpanEvents.Attributes.Exclude = append(cfg.SpanEvents.Attributes.Exclude,
  	newrelic.SpanAttributeDBStatement)
  ```

  To disable the collection of all attributes from your transaction trace
  segments, modify your config like this:

  ```go
  cfg.TransactionTracer.Segments.Attributes.Enabled = false
  ```

### Bug Fixes

* Fixed a bug that would prevent External Segments from being created under
  certain error conditions related to Cross Application Tracing.

### Miscellaneous

* Improved linking between Cross Application Transaction Traces in the APM UI.
  When `Config.CrossApplicationTracer.Enabled = true`, External segments in the
  Transaction Traces details will now link to the downstream Transaction Trace
  if there is one. Additionally, the segment name will now include the name of
  the downstream application and the name of the downstream transaction.

* Update attribute names of Datastore and External segments on Transaction
  Traces to be in line with attribute names on Spans. Specifically:
    * `"uri"` => `"http.url"`
    * `"query"` => `"db.statement"`
    * `"database_name"` => `"db.instance"`
    * `"host"` => `"peer.hostname"`
    * `"port_path_or_id"` + `"host"` => `"peer.address"`

## 2.5.0

* Added support for [New Relic Browser](https://docs.newrelic.com/docs/browser)
  using the new `BrowserTimingHeader` method on the
  [`Transaction`](https://godoc.org/github.com/newrelic/go-agent#Transaction)
  which returns a
  [BrowserTimingHeader](https://godoc.org/github.com/newrelic/go-agent#BrowserTimingHeader).
  The New Relic Browser JavaScript code measures page load timing, also known as
  real user monitoring.  The Pro version of this feature measures AJAX requests,
  single-page applications, JavaScript errors, and much more!  Example use:

```go
func browser(w http.ResponseWriter, r *http.Request) {
	hdr, err := w.(newrelic.Transaction).BrowserTimingHeader()
	if nil != err {
		log.Printf("unable to create browser timing header: %v", err)
	}
	// BrowserTimingHeader() will always return a header whose methods can
	// be safely called.
	if js := hdr.WithTags(); js != nil {
		w.Write(js)
	}
	io.WriteString(w, "browser header page")
}
```

* The Go agent now collects an attribute named `request.uri` on Transaction
  Traces, Transaction Events, Error Traces, and Error Events.  `request.uri`
  will never contain user, password, query parameters, or fragment.  To prevent
  the request's URL from being collected in any data, modify your `Config` like
  this:

```go
cfg.Attributes.Exclude = append(cfg.Attributes.Exclude, newrelic.AttributeRequestURI)
```

## 2.4.0

* Introduced `Transaction.Application` method which returns the `Application`
  that started the `Transaction`.  This method is useful since it may prevent
  having to pass the `Application` to code that already has access to the
  `Transaction`.  Example use:

```go
txn.Application().RecordCustomEvent("customerOrder", map[string]interface{}{
	"numItems":   2,
	"totalPrice": 13.75,
})
```

* The `Transaction.AddAttribute` method no longer accepts `nil` values since
  our backend ignores them.

## 2.3.0

* Added support for [Echo](https://echo.labstack.com) in the new `nrecho`
  package.
  * [Documentation](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrecho)
  * [Example](_integrations/nrecho/example/main.go)

* Introduced `Transaction.SetWebResponse(http.ResponseWriter)` method which sets
  the transaction's response writer.  After calling this method, the
  `Transaction` may be used in place of the `http.ResponseWriter` to intercept
  the response code.  This method is useful when the `http.ResponseWriter` is
  not available at the beginning of the transaction (if so, it can be given as a
  parameter to `Application.StartTransaction`).  This method will return a
  reference to the transaction which implements the combination of
  `http.CloseNotifier`, `http.Flusher`, `http.Hijacker`, and `io.ReaderFrom`
  implemented by the ResponseWriter.  Example:

```go
func setResponseDemo(txn newrelic.Transaction) {
	recorder := httptest.NewRecorder()
	txn = txn.SetWebResponse(recorder)
	txn.WriteHeader(200)
	fmt.Println("response code recorded:", recorder.Code)
}
```

* The `Transaction`'s `http.ResponseWriter` methods may now be called safely if
  a `http.ResponseWriter` has not been set.  This allows you to add a response code
  to the transaction without using a `http.ResponseWriter`.  Example:

```go
func transactionWithResponseCode(app newrelic.Application) {
       txn := app.StartTransaction("hasResponseCode", nil, nil)
       defer txn.End()
       txn.WriteHeader(200) // Safe!
}
```

* The agent now collects environment variables prefixed by
  `NEW_RELIC_METADATA_`.  Some of these may be added
  Transaction events to provide context between your Kubernetes cluster and your
  services. For details on the benefits (currently in beta) see [this blog
  post](https://blog.newrelic.com/engineering/monitoring-application-performance-in-kubernetes/)

* The agent now collects the `KUBERNETES_SERVICE_HOST` environment variable to
  detect when the application is running on Kubernetes.

* The agent now collects the fully qualified domain name of the host and
  local IP addresses for improved linking with our infrastructure product.

## 2.2.0

* The `Transaction` parameter to
[NewRoundTripper](https://godoc.org/github.com/newrelic/go-agent#NewRoundTripper)
and
[StartExternalSegment](https://godoc.org/github.com/newrelic/go-agent#StartExternalSegment)
is now optional:  If it is `nil`, then a `Transaction` will be looked for in the
request's context (using
[FromContext](https://godoc.org/github.com/newrelic/go-agent#FromContext)).
Passing a `nil` transaction is **STRONGLY** recommended when using
[NewRoundTripper](https://godoc.org/github.com/newrelic/go-agent#NewRoundTripper)
since it allows one `http.Client.Transport` to be used for multiple
transactions.  Example use:

```go
client := &http.Client{}
client.Transport = newrelic.NewRoundTripper(nil, client.Transport)
request, _ := http.NewRequest("GET", "http://example.com", nil)
request = newrelic.RequestWithTransactionContext(request, txn)
resp, err := client.Do(request)
```

* Introduced `Transaction.SetWebRequest(WebRequest)` method which marks the
transaction as a web transaction.  If the `WebRequest` parameter is non-nil,
`SetWebRequest` will collect details on request attributes, url, and method.
This method is useful if you don't have access to the request at the beginning
of the transaction, or if your request is not an `*http.Request` (just add
methods to your request that satisfy
[WebRequest](https://godoc.org/github.com/newrelic/go-agent#WebRequest)).  To
use an `*http.Request` as the parameter, use the
[NewWebRequest](https://godoc.org/github.com/newrelic/go-agent#NewWebRequest)
transformation function.  Example:

```go
var request *http.Request = getInboundRequest()
txn.SetWebRequest(newrelic.NewWebRequest(request))
```

* Fixed `Debug` in `nrlogrus` package.  Previous versions of the New Relic Go Agent incorrectly
logged to Info level instead of Debug.  This has now been fixed.  Thanks to @paddycarey for catching this.

* [nrgin.Transaction](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrgin/v1#Transaction)
may now be called with either a `context.Context` or a `*gin.Context`.  If you were passing a `*gin.Context`
around your functions as a `context.Context`, you may access the Transaction by calling either
[nrgin.Transaction](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrgin/v1#Transaction)
or [FromContext](https://godoc.org/github.com/newrelic/go-agent#FromContext).
These functions now work nicely together.
For example, [FromContext](https://godoc.org/github.com/newrelic/go-agent#FromContext) will return the `Transaction`
added by [nrgin.Middleware](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrgin/v1#Middleware).
Thanks to @rodriguezgustavo for the suggestion.  

## 2.1.0

* The Go Agent now supports distributed tracing.

  Distributed tracing lets you see the path that a request takes as it travels through your distributed system. By
  showing the distributed activity through a unified view, you can troubleshoot and understand a complex system better
  than ever before.

  Distributed tracing is available with an APM Pro or equivalent subscription. To see a complete distributed trace, you
  need to enable the feature on a set of neighboring services. Enabling distributed tracing changes the behavior of
  some New Relic features, so carefully consult the
  [transition guide](https://docs.newrelic.com/docs/transition-guide-distributed-tracing) before you enable this
  feature.

  To enable distributed tracing, set the following fields in your config.  Note that distributed tracing and cross
  application tracing cannot be used simultaneously.

```
  config := newrelic.NewConfig("Your Application Name", "__YOUR_NEW_RELIC_LICENSE_KEY__")
  config.CrossApplicationTracer.Enabled = false
  config.DistributedTracer.Enabled = true
```

  Please refer to the
  [distributed tracing section of the guide](GUIDE.md#distributed-tracing)
  for more detail on how to ensure you get the most out of the Go agent's distributed tracing support.

* Added functions [NewContext](https://godoc.org/github.com/newrelic/go-agent#NewContext)
  and [FromContext](https://godoc.org/github.com/newrelic/go-agent#FromContext)
  for adding and retrieving the Transaction from a Context.  Handlers
  instrumented by
  [WrapHandle](https://godoc.org/github.com/newrelic/go-agent#WrapHandle),
  [WrapHandleFunc](https://godoc.org/github.com/newrelic/go-agent#WrapHandleFunc),
  and [nrgorilla.InstrumentRoutes](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrgorilla/v1#InstrumentRoutes)
  may use [FromContext](https://godoc.org/github.com/newrelic/go-agent#FromContext)
  on the request's context to access the Transaction.
  Thanks to @caarlos0 for the contribution!  Though [NewContext](https://godoc.org/github.com/newrelic/go-agent#NewContext)
  and [FromContext](https://godoc.org/github.com/newrelic/go-agent#FromContext)
  require Go 1.7+ (when [context](https://golang.org/pkg/context/) was added),
  [RequestWithTransactionContext](https://godoc.org/github.com/newrelic/go-agent#RequestWithTransactionContext) is always exported so that it can be used in all framework and library
  instrumentation.

## 2.0.0

* The `End()` functions defined on the `Segment`, `DatastoreSegment`, and
  `ExternalSegment` types now receive the segment as a pointer, rather than as
  a value. This prevents unexpected behaviour when a call to `End()` is
  deferred before one or more fields are changed on the segment.

  In practice, this is likely to only affect this pattern:

    ```go
    defer newrelic.DatastoreSegment{
      // ...
    }.End()
    ```

  Instead, you will now need to separate the literal from the deferred call:

    ```go
    ds := newrelic.DatastoreSegment{
      // ...
    }
    defer ds.End()
    ```

  When creating custom and external segments, we recommend using
  [`newrelic.StartSegment()`](https://godoc.org/github.com/newrelic/go-agent#StartSegment)
  and
  [`newrelic.StartExternalSegment()`](https://godoc.org/github.com/newrelic/go-agent#StartExternalSegment),
  respectively.

* Added GoDoc badge to README.  Thanks to @mrhwick for the contribution!

* `Config.UseTLS` configuration setting has been removed to increase security.
   TLS will now always be used in communication with New Relic Servers.

## 1.11.0

* We've closed the Issues tab on GitHub. Please visit our
  [support site](https://support.newrelic.com) to get timely help with any
  problems you're having, or to report issues.

* Added support for Cross Application Tracing (CAT). Please refer to the
  [CAT section of the guide](GUIDE.md#cross-application-tracing)
  for more detail on how to ensure you get the most out of the Go agent's new
  CAT support.

* The agent now collects additional metadata when running within Amazon Web
  Services, Google Cloud Platform, Microsoft Azure, and Pivotal Cloud Foundry.
  This information is used to provide an enhanced experience when the agent is
  deployed on those platforms.

## 1.10.0

* Added new `RecordCustomMetric` method to [Application](https://godoc.org/github.com/newrelic/go-agent#Application).
  This functionality can be used to track averages or counters without using
  custom events.
  * [Custom Metric Documentation](https://docs.newrelic.com/docs/agents/manage-apm-agents/agent-data/collect-custom-metrics)

* Fixed import needed for logrus.  The import Sirupsen/logrus had been renamed to sirupsen/logrus.
  Thanks to @alfred-landrum for spotting this.

* Added [ErrorAttributer](https://godoc.org/github.com/newrelic/go-agent#ErrorAttributer),
  an optional interface that can be implemented by errors provided to
  `Transaction.NoticeError` to attach additional attributes.  These attributes are
  subject to attribute configuration.

* Added [Error](https://godoc.org/github.com/newrelic/go-agent#Error), a type
  that allows direct control of error fields.  Example use:

```go
txn.NoticeError(newrelic.Error{
	// Message is returned by the Error() method.
	Message: "error message: something went very wrong",
	Class:   "errors are aggregated by class",
	Attributes: map[string]interface{}{
		"important_number": 97232,
		"relevant_string":  "zap",
	},
})
```

* Updated license to address scope of usage.

## 1.9.0

* Added support for [github.com/gin-gonic/gin](https://github.com/gin-gonic/gin)
  in the new `nrgin` package.
  * [Documentation](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrgin/v1)
  * [Example](examples/_gin/main.go)

## 1.8.0

* Fixed incorrect metric rule application when the metric rule is flagged to
  terminate and matches but the name is unchanged.

* `Segment.End()`, `DatastoreSegment.End()`, and `ExternalSegment.End()` methods now return an
  error which may be helpful in diagnosing situations where segment data is unexpectedly missing.

## 1.7.0

* Added support for [gorilla/mux](http://github.com/gorilla/mux) in the new `nrgorilla`
  package.
  * [Documentation](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrgorilla/v1)
  * [Example](examples/_gorilla/main.go)

## 1.6.0

* Added support for custom error messages and stack traces.  Errors provided
  to `Transaction.NoticeError` will now be checked to see if
  they implement [ErrorClasser](https://godoc.org/github.com/newrelic/go-agent#ErrorClasser)
  and/or [StackTracer](https://godoc.org/github.com/newrelic/go-agent#StackTracer).
  Thanks to @fgrosse for this proposal.

* Added support for [pkg/errors](https://github.com/pkg/errors).  Thanks to
  @fgrosse for this work.
  * [documentation](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrpkgerrors)
  * [example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrpkgerrors/nrpkgerrors.go)

* Fixed tests for Go 1.8.

## 1.5.0

* Added support for Windows.  Thanks to @ianomad and @lvxv for the contributions.

* The number of heap objects allocated is recorded in the
  `Memory/Heap/AllocatedObjects` metric.  This will soon be displayed on the "Go
  runtime" page.

* If the [DatastoreSegment](https://godoc.org/github.com/newrelic/go-agent#DatastoreSegment)
  fields `Host` and `PortPathOrID` are not provided, they will no longer appear
  as `"unknown"` in transaction traces and slow query traces.

* Stack traces will now be nicely aligned in the APM UI.

## 1.4.0

* Added support for slow query traces.  Slow datastore segments will now
 generate slow query traces viewable on the datastore tab.  These traces include
 a stack trace and help you to debug slow datastore activity.
 [Slow Query Documentation](https://docs.newrelic.com/docs/apm/applications-menu/monitoring/viewing-slow-query-details)

* Added new
[DatastoreSegment](https://godoc.org/github.com/newrelic/go-agent#DatastoreSegment)
fields `ParameterizedQuery`, `QueryParameters`, `Host`, `PortPathOrID`, and
`DatabaseName`.  These fields will be shown in transaction traces and in slow
query traces.

## 1.3.0

* Breaking Change: Added a timeout parameter to the `Application.Shutdown` method.

## 1.2.0

* Added support for instrumenting short-lived processes:
  * The new `Application.Shutdown` method allows applications to report
    data to New Relic without waiting a full minute.
  * The new `Application.WaitForConnection` method allows your process to
    defer instrumentation until the application is connected and ready to
    gather data.
  * Full documentation here: [application.go](application.go)
  * Example short-lived process: [examples/short-lived-process/main.go](examples/short-lived-process/main.go)

* Error metrics are no longer created when `ErrorCollector.Enabled = false`.

* Added support for [github.com/mgutz/logxi](github.com/mgutz/logxi).  See
  [_integrations/nrlogxi/v1/nrlogxi.go](_integrations/nrlogxi/v1/nrlogxi.go).

* Fixed bug where Transaction Trace thresholds based upon Apdex were not being
  applied to background transactions.

## 1.1.0

* Added support for Transaction Traces.

* Stack trace filenames have been shortened: Any thing preceding the first
  `/src/` is now removed.

## 1.0.0

* Removed `BetaToken` from the `Config` structure.

* Breaking Datastore Change:  `datastore` package contents moved to top level
  `newrelic` package.  `datastore.MySQL` has become `newrelic.DatastoreMySQL`.

* Breaking Attributes Change:  `attributes` package contents moved to top
  level `newrelic` package.  `attributes.ResponseCode` has become
  `newrelic.AttributeResponseCode`.  Some attribute name constants have been
  shortened.

* Added "runtime.NumCPU" to the environment tab.  Thanks sergeylanzman for the
  contribution.

* Prefixed the environment tab values "Compiler", "GOARCH", "GOOS", and
  "Version" with "runtime.".

## 0.8.0

* Breaking Segments API Changes:  The segments API has been rewritten with the
  goal of being easier to use and to avoid nil Transaction checks.  See:

  * [segments.go](segments.go)
  * [examples/server/main.go](examples/server/main.go)
  * [GUIDE.md#segments](GUIDE.md#segments)

* Updated LICENSE.txt with contribution information.

## 0.7.1

* Fixed a bug causing the `Config` to fail to serialize into JSON when the
  `Transport` field was populated.

## 0.7.0

* Eliminated `api`, `version`, and `log` packages.  `Version`, `Config`,
  `Application`, and `Transaction` now live in the top level `newrelic` package.
  If you imported the  `attributes` or `datastore` packages then you will need
  to remove `api` from the import path.

* Breaking Logging Changes

Logging is no longer controlled though a single global.  Instead, logging is
configured on a per-application basis with the new `Config.Logger` field.  The
logger is an interface described in [log.go](log.go).  See
[GUIDE.md#logging](GUIDE.md#logging).

## 0.6.1

* No longer create "GC/System/Pauses" metric if no GC pauses happened.

## 0.6.0

* Introduced beta token to support our beta program.

* Rename `Config.Development` to `Config.Enabled` (and change boolean
  direction).

* Fixed a bug where exclusive time could be incorrect if segments were not
  ended.

* Fix unit tests broken in 1.6.

* In `Config.Enabled = false` mode, the license must be the proper length or empty.

* Added runtime statistics for CPU/memory usage, garbage collection, and number
  of goroutines.

## 0.5.0

* Added segment timing methods to `Transaction`.  These methods must only be
  used in a single goroutine.

* The license length check will not be performed in `Development` mode.

* Rename `SetLogFile` to `SetFile` to reduce redundancy.

* Added `DebugEnabled` logging guard to reduce overhead.

* `Transaction` now implements an `Ignore` method which will prevent
  any of the transaction's data from being recorded.

* `Transaction` now implements a subset of the interfaces
  `http.CloseNotifier`, `http.Flusher`, `http.Hijacker`, and `io.ReaderFrom`
  to match the behavior of its wrapped `http.ResponseWriter`.

* Changed project name from `go-sdk` to `go-agent`.

## 0.4.0

* Queue time support added: if the inbound request contains an
`"X-Request-Start"` or `"X-Queue-Start"` header with a unix timestamp, the
agent will report queue time metrics.  Queue time will appear on the
application overview chart.  The timestamp may fractional seconds,
milliseconds, or microseconds: the agent will deduce the correct units.
