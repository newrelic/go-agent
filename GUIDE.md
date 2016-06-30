# New Relic Go Agent Guide

* [Config and Application](#config-and-application)
* [Transactions](#transactions)
* [Segments](#segments)
  * [Datastore Segments](#datastore-segments)
  * [External Segments](#external-segments)
* [Attributes](#attributes)
* [Request Queuing](#request-queuing)

## Installation

Installing the Go Agent is the same as installing any other Go library.  The
simplest way is to run:

```
go get github.com/newrelic/go-agent
```

Then import the `github.com/newrelic/go-agent` package in your application.

## Config and Application

* [config.go](api/config.go)
* [application.go](api/application.go)

In your `main` function or in an `init` block:

```go
config := newrelic.NewConfig("Your Application Name", "__YOUR_NEW_RELIC_LICENSE_KEY__")
app, err := newrelic.NewApplication(config)
```

Find your application in the New Relic UI.  Click on it to see the Go runtime
tab that shows information about goroutine counts, garbage collection, memory,
and CPU usage.

If you are working in a development environment or running unit tests, you may
not want the Go Agent to spawn goroutines or report to New Relic.  You're in
luck!  Set the config's `Development` field to true.  This makes the license key
optional.

```go
config := newrelic.NewConfig("Your Application Name", "")
config.Development = true
app, err := newrelic.NewApplication(config)
```

## Transactions

* [transaction.go](api/transaction.go)
* [More info on Transactions](https://docs.newrelic.com/docs/apm/applications-menu/monitoring/transactions-page)

Transactions time requests and background tasks.  Each transaction should only
be used in a single goroutine.  Start a new transaction when you spawn a new
goroutine.

The simplest way to create transactions is to use
`Application.StartTransaction` and `Transaction.End`.

```go
txn := app.StartTransaction("transactionName", responseWriter, request)
defer txn.End()
```

The response writer and request parameters are optional.  Leave them `nil` to
instrument a background task.

```go
txn := app.StartTransaction("backgroundTask", nil, nil)
defer txn.End()
```

The transaction has helpful methods like `NoticeError` and `SetName`.
See more in [transaction.go](api/transaction.go).

If you are using the `http` standard library package, use `WrapHandle` and
`WrapHandleFunc`.  These wrappers automatically start and end transactions with
the request and response writer.  See [instrumentation.go](instrumentation.go).

```go
http.HandleFunc(newrelic.WrapHandleFunc(app, "/users", usersHandler))
```

To access the transaction in your handler, use type assertion on the response
writer passed to the handler.

```go
func myHandler(w http.ResponseWriter, r *http.Request) {
	if txn, ok := w.(newrelic.Transaction); ok {
		txn.NoticeError(errors.New("my error message"))
	}
}
```

## Segments

* [segments.go](api/segments.go)

Find out where the time in your transactions is being spent!  Each transaction
should only track segments in a single goroutine.

`Transaction` has methods to time external calls, datastore calls, functions,
and arbitrary blocks of code.  

To time a function, add the following line to the beginning of that function:

```go
defer txn.EndSegment(txn.StartSegment(), "mySegmentName")
```

The `defer` pattern will execute the `txn.StartSegment()` when this line is
encountered and the `EndSegment()` method when this function returns.  More
information can be found on `defer` [here](https://gobyexample.com/defer).

To time a block of code, use the following pattern:

```go
token := txn.StartSegment()
// ... code you want to time here ...
txn.EndSegment(token, "mySegmentName")
```

Segments may be nested.  The segment being ended must be the most recently
started segment.

```go
token1 := txn.StartSegment()
token2 := txn.StartSegment()
// token2 must be ended before token1
txn.EndSegment(token2, "innerSegment")
txn.EndSegment(token1, "outerSegment")
```

### Datastore Segments

Datastore segments appear in the transaction "Breakdown table" and in the
"Databases" tab.  They are finished using `EndDatastore`.  This requires
importing the `api/datastore` subpackage.

* [datastore.go](api/datastore/datastore.go)
* [More info on Databases tab](https://docs.newrelic.com/docs/apm/applications-menu/monitoring/databases-slow-queries-page)

```go
defer txn.EndDatastore(txn.StartSegment(), datastore.Segment{
	// Product is the datastore type.
	// See the constants in api/datastore/datastore.go.
	Product: datastore.MySQL,
	// Collection is the table or group.
	Collection: "my_table",
	// Operation is the relevant action, e.g. "SELECT" or "GET".
	Operation: "SELECT",
})
```

### External Segments

External segments appear in the transaction "Breakdown table" and in the
"External services" tab.  

* [More info on External Services tab](https://docs.newrelic.com/docs/apm/applications-menu/monitoring/external-services-page)

There are a couple of ways to instrument external
segments.  The simplest way is to use `EndExternal`:

```go
func externalCall(url string, txn newrelic.Transaction) (*http.Response, error) {
	defer txn.EndExternal(txn.StartSegment(), url)

	return http.Get(url)
}
```

The functions `PrepareRequest` and `EndRequest` are recommended since they will
be used in the future to trace activity between distributed applications using
headers.

```go
token := txn.StartSegment()
txn.PrepareRequest(token, request)
response, err := client.Do(request)
txn.EndRequest(token, request, response)
```

`NewRoundTripper` is a helper built on top of `PrepareRequest` and `EndRequest`.
This round tripper **must** be used the same goroutine as the transaction.

```go
client := &http.Client{}
client.Transport = newrelic.NewRoundTripper(txn, nil)
resp, err := client.Get("http://example.com/")
```

## Attributes

Attributes add context to errors and allow you to filter performance data
in Insights.

* [More info on Insights attributes](https://docs.newrelic.com/docs/insights/new-relic-insights/decorating-events/insights-custom-attributes)

```go
txn.AddAttribute("key", "value")
txn.AddAttribute("product", "widget")
txn.AddAttribute("price", 19.99)
txn.AddAttribute("importantCustomer", true)
```

Some attributes are recorded automatically.  They are listed here:

* [api/attributes.go](api/attributes.go)

To disable one of these attributes, `RequestHeadersUserAgent` for example,
modify the config like this:

```go
// needs import "github.com/newrelic/go-agent/api/attributes"
config.Attributes.Exclude = append(config.Attributes.Exclude,
	attributes.RequestHeadersUserAgent)
```

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

[more info](docs.newrelic.com/docs/apm/applications-menu/features/request-queuing-tracking-front-end-time)
