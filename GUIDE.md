# New Relic Go Agent Guide

* [Upgrading](#upgrading)
* [Installation](#installation)
* [Full list of `Config` options and `Application` settings](#full-list-of-config-options-and-application-settings)
* [Logging](#logging)
* [Transactions](#transactions)
* [Segments](#segments)
  * [Datastore Segments](#datastore-segments)
  * [External Segments](#external-segments)
  * [Message Producer Segments](#message-producer-segments)
* [Attributes](#attributes)
* [Tracing](#tracing)
  * [Distributed Tracing](#distributed-tracing)
  * [Cross-Application Tracing](#cross-application-tracing)
  * [Tracing instrumentation](#tracing-instrumentation)
    * [Getting Tracing Instrumentation Out-of-the-Box](#getting-tracing-instrumentation-out-of-the-box)
    * [Manually Implementing Distributed Tracing](#manually-implementing-distributed-tracing)
* [Distributed Tracing](#distributed-tracing)
* [Custom Metrics](#custom-metrics)
* [Custom Events](#custom-events)
* [Request Queuing](#request-queuing)
* [Error Reporting](#error-reporting)
  * [NoticeError](#noticeerror)
  * [Panics](#panics)
  * [Error Response Codes](#error-response-codes)
* [Naming Transactions and Metrics](#naming-transactions-and-metrics)
* [Browser](#browser)
* [For More Help](#for-more-help)

## Upgrading

This guide documents version 3.x of the agent which resides in package
`"github.com/newrelic/go-agent/v3/newrelic"`.
If you have already been using version 2.X of the agent and are upgrading to
version 3.0, see our [Migration Guide](MIGRATION.md) for details.

## Installation

(Also see [GETTING_STARTED](https://github.com/newrelic/go-agent/blob/master/GETTING_STARTED.md) if you are using the Go agent for the first time).

In order to install the New Relic Go agent, you need a New Relic license key. 
Then, installing the Go Agent is the same as installing any other Go library.  The
simplest way is to run:

```
go get github.com/newrelic/go-agent
```

Then import the package in your application:
```go
import "github.com/newrelic/go-agent/v3/newrelic"
```

Initialize the New Relic Go agent by adding the following `Config` options and `Application` settings in the `main` function or in an `init` block:

```go
app, err := newrelic.NewApplication(
    newrelic.ConfigAppName("Your Application Name"),
    newrelic.ConfigLicense("__YOUR_NEW_RELIC_LICENSE_KEY__"),
)
```

This will allow you to see Go runtime information.

Now, add instrumentation to your Go application to get additional performance data:
* Import any of our [integration packages](https://github.com/newrelic/go-agent#integrations) for out-of-the box support for many popular Go web 
frameworks and libraries. 
* [Instrument Transactions](#transactions)
* [Use Distributed Tracing](#distributed-tracing)
* [(Optional) Instrument Segments](#segments) for an extra level of timing detail
  * External segments are needed for Distributed Tracing
* Read through the rest of this GUIDE for more instrumentation

Compile and deploy your application.

Find your application in the New Relic UI.  Click on it to see application performance, 
including the Go runtime page that shows information about goroutine counts, garbage 
collection, memory, and CPU usage. Data should show up within 5 minutes.


If you are working in a development environment or running unit tests, you may
not want the Go Agent to spawn goroutines or report to New Relic.  You're in
luck!  Use the `ConfigEnabled` function to disable the agent.  This makes the license key
optional.

```go
app, err := newrelic.NewApplication(
    newrelic.ConfigAppName("Your Application Name"),
    newrelic.ConfigEnabled(false),
)
```



## Full list of `Config` options and `Application` settings

* [Config godoc](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Config)
* [Application godoc](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Application)



## Logging

The agent's logging system is designed to be easily extensible.  By default, no
logging will occur.  To enable logging, use the following config functions
with an [io.Writer](https://godoc.org/github.com/pkg/io/#Writer):
[ConfigInfoLogger](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/#ConfigInfoLogger),
which logs at info level, and
[ConfigDebugLogger](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/#ConfigDebugLogger)
which logs at debug level.

To log at debug level to standard out, set:

```go
app, err := newrelic.NewApplication(
    newrelic.ConfigAppName("Your Application Name"),
    newrelic.ConfigLicense("__YOUR_NEW_RELIC_LICENSE_KEY__"),
    // Add debug logging:
    newrelic.ConfigDebugLogger(os.Stdout),
)
```

To log at info level to a file, set:

```go
w, err := os.OpenFile("my_log_file", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
if nil == err {
    app, _ := newrelic.NewApplication(
        newrelic.ConfigAppName("Your Application Name"),
        newrelic.ConfigLicense("__YOUR_NEW_RELIC_LICENSE_KEY__"),
        newrelic.ConfigInfoLogger(w),
    )
}
```

Popular logging libraries `logrus`, `logxi` and `zap` are supported by
integration packages:

* [v3/integrations/nrlogrus](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrlogrus/)
* [v3/integrations/nrlogxi](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrlogxi/)
* [v3/integrations/nrzap](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrzap/)

## Transactions

* [Transaction godoc](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction)
* [Naming Transactions](#naming-transactions-and-metrics)
* [More info on Transactions](https://docs.newrelic.com/docs/apm/applications-menu/monitoring/transactions-page)

Transactions time requests and background tasks.  The simplest way to create
transactions is to use `Application.StartTransaction` and `Transaction.End`.

```go
txn := app.StartTransaction("transactionName")
defer txn.End()
```

If you are instrumenting a background transaction, this is all that is needed. If, however,
you are instrumenting a web transaction, you will want to use the
 `SetWebRequestHTTP` and `SetWebResponse` methods as well.

[SetWebRequestHTTP](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/#Transaction.SetWebRequestHTTP)
marks the transaction as a web transaction. If the [http.Request](https://godoc.org/net/http#Request)
is non-nil, `SetWebRequestHTTP` will additionally collect details on request
attributes, url, and method. If headers are present, the agent will look for a
distributed tracing header.

If you want to mark a transaction as a web transaction, but don't have access
 to an `http.Request`, you can use the [SetWebRequest](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/#Transaction.SetWebRequest)
method, using a manually constructed [WebRequest](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/#WebRequest)
object.

[SetWebResponse](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/#Transaction.SetWebResponse)
allows the Transaction to instrument response code and response headers. Pass in
your [http.ResponseWriter](https://godoc.org/net/http#ResponseWriter) as a
parameter, and then use the return value of this method in place of the input
parameter in your instrumentation.

Here is an example using both methods:

```go
func (h *handler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
    txn := h.App.StartTransaction("transactionName")
    defer txn.End()
    // This marks the transaction as a web transactions and collects details on
    // the request attributes
    txn.SetWebRequestHTTP(req)
    // This collects details on response code and headers. Use the returned
    // Writer from here on.
    writer = txn.SetWebResponse(writer)
    // ... handler code continues here using the new writer
}
```

The transaction has helpful methods like `NoticeError` and `SetName`.
See more in [godocs](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction).

If you are using [`http.ServeMux`](https://golang.org/pkg/net/http/#ServeMux),
use `WrapHandle` and `WrapHandleFunc`.  These wrappers automatically start and
end transactions with the request and response writer.

```go
http.HandleFunc(newrelic.WrapHandleFunc(app, "/users", usersHandler))
```

To access the transaction in your handler, we recommend getting it from the
 Request context:

```go
func myHandler(w http.ResponseWriter, r *http.Request) {
    txn := newrelic.FromContext(r.Context())
    // ... handler code here
}
```

To monitor a transaction across multiple goroutines, use
`Transaction.NewGoroutine()`. The `NewGoroutine` method returns a new reference
to the `Transaction`, which is required by each segment-creating goroutine. It
does not matter if you call `NewGoroutine` before or after the other goroutine
starts.

```go
go func(txn newrelic.Transaction) {
	defer txn.StartSegment("async").End()
	time.Sleep(100 * time.Millisecond)
}(txn.NewGoroutine())
```

## Segments

Find out where the time in your transactions is being spent!

`Segment` is used to instrument functions, methods, and blocks of code. A
segment begins when its `StartTime` field is populated, and finishes when its
`End` method is called.

```go
segment := newrelic.Segment{}
segment.Name = "mySegmentName"
segment.StartTime = txn.StartSegmentNow()
// ... code you want to time here ...
segment.End()
```

`Transaction.StartSegment` is a convenient helper.  It creates a segment and
 starts it:

```go
segment := txn.StartSegment("mySegmentName")
// ... code you want to time here ...
segment.End()
```

Timing a function is easy using `StartSegment` and `defer`.  Just add the
following line to the beginning of that function:

```go
defer txn.StartSegment("mySegmentName").End()
```

Segments may be nested.  The segment being ended must be the most recently
started segment.

```go
s1 := txn.StartSegment("outerSegment")
s2 := txn.StartSegment("innerSegment")
// s2 must be ended before s1
s2.End()
s1.End()
```

A zero value segment may safely be ended.  Therefore, the following code
is safe even if the conditional fails:

```go
var s newrelic.Segment
txn := newrelic.FromContext(ctx)
if shouldDoSomething() {
    s.StartTime = txn.StartSegmentNow(),
}
// ... code you wish to time here ...
s.End()
```

### Datastore Segments

Datastore segments appear in the transaction "Breakdown table" and in the
"Databases" page.

* [More info on Databases page](https://docs.newrelic.com/docs/apm/applications-menu/monitoring/databases-slow-queries-page)

Datastore segments are instrumented using
[DatastoreSegment](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/#DatastoreSegment).
Just like basic segments, datastore segments begin when the `StartTime` field
is populated and finish when the `End` method is called.  Here is an example:

```go
s := newrelic.DatastoreSegment{
    // Product is the datastore type.  See the constants in
    // https://github.com/newrelic/go-agent/blob/master/v3/newrelic/datastore.go.  Product
    // is one of the fields primarily responsible for the grouping of Datastore
    // metrics.
    Product: newrelic.DatastoreMySQL,
    // Collection is the table or group being operated upon in the datastore,
    // e.g. "users_table".  This becomes the db.collection attribute on Span
    // events and Transaction Trace segments.  Collection is one of the fields
    // primarily responsible for the grouping of Datastore metrics.
    Collection: "users_table",
    // Operation is the relevant action, e.g. "SELECT" or "GET".  Operation is
    // one of the fields primarily responsible for the grouping of Datastore
    // metrics.
    Operation: "SELECT",
}
s.StartTime = txn.StartSegmentNow()
// ... make the datastore call
s.End()
```

This may be combined into two lines when instrumenting a datastore call
that spans an entire function call:

```go
s := newrelic.DatastoreSegment{
    StartTime:  txn.StartSegmentNow(),
    Product:    newrelic.DatastoreMySQL,
    Collection: "my_table",
    Operation:  "SELECT",
}
defer s.End()
```

If you are using the standard library's
[database/sql](https://golang.org/pkg/database/sql/) package with
[MySQL](https://github.com/go-sql-driver/mysql),
[PostgreSQL](https://github.com/lib/pq), or
[SQLite](https://github.com/mattn/go-sqlite3) then you can avoid creating
DatastoreSegments by hand by using an integration package:

* [v3/integrations/nrpq](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrpq)
* [v3/integrations/nrmysql](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrmysql)
* [v3/integrations/nrsqlite3](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsqlite3)
* [v3/integrations/nrmongo](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrmongo)

### External Segments

External segments appear in the transaction "Breakdown table" and in the
"External services" page. Version 1.11.0 of the Go Agent adds support for
cross-application tracing (CAT), which will result in external segments also
appearing in the "Service maps" page and being linked in transaction traces when
both sides of the request have traces. Version 2.1.0 of the Go Agent adds
support for distributed tracing, which lets you see the path a request takes as
it travels through distributed APM apps.

* [More info on External Services page](https://docs.newrelic.com/docs/apm/applications-menu/monitoring/external-services-page)
* [More info on Cross-Application Tracing](https://docs.newrelic.com/docs/apm/transactions/cross-application-traces/introduction-cross-application-traces)
* [More info on Distributed Tracing](https://docs.newrelic.com/docs/apm/distributed-tracing/getting-started/introduction-distributed-tracing)

External segments are instrumented using `ExternalSegment`. There are three
ways to use this functionality:

1. Using `StartExternalSegment` to create an `ExternalSegment` before the
   request is sent, and then calling `ExternalSegment.End` when the external
   request is complete.

   For CAT support to operate, an `http.Request` must be provided to
   `StartExternalSegment`, and the `ExternalSegment.Response` field must be set
   before `ExternalSegment.End` is called or deferred.

   For example:

    ```go
    func external(txn newrelic.Transaction, req *http.Request) (*http.Response, error) {
      s := txn.StartExternalSegment(req)
      response, err := http.DefaultClient.Do(req)
      s.Response = response
      s.End()
      return response, err
    }
    ```

    If the transaction is `nil` then `StartExternalSegment` will look for a
    transaction in the request's context using
    [FromContext](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/#FromContext).

2. Using `NewRoundTripper` to get a
   [`http.RoundTripper`](https://golang.org/pkg/net/http/#RoundTripper) that
   will automatically instrument all requests made via
   [`http.Client`](https://golang.org/pkg/net/http/#Client) instances that use
   that round tripper as their `Transport`. This option results in CAT support,
   provided the Go Agent is version 1.11.0, and in distributed tracing support,
   provided the Go Agent is version 2.1.0.  `NewRoundTripper` will look for a
   transaction in the request's context using
   [FromContext](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/#FromContext).

   For example:

    ```go
    client := &http.Client{}
    client.Transport = newrelic.NewRoundTripper(client.Transport)
    request, _ := http.NewRequest("GET", "http://example.com", nil)
    // Put transaction in the request's context:
    request = newrelic.RequestWithTransactionContext(request, txn)
    resp, err := client.Do(request)
    ```

3. Directly creating an `ExternalSegment` via a struct literal with an explicit
   `URL` or `Request`, and then calling `ExternalSegment.End`. This option does
   not support CAT, and may be removed or changed in a future major version of
   the Go Agent. As a result, we suggest using one of the other options above
   wherever possible.

   For example:

    ```go
    func external(txn newrelic.Transaction, url string) (*http.Response, error) {
      es := newrelic.ExternalSegment{
        StartTime: txn.StartSegmentNow(),
        URL:   url,
      }
      defer es.End()

      return http.Get(url)
    }
    ```

### Message Producer Segments

Message producer segments appear in the transaction "Breakdown table".

Message producer segments are instrumented using
[MessageProducerSegment](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/#MessageProducerSegment).
Just like basic segments, message producer segments begin when the `StartTime`
field is populated and finish when the `End` method is called.  Here is an
example:

```go
s := newrelic.MessageProducerSegment{
    // Library is the name of the library instrumented.
    Library: "RabbitMQ",
    // DestinationType is the destination type.
    DestinationType: newrelic.MessageExchange,
    // DestinationName is the name of your queue or topic.
    DestinationName: "myExchange",
    // DestinationTemporary must be set to true if destination is temporary
    // to improve metric grouping.
    DestinationTemporary: false,
}
s.StartTime = txn.StartSegmentNow()
// ... add message to queue here
s.End()
```

This may be combined into two lines when instrumenting a message producer
call that spans an entire function call:

```go
s := newrelic.MessageProducerSegment{
    StartTime:            txn.StartSegmentNow(),
    Library:              "RabbitMQ",
    DestinationType:      newrelic.MessageExchange,
    DestinationName:      "myExchange",
    DestinationTemporary: false,
}
defer s.End()
```

## Attributes

Attributes add context to errors and allow you to filter performance data
in Insights.

You may add them using the `Transaction.AddAttribute` and `Segment.AddAttribute`
methods.

```go
txn.AddAttribute("key", "value")
txn.AddAttribute("product", "widget")
txn.AddAttribute("price", 19.99)
txn.AddAttribute("importantCustomer", true)

seg.AddAttribute("count", 14)
```

* [More info on Custom Attributes](https://docs.newrelic.com/docs/insights/new-relic-insights/decorating-events/insights-custom-attributes)

Some attributes are recorded automatically.  These are called agent attributes.
They are listed here:

* [newrelic package constants](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#pkg-constants)

To disable one of these agents attributes, for example `AttributeHostDisplayName`,
modify the config like this:

```go
app, err := newrelic.NewApplication(
    newrelic.ConfigAppName("Your Application Name"),    
    newrelic.ConfigLicense("__YOUR_NEW_RELIC_LICENSE_KEY__"),
    func(cfg *newrelic.Config) {
        config.Attributes.Exclude = append(config.Attributes.Exclude, newrelic.AttributeHostDisplayName)
    }
)
```

* [More info on Agent Attributes](https://docs.newrelic.com/docs/agents/manage-apm-agents/agent-metrics/agent-attributes)

## Tracing

New Relic's [distributed tracing](https://docs.newrelic.com/docs/apm/distributed-tracing/getting-started/introduction-distributed-tracing)
is the next generation of the previous cross-application tracing feature.
Compared to cross-application tracing, distributed tracing gives more detail
about cross-service activity and provides more complete end-to-end
visibility.  This section discusses distributed tracing and cross-application
tracing in turn.

### Distributed Tracing

New Relic's [distributed
tracing](https://docs.newrelic.com/docs/apm/distributed-tracing/getting-started/introduction-distributed-tracing)
feature lets you see the path that a request takes as it travels through distributed APM
apps, which is vital for applications implementing a service-oriented or
microservices architecture. Support for distributed tracing was added in
version 2.1.0 of the Go Agent.

The config's `DistributedTracer.Enabled` field has to be set. When true, the
agent will add distributed tracing headers in outbound requests, and scan
incoming requests for distributed tracing headers. Distributed tracing will
override cross-application tracing.

```go
app, err := newrelic.NewApplication(
    newrelic.ConfigAppName("Your Application Name"),
    newrelic.ConfigLicense("__YOUR_NEW_RELIC_LICENSE_KEY__"),
    newrelic.ConfigDistributedTracerEnabled(true),  
)
```

### Cross-Application Tracing [Deprecated]

New Relic's
[cross-application tracing](https://docs.newrelic.com/docs/apm/transactions/cross-application-traces/introduction-cross-application-traces)
feature, or CAT for short, links transactions between applications in APM to
help identify performance problems within your service-oriented architecture.
Support for CAT was added in version 1.11.0 of the Go Agent. We recommend using
[Distributed Tracing](#distributed-tracing) as the most recent, complete feature.

As CAT uses HTTP headers to track requests across applications, the Go Agent
needs to be able to access and modify request and response headers both for
incoming and outgoing requests.

### Tracing Instrumentation

Both distributed tracing and cross-application tracing work by propagating
[header information](https://docs.newrelic.com/docs/apm/distributed-tracing/getting-started/how-new-relic-distributed-tracing-works#headers)
from service to service in a request path. In many scenarios, the Go Agent offers tracing instrumentation
out-of-the-box, for both distributed tracing and cross-application tracing. For other scenarios customers may implement
distributed tracing based on the examples provided in this guide.

#### Getting Tracing Instrumentation Out-of-the-Box

The Go Agent automatically creates and propagates tracing header information
for each of the following scenarios:

For server applications:

1. Using `WrapHandle` or `WrapHandleFunc` to instrument a server that
   uses [`http.ServeMux`](https://golang.org/pkg/net/http/#ServeMux)
   ([Example](v3/examples/server/main.go)).

2. Using any of the Go Agent's HTTP integrations, which are listed [here
](README.md#integrations).

3. Using another framework or [`http.Server`](https://golang.org/pkg/net/http/#Server) while ensuring that:

      1. After calling `StartTransaction`, make sure to call `Transaction.SetWebRequest`
      and `Transaction.SetWebResponse` on the transaction, and
      2. the `http.ResponseWriter` that is returned from `Transaction.SetWebResponse`
      is used instead of calling `WriteHeader` directly on the original response
      writer, as described in the [transactions section of this guide](#transactions)
       ([Example](v3/examples/server-http/main.go)).

For client applications:

1. Using `NewRoundTripper`, as described in the
   [external segments section of this guide](#external-segments)
   ([Example](v3/examples/client-round-tripper/main.go)).

2. Using the call `StartExternalSegment` and providing an `http.Request`, as
   described in the [external segments section of this guide](#external-segments)
   ([Example](v3/examples/client/main.go)).

#### Manually Implementing Distributed Tracing

Consider [manual instrumentation](https://docs.newrelic.com/docs/apm/distributed-tracing/enable-configure/enable-distributed-tracing#agent-apis)
for services not instrumented automatically by the Go Agent. In such scenarios, the
calling service has to insert the appropriate header(s) into the request headers:

```go
var h http.Headers
callingTxn.InsertDistributedTraceHeaders(h)
```

These headers have to be added to the call to the destination service, which in
turn invokes the call for accepting the headers:

```go
var h http.Headers
calledTxn.AcceptDistributedTraceHeaders(newrelic.TransportOther, h)
```

A complete example can be found
[here](v3/examples/custom-instrumentation/main.go).


## Custom Metrics

* [More info on Custom Metrics](https://docs.newrelic.com/docs/agents/go-agent/instrumentation/create-custom-metrics-go)

You may [create custom metrics](https://docs.newrelic.com/docs/agents/manage-apm-agents/agent-data/collect-custom-metrics)
via the `RecordCustomMetric` method.

```go
app.RecordCustomMetric(
    "CustomMetricName", // Name of your metric
    132,                // Value
)
```

**Note:** The Go Agent will automatically prepend the metric name you pass to
`RecordCustomMetric` (`"CustomMetricName"` above) with the string `Custom/`.
This means the above code would produce a metric named
`Custom/CustomMetricName`.  You'll also want to read over the
[Naming Transactions and Metrics](#naming-transactions-and-metrics) section
below for advice on coming up with appropriate metric names.

## Custom Events

You may track arbitrary events using custom Insights events.

```go
app.RecordCustomEvent("MyEventType", map[string]interface{}{
    "myString": "hello",
    "myFloat":  0.603,
    "myInt":    123,
    "myBool":   true,
})
```

## Request Queuing

If you are running a load balancer or reverse web proxy then you may configure
it to add a `X-Queue-Start` header with a Unix timestamp.  This will create a
band on the application overview chart showing queue time.

* [More info on Request Queuing](https://docs.newrelic.com/docs/apm/applications-menu/features/request-queuing-tracking-front-end-time)

## Error Reporting

The Go Agent captures errors in three different ways:

1. [the Transaction.NoticeError method](#noticeerror)
2. [panics recovered in defer Transaction.End](#panics)
3. [error response status codes recorded with Transaction.WriteHeader](#error-response-codes)

### NoticeError

You may track errors using the `Transaction.NoticeError` method.  The easiest
way to get started with `NoticeError` is to use errors based on
[Go's standard error interface](https://blog.golang.org/error-handling-and-go).

```go
txn.NoticeError(errors.New("my error message"))
```

`NoticeError` will work with *any* sort of object that implements Go's standard
error type interface -- not just `errorStrings` created via `errors.New`.  

If you're interested in sending more than an error *message* to New Relic, the
Go Agent also offers a `newrelic.Error` struct.

```go
txn.NoticeError(newrelic.Error{
    Message: "my error message",
    Class:   "IdentifierForError",
    Attributes: map[string]interface{}{
        "important_number": 97232,
        "relevant_string":  "zap",
    },
})
```

Using the `newrelic.Error` struct requires you to manually marshal your error
data into the `Message`, `Class`, and `Attributes` fields.  However, there's two
**advantages** to using the `newrelic.Error` struct.

First, by setting an error `Class`, New Relic will be able to aggregate errors
in the *Error Analytics* section of APM.  Second, the `Attributes` field allows
you to send through key/value pairs with additional error debugging information
(also exposed in the *Error Analytics* section of APM).

### Panics

When the Transaction is ended using `defer`, the Transaction will optionally recover any
panic that occurs, record it as an error, and re-throw it. You can enable this feature by
setting the configuration:

```go
app, err := newrelic.NewApplication(
    newrelic.ConfigAppName("Your Application Name"),
    newrelic.ConfigLicense("__YOUR_NEW_RELIC_LICENSE_KEY__"),
    func(cfg *newrelic.Config) {
        cfg.ErrorCollector.RecordPanics = true
    }
)
```

As a result of this configuration, panics may appear to be originating from `Transaction.End`.

```go
func unstableTask(app newrelic.Application) {
    txn := app.StartTransaction("unstableTask", nil, nil)
    defer txn.End()

    // This panic will be recorded as an error.
    panic("something went wrong")
}
```

### Error Response Codes

Setting the WebResponse on the transaction using `Transaction.SetWebResponse`
returns an
[http.ResponseWriter](https://golang.org/pkg/net/http/#ResponseWriter), and you
can use that returned ResponseWriter to call `WriteHeader` to record the response
status code.  The transaction will record an error if the status code is
at or above 400 or strictly below 100 and not in the ignored status codes
configuration list.  The ignored status codes list is configured by the
`Config.ErrorCollector.IgnoreStatusCodes` field or within the New Relic UI
if your application has server side configuration enabled.

As a result, using `Transaction.NoticeError` in situations where your code is
returning an erroneous status code may result in redundant errors.
`NoticeError` is not affected by the ignored status codes configuration list.

## Naming Transactions and Metrics

You'll want to think carefully about how you name your transactions and custom
metrics.  If your program creates too many unique names, you may end up with a
[Metric Grouping Issue (or MGI)](https://docs.newrelic.com/docs/agents/manage-apm-agents/troubleshooting/metric-grouping-issues).

MGIs occur when the granularity of names is too fine, resulting in hundreds or
thousands of uniquely identified metrics and transactions.  One common cause of
MGIs is relying on the full URL name for metric naming in web transactions.  A
few major code paths may generate many different full URL paths to unique
documents, articles, page, etc. If the unique element of the URL path is
included in the metric name, each of these common paths will have its own unique
metric name.

## Browser

To enable support for
[New Relic Browser](https://docs.newrelic.com/docs/browser), your HTML pages
must include a JavaScript snippet that will load the Browser agent and
configure it with the correct application name. This snippet is available via
the `Transaction.BrowserTimingHeader` method.  Include the byte slice returned
by `Transaction.BrowserTimingHeader().WithTags()` as early as possible in the
`<head>` section of your HTML after any `<meta charset>` tags.

```go
func indexHandler(w http.ResponseWriter, req *http.Request) {
    io.WriteString(w, "<html><head>")
    // The New Relic browser javascript should be placed as high in the
    // HTML as possible.  We suggest including it immediately after the
    // opening <head> tag and any <meta charset> tags.
    txn := newrelic.FromContext(req.Context())
    hdr, err := txn.BrowserTimingHeader()
    if nil != err {
        log.Printf("unable to create browser timing header: %v", err)
    }
    // BrowserTimingHeader() will always return a header whose methods can
    // be safely called.
    if js := hdr.WithTags(); js != nil {
        w.Write(js)
    }
    io.WriteString(w, "</head><body>browser header page</body></html>")
}
```


## For More Help

There's a variety of places online to learn more about the Go Agent.

[The New Relic docs site](https://docs.newrelic.com/docs/agents/go-agent/get-started/introduction-new-relic-go)
contains a number of useful code samples and more context about how to use the Go Agent.

[New Relic's discussion forums](https://discuss.newrelic.com) have a dedicated
public forum [for the Go Agent](https://discuss.newrelic.com/c/support-products-agents/go-agent).

When in doubt, [the New Relic support site](https://support.newrelic.com/) is
the best place to get started troubleshooting an agent issue.
