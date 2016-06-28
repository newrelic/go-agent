# Go Agent Guide

## Description

The New Relic Go Agent allows you to monitor your Go applications with New
Relic.  It helps you track transactions, outbound requests, database calls, and
other parts of your Go application's behavior while automatically providing a
running overview of garbage collection events, goroutine activity, and memory
use.

## Installation

To install, run `go get github.com/newrelic/go-agent`, and import the
`github.com/newrelic/go-agent` package within your application.

## Config and Application

* [config.go](api/config.go)
* [application.go](api/application.go)

In your `main` function or in an `init` block:

```go
config := newrelic.NewConfig("Your Application Name", "__YOUR_NEW_RELIC_LICENSE_KEY__")
app, err := newrelic.NewApplication(config)
```

Your application will appear in the New Relic UI with a page showing goroutine,
GC, memory, and CPU metrics.

If you are working in a development environment or running unit tests then you
can prevent the Go Agent from spawning goroutines or reporting to New Relic
by setting the config's `Development` field.

```go
config := newrelic.NewConfig("Your Application Name", "")
config.Development = true
app, err := newrelic.NewApplication(config)
```

## Transactions

* [transaction.go](api/transaction.go)

Transactions time requests and background tasks.  Transaction should only be
used in a single goroutine.  You must start a new transaction when a new
goroutine is spawned.

Transactions may be created directly using the application's `StartTransaction`
method.

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

Use `WrapHandle` and `WrapHandleFunc` to create transactions for requests
handled by the `http` standard library package.

```go
http.HandleFunc(newrelic.WrapHandleFunc(app, "/users", usersHandler))
```

You may then access the transaction in your handlers using type assertion on the
response writer.

```go
if txn, ok := responseWriter.(newrelic.Transaction); ok {
	txn.SetName("otherName")
}
```

## Segments

* [segments.go](api/segments.go)

Find out where the time in your transactions is being spent!  Each transaction
should only track segments in a single goroutine.

`Transaction` has methods to time external calls, datastore calls, functions,
and arbitrary blocks of code.  
To time a function, add the following line to the
beginning of that function:

```go
defer txn.EndSegment(txn.StartSegment(), "mySegmentName")
```

The `defer` pattern will execute the `txn.StartSegment()` when this line is
encountered and the `EndSegment()` method when this function returns.  More
information can be found on `defer` [here](https://gobyexample.com/defer)

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
"External services" tab.  There are a couple of ways to instrument external
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

```go
txn.AddAttribute("key", "value")
txn.AddAttribute("product", "widget")
txn.AddAttribute("price", 19.99)
txn.AddAttribute("importantCustomer", true)
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
