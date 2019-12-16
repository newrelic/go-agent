# Migration Guide - 3.0

This guide is intended to help with upgrading from version 2.x (`"github.com/newrelic/go-agent"`) to version 3.x (`"github.com/newrelic/go-agent/v3/newrelic"`). This information can also be found on
[our documentation website](https://docs.newrelic.com/docs/agents/go-agent/installation/update-go-agent).

* [List of all changes](#all-changes)
* [Checklist for upgrading](#checklist-for-upgrading)

## All Changes

### Dropped support for Go versions < 1.7

The minimum required Go version to run the New Relic Go Agent is now 1.7.

### Package names

The agent has been placed in a new `/v3` directory, leaving the top level directory with the now deprecated v2 agent. More specifically:
* The `newrelic` package has moved from `"github.com/newrelic/go-agent"` to `"github.com/newrelic/go-agent/v3/newrelic"`. This makes named imports unnecessary.
* The underscore in the `_integrations` directory is removed.  Thus the `"github.com/newrelic/go-agent/_integrations/nrlogrus"` import path becomes `"github.com/newrelic/go-agent/v3/integrations/nrlogrus"`.  Some of the integration packages have had other changes as well:
  * `_integrations/nrawssdk/v1` moves to `v3/integrations/nrawssdk-v1`
  * `_integrations/nrawssdk/v2` moves to `v3/integrations/nrawssdk-v2`
  * `_integrations/nrgin/v1` moves to `v3/integrations/nrgin`
  * `_integrations/nrgorilla/v1` moves to `v3/integrations/nrgorilla`
  * `_integrations/nrlogxi/v1` moves to `v3/integrations/nrlogxi`
  * `_integrations/nrecho` moves to `v3/integrations/nrecho-v3` and a new  `v3/integrations/nrecho-v4` has been added to support Echo version 4.

### Transaction Name Changes

Transaction names created by [`WrapHandle`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#WrapHandle),
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
`WebTransaction/Go/users`.

**As a result of this change, you may need to update your alerts and dashboards.**

### Go modules

We have added go module support. The top level `"github.com/newrelic/go-agent/v3/newrelic"` package now has a `go.mod` file. Separate `go.mod` files are also included with each integration in the integrations directory.

### Configuration

`NewConfig` was removed and the [`NewApplication`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewApplication) signature has changed to:

```go
func NewApplication(opts ...ConfigOption) (*Application, error)
`````

New [`ConfigOption`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigOption) functions are provided to modify the [`Config`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Config). Here's what your Application creation will look like:

```go
app, err := newrelic.NewApplication(
    newrelic.ConfigAppName("My Application"),
    newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
)
```

A complete list of `ConfigOption`s can be found in the [Go Docs](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigOption).

### Config.TransactionTracer

The location of two [`Config`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Config) fields have been moved. The `Config.TransactionTracer.SegmentThreshold` field has moved to `Config.TransactionTracer.Segments.Threshold` and the  `Config.TransactionTracer.StackTraceThreshold` field has moved to to `Config.TransactionTracer.Segments.StackTraceThreshold`.

###  Remove API error return values

The following method signatures have changed to no longer return an error; instead the error is logged to the agent logs.

```go
func (txn *Transaction) End() {...}
func (txn *Transaction) Ignore() {...}
func (txn *Transaction) SetName(name string) {...}
func (txn *Transaction) NoticeError(err error) {...}
func (txn *Transaction) AddAttribute(key string, value interface{}) {...}
func (txn *Transaction) SetWebRequestHTTP(r *http.Request) {...}
func (txn *Transaction) SetWebRequest(r *WebRequest) {...}
func (txn *Transaction) AcceptDistributedTracePayload(t TransportType, payload interface{}) {...}
func (txn *Transaction) BrowserTimingHeader() *BrowserTimingHeader {...}
func (s *Segment) End() {...}
func (s *DatastoreSegment) End() {...}
func (s *ExternalSegment) End() {...}
func (s *MessageProducerSegment) End() {...}
func (app *Application) RecordCustomEvent(eventType string, params map[string]interface{}) {...}
func (app *Application) RecordCustomMetric(name string, value float64) {...}
```

### `Application.StartTransaction` signature change

The signature of [`Application.StartTransaction`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Application.StartTransaction)
has changed to no longer take a `http.ResponseWriter` or `*http.Request`. The new signature just takes a string for
the transaction name:

```go
func (app *Application) StartTransaction(name string) *Transaction
```

If you previously had code that used all three parameters, such as:

```go
var writer http.ResponseWriter
var req *http.Request
txn := h.App.StartTransaction("server-txn", writer, req)
```

After the upgrade, it should look like this:

```go
var writer http.ResponseWriter
var req *http.Request
txn := h.App.StartTransaction("server-txn")
writer = txn.SetWebResponse(writer)
txn.SetWebRequestHTTP(req)
```

### Application and Transaction

[`Application`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Application) and [`Transaction`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction) have changed from interfaces to structs. All methods on these types have pointer receivers. Methods on these types are now nil-safe. References to these types in your code will need a pointer added. See the [checklist](#checklist-for-upgrading) for examples.

### Renamed attributes

Two attributes have been renamed.  The old names will still be reported, but are deprecated and will be removed entirely in a future release.

| Old (deprecated) attribute   | New attribute                |
|------------------------------|------------------------------|
| `httpResponseCode`           | `http.statusCode`            |
| `request.headers.User-Agent` | `request.headers.userAgent`  |

Since in v3.0 both the deprecated and the new attribute are being reported, if you have configured your application to ignore one or both of these attributes, such as with `Config.Attributes.Exclude`, you will now need to specify both the deprecated and the new attribute name in your configuration.

### RecordPanics configuration option

This version introduces a new configuration option, `Config.ErrorCollector.RecordPanics`. This configuration controls whether or not a deferred `Transaction.End` will attempt to recover panics, record them as errors, and then re-panic them.  By default, this is set to `false`.  Previous versions of the agent always recovered panics, i.e. a default of `true`.

### New config option for getting data from environment variables

Along with the new format for configuring an application, there is now an option to populate the configuration from environment variables. The full list of environment variables that are considered are included in the [Go Docs](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigFromEnvironment). The new configuration function is used as follows:

```go
app, err := newrelic.NewApplication(newrelic.ConfigFromEnvironment())
```

### `Transaction` no longer implements `http.ResponseWriter`.

As mentioned above, the [`Application.StartTransaction`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Application.StartTransaction) no longer takes a `http.ResponseWriter` or `http.Request`; instead, after you start the transaction, you can set the `ResponseWriter` by calling [`Transaction.SetWebResponse`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.SetWebResponse):

```go
txn := h.App.StartTransaction("server-txn")
writer = txn.SetWebResponse(writer)
```

The [`Transaction.SetWebResponse`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.SetWebResponse) method now returns a replacement `http.ResponseWriter` that implements the combination of `http.CloseNotifier`, `http.Flusher`, `http.Hijacker`, and `io.ReaderFrom` implemented by the input `http.ResponseWriter`.

### The `WebRequest` type has changed from an interface to a struct

[`WebRequest`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#WebRequest) has changed from an interface to a struct, which can be created via code like this:

```go
webReq := newrelic.WebRequest{
	Header:    hdrs,
	URL:       url,
	Method:    method,
	Transport: newrelic.TransportHTTP,
}
```

The [`Transaction.SetWebRequest`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.SetWebRequest) method takes one of these structs.

### `SetWebRequestHTTP` method added

In addition to the [`Transaction.SetWebRequest`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.SetWebRequest) method discussed in the section above, we have added a method [`Transaction.SetWebRequestHTTP`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.SetWebRequestHTTP) that takes an `*http.Request` and sets the appropriate fields.

As described in the [earlier section](#applicationstarttransaction-signature-change), this can be used in your code as part of the signature change of [`Application.StartTransaction`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Application.StartTransaction):

```go
var writer http.ResponseWriter
var req *http.Request
txn := h.App.StartTransaction("server-txn")
writer = txn.SetWebResponse(writer)
txn.SetWebRequestHTTP(req)
```

### `NewRoundTripper`

The transaction parameter to [`NewRoundTripper`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewRoundTripper) has been removed. The function signature is now:

```go
func NewRoundTripper(t http.RoundTripper) http.RoundTripper
```

[`NewRoundTripper`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewRoundTripper) will look for a transaction in the request's context using [`FromContext`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#FromContext).

### Distributed Trace methods

When manually creating or accepting Distributed Tracing payloads, the method signatures have changed.

This [`Transaction.InsertDistributedTraceHeaders`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.InsertDistributedTraceHeaders) method will insert the Distributed Tracing headers into the `http.Header` object passed as a parameter:

```go
func (txn *Transaction) InsertDistributedTraceHeaders(hdrs http.Header)
```

This [`Transaction.AcceptDistributedTraceHeaders`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.AcceptDistributedTraceHeaders) method takes a [`TransportType`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#TransportType) and an `http.Header` object that contains Distributed Tracing header(s) and links this transaction to other transactions specified in the headers:

```go
func (txn *Transaction) AcceptDistributedTraceHeaders(t TransportType, hdrs http.Header)
```

Additionally, the `DistributedTracePayload` struct is no longer needed and has been removed from the agent's API. Instead, distributed tracing information is passed around as key/value pairs in the `http.Header` object.

### Several functions marked as deprecated

The functions [`StartSegmentNow`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#StartSegmentNow) and [`StartSegment`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#StartSegment) have been marked as deprecated.  The preferred new method of starting a segment have moved to [`Transaction.StartSegmentNow`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.StartSegmentNow) and [`Transaction.StartSegment`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.StartSegment) respectively.

```go
// DEPRECATED:
startTime := newrelic.StartSegmentNow(txn)
// and
sgmt := newrelic.StartSegment(txn, "segment1")
```

```go
// NEW, PREFERRED WAY:
startTime := txn.StartSegmentNow()
// and
sgmt := txn.StartSegment("segment1")
```

Additionally the functions [`NewLogger`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewLogger) and [`NewDebugLogger`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewDebugLogger) have been marked as deprecated.  The preferred new method of configuring agent logging is using the [`ConfigInfoLogger`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigInfoLogger) and [`ConfigDebugLogger`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigDebugLogger) `ConfigOptions` respectively.

  ```go
  // DEPRECATED:
  app, err := newrelic.NewApplication(
      ...
      func(cfg *newrelic.Config) {
          cfg.Logger = newrelic.NewLogger(os.Stdout)
      }
  )

  // or

  app, err := newrelic.NewApplication(
      ...
      func(cfg *newrelic.Config) {
          cfg.Logger = newrelic.NewDebugLogger(os.Stdout)
      }
  )
  ```

  ```go
  // NEW, PREFERRED WAY:
  app, err := newrelic.NewApplication(
      ...
      newrelic.ConfigInfoLogger(os.Stdout),
  )

  // or

  app, err := newrelic.NewApplication(
      ...
      newrelic.ConfigDebugLogger(os.Stdout),
  )
  ```

### Removed optional interfaces from error

The interfaces `ErrorAttributer`, `ErrorClasser`, `StackTracer` are no longer exported. Thus, if you have any code checking to ensure that your custom type fulfills these interfaces, that code will no longer work. Example:

```go
// This will no longer compile.
type MyErrorType struct{}
var _ newrelic.ErrorAttributer = MyErrorType{}
```

### Changed Distributed Tracing Constant

`DistributedTracePayloadHeader` has been changed to `DistributedTraceNewRelicHeader`.

### `TransportType`

[`TransportType`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#TransportType) type is changed from a struct to a string.

## Checklist for upgrading

- [ ] Ensure your Go version is at least 1.7 (older versions are no longer supported).

- [ ] Update imports. The v3.x agent now lives at "github.com/newrelic/go-agent/v3/newrelic" and no longer requires a named import.

  From:

  ```go
  import newrelic "github.com/newrelic/go-agent"
  ```

  To:

  ```go
  import "github.com/newrelic/go-agent/v3/newrelic"
  ```

  Additionally, if you are using any integrations, they too have moved. Each has its own version which matches the version of the 3rd party package it supports.

  From:

  ```go
  import "github.com/newrelic/go-agent/_integrations/nrlogrus"
  ```

  To:

  ```go
  import "github.com/newrelic/go-agent/v3/integrations/nrlogrus"
  ```

- [ ] Update how you configure your application. The [`NewApplication`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewApplication) function now accepts [`ConfigOption`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigOption)s a list of which can be [found here](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigOption). If a [`ConfigOption`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigOption) is not available for your setting, create one yourself!

  From:

  ```go
  cfg := newrelic.NewConfig("appName", "__license__")
  cfg.CrossApplicationTracer.Enabled = false
  cfg.CustomInsightsEvents.Enabled = false
  cfg.ErrorCollector.IgnoreStatusCodes = []int{404, 418}
  cfg.DatastoreTracer.SlowQuery.Threshold = 3
  cfg.DistributedTracer.Enabled = true
  cfg.TransactionTracer.Threshold.Duration = 2
  cfg.TransactionTracer.Threshold.IsApdexFailing = false
  app, err := newrelic.NewApplication(cfg)
  ````

  To:

  ```go
  app, err := newrelic.NewApplication(
      newrelic.ConfigAppName("appName"),
      newrelic.ConfigLicense("__license__"),
      func(cfg *newrelic.Config) {
          cfg.CrossApplicationTracer.Enabled = false
          cfg.CustomInsightsEvents.Enabled = false
          cfg.ErrorCollector.IgnoreStatusCodes = []int{404, 418}
          cfg.DatastoreTracer.SlowQuery.Threshold = 3
          cfg.DistributedTracer.Enabled = true
          cfg.TransactionTracer.Threshold.Duration = 2
          cfg.TransactionTracer.Threshold.IsApdexFailing = false
      },
  )
  ```

  You can use [`ConfigFromEnvironment`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigFromEnvironment) to provide configuration from environment variables:

  ```go
  app, err := newrelic.NewApplication(newrelic.ConfigFromEnvironment())
  ```

- [ ] Update the Transaction Tracer configuration. Change the fields for the two changed configuration options.

  | Old Config Field                               | New Config Field                                        |
  |------------------------------------------------|---------------------------------------------------------|
  | `Config.TransactionTracer.SegmentThreshold`    | `Config.TransactionTracer.Segments.Threshold`           |
  | `Config.TransactionTracer.StackTraceThreshold` | `Config.TransactionTracer.Segments.StackTraceThreshold` |

- [ ] If you choose, set the `Config.ErrorCollector.RecordPanics` configuration option. This is a new configuration option that controls whether or not a deferred [`Transaction.End`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.End) will attempt to recover panics, record them as errors, and then re-panic them. Previously, the agent acted as though this option was set to `true`; with the new configuration it defaults to `false`.  If you wish to maintain the old agent behavior with regards to panics, be sure to set this to `true`.

- [ ] Update code to use the new [`Application.StartTransaction`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Application.StartTransaction) signature.

  From:

  ```go
  txn := app.StartTransaction("name", nil, nil)

  // or

  // writer is an http.ResponseWriter
  // req is an *http.Request
  txn := app.StartTransaction("name", writer, req)
  txn.WriteHeader(500)
  ```

  To, respectively:

  ```go
  txn := app.StartTransaction("name")

  // or

  // writer is an http.ResponseWriter
  // req is an *http.Request
  txn:= app.StartTransaction("name")
  writer = txn.SetWebResponse(writer)
  txn.SetWebRequestHTTP(req)
  writer.WriteHeader(500)
  ```

  Notice too that the [`Transaction`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction) no longer fulfills the `http.ResponseWriter` interface. Instead, the writer returned from [`Transaction.SetWebResponse`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.SetWebResponse) should be used.

- [ ] Update code to no longer expect an error returned from these updated methods. Instead, check the agent logs for errors by using one of the [`ConfigLogger`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigLogger), [`ConfigInfoLogger`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigInfoLogger), or [`ConfigDebugLogger`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigDebugLogger) configuration options.

  ```go
  func (txn *Transaction) End() {...}
  func (txn *Transaction) Ignore() {...}
  func (txn *Transaction) SetName(name string) {...}
  func (txn *Transaction) NoticeError(err error) {...}
  func (txn *Transaction) AddAttribute(key string, value interface{}) {...}
  func (txn *Transaction) SetWebRequestHTTP(r *http.Request) {...}
  func (txn *Transaction) SetWebRequest(r *WebRequest) {...}
  func (txn *Transaction) AcceptDistributedTracePayload(t TransportType, payload interface{}) {...}
  func (txn *Transaction) BrowserTimingHeader() *BrowserTimingHeader {...}
  func (s *Segment) End() {...}
  func (s *DatastoreSegment) End() {...}
  func (s *ExternalSegment) End() {...}
  func (s *MessageProducerSegment) End() {...}
  func (app *Application) RecordCustomEvent(eventType string, params map[string]interface{}) {...}
  func (app *Application) RecordCustomMetric(name string, value float64) {...}
  ```

- [ ] Update uses of [`Application`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Application) and [`Transaction`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction) to be pointers, instead of direct references.

  From:

  ```go
  func doSomething(txn newrelic.Transaction) {...}
  func instrumentSomething(app newrelic.Application, h http.Handler, name string) {...}
  ```

  To:

  ```go
  func doSomething(txn *newrelic.Transaction) {...}
  func instrumentSomething(app *newrelic.Application, h http.Handler,  name string) {...}
  ```

- [ ] If you are using a [`WebRequest`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#WebRequest) type, it has changed from an interface to a struct. You can use it as follows:

  ```go
  wr := newrelic.WebRequest{
      Header:    r.Header,
      URL:       r.URL,
      Method:    r.Method,
      Transport: newrelic.TransportHTTP,
  }
  txn.SetWebRequest(wr)
  ```

- [ ] Remove the `Transaction` parameter from the [`NewRoundTripper`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewRoundTripper), and instead ensure that the transaction is available via the request's context, using [`RequestWithTransactionContext`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#RequestWithTransactionContext).

  From:

  ```go
  client := &http.Client{}
  client.Transport = newrelic.NewRoundTripper(txn, client.Transport)
  req, _ := http.NewRequest("GET", "http://example.com", nil)
  client.Do(req)
  ```

  To:

  ```go
  client := &http.Client{}
  client.Transport = newrelic.NewRoundTripper(client.Transport)
  req, _ := http.NewRequest("GET", "http://example.com", nil)
  req = newrelic.RequestWithTransactionContext(req, txn)
  client.Do(req)
  ```

- [ ] Update any usage of Distributed Tracing accept/create functions. The method for creating a distributed trace payload has changed to [`Transaction.InsertDistributedTraceHeaders`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.InsertDistributedTraceHeaders). Instead of returning a payload, it now accepts an `http.Header` and inserts the header(s) directly into it.

  From:

  ```go
  hdrs := http.Header{}
  payload := txn.CreateDistributedTracePayload()
  hdrs.Set(newrelic.DistributedTracePayloadHeader, payload.Text())
  ```

  To:

  ```go
  hdrs := http.Header{}
  txn.InsertDistributedTraceHeaders(hdrs)
  ```

  Similarly, the method for accepting distributed trace payloads has changed to [`Transaction.AcceptDistributedTraceHeaders`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.AcceptDistributedTraceHeaders). Instead of taking an interface representing the payload value, it now accepts an `http.Header` representing both the keys and values.

  From:

  ```go
  hdrs := request.Headers()
  payload := hdrs.Get(newrelic.DistributedTracePayloadHeader)
  txn.AcceptDistributedTracePayload(newrelic.TransportKafka, payload)
  ```

  To:

  ```go
  hdrs := request.Headers()
  txn.AcceptDistributedTraceHeaders(newrelic.TransportKafka, hdrs)
  ```

  Additionally, the `DistributedTracePayload` struct is no longer needed and has been removed from the agent's API. Instead, distributed tracing information is passed around as key/value pairs in the `http.Header` object. You should remove all references to `DistributedTracePayload` in your code.

- [ ] Change `newrelic.DistributedTracePayloadHeader` to `newrelic.DistributedTraceNewRelicHeader`.

- [ ] If you have configured your application to **ignore** either attribute described [here](#renamed-attributes), you will now need to specify both the deprecated and the new attribute name in your configuration.

  Configuration options where these codes might be used:

  ```go
  Config.TransactionEvents.Attributes
  Config.ErrorCollector.Attributes
  Config.TransactionTracer.Attributes
  Config.TransactionTrace.Segments.Attributes
  Config.BrowserMonitoring.Attributes
  Config.SpanEvents.Attributes
  Config.Attributes
  ```

  From old configuration example:

  ```go
  config.ErrorCollector.Attributes.Exclude = []string{
      "httpResponseCode",
      "request.headers.User-Agent",
  }
  // or
  config.ErrorCollector.Attributes.Exclude = []string{
      newrelic.AttributeResponseCode,
      newrelic.AttributeRequestUserAgent,
  }
  ```

  To:

  ```go
  config.ErrorCollector.Attributes.Exclude = []string{
      "http.statusCode",
      "httpResponseCode",
      "request.headers.userAgent",
      "request.headers.User-Agent",
  }
  // or
  config.ErrorCollector.Attributes.Exclude = []string{
      newrelic.AttributeResponseCode,
      newrelic.AttributeResponseCodeDeprecated,
      newrelic.AttributeRequestUserAgent,
      newrelic.AttributeRequestUserAgentDeprecated,
  }
  ```

- [ ] Update alerts and dashboards with new transaction names:

  Transaction names created by
  [`WrapHandle`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#WrapHandle),
  [`WrapHandleFunc`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#WrapHandleFunc),
  [nrecho-v3](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrecho-v3),
  [nrecho-v4](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrecho-v4),
  [nrgorilla](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgorilla), and
  [nrgin](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgin) now
  include the HTTP method.  Thus the transaction name `WebTransaction/Go/users` becomes `WebTransaction/Go/GET /users`.

- [ ] Not required for upgrade, but recommended: update your usages of the now deprecated [`StartSegment`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#StartSegment) and [`StartSegmentNow`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#StartSegmentNow) to use the methods on the transaction: [`Transaction.StartSegment`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.StartSegment) and [`Transaction.StartSegmentNow`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Trnasaction.StartSEgmentNow) respectively. This step is optional but highly recommended.

  From:

  ```go
  startTime := newrelic.StartSegmentNow(txn)
  // and
  sgmt := newrelic.StartSegment(txn, "segment1")
  ```

  To:

  ```go
  startTime := txn.StartSegmentNow()
  // and
  sgmt := txn.StartSegment("segment1")
  ```

- [ ] Not required for upgrade, but recommended: update your usages of the now deprecated [`NewLogger`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewLogger) and [`NewDebugLogger`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewDebugLogger).  Instead use the new `ConfigOption`s [`ConfigInfoLogger`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigInfoLogger) and [`ConfigDebugLogger`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigDebugLogger) respectively.

  From:

  ```go
  app, err := newrelic.NewApplication(
      ...
      func(cfg *newrelic.Config) {
          cfg.Logger = newrelic.NewLogger(os.Stdout)
      }
  )

  // or

  app, err := newrelic.NewApplication(
      ...
      func(cfg *newrelic.Config) {
          cfg.Logger = newrelic.NewDebugLogger(os.Stdout)
      }
  )
  ```

  To:

  ```go
  app, err := newrelic.NewApplication(
      ...
      newrelic.ConfigInfoLogger(os.Stdout),
  )

  // or

  app, err := newrelic.NewApplication(
      ...
      newrelic.ConfigDebugLogger(os.Stdout),
  )
  ```
